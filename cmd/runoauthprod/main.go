package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/veggiemonk/cloud-run-auth/internal/assets"
	"github.com/veggiemonk/cloud-run-auth/internal/handler/oauthhandler"
	"github.com/veggiemonk/cloud-run-auth/internal/middleware"
	"github.com/veggiemonk/cloud-run-auth/internal/session"
	"github.com/veggiemonk/cloud-run-auth/internal/shared"
	"github.com/veggiemonk/cloud-run-auth/internal/shared/reqlog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := MustParse()

	// Decode encryption key (base64 → raw bytes).
	encKey, err := base64.StdEncoding.DecodeString(cfg.SessionEncryptionKey)
	if err != nil || len(encKey) != 32 {
		return errors.New("SESSION_ENCRYPTION_KEY must be a base64-encoded 32-byte key")
	}

	// Initialize Firestore session store with encryption.
	ctx := context.Background()
	store, err := session.NewStore(ctx, cfg.ProjectID, cfg.FirestoreDB, encKey)
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			slog.Warn("failed to close session store", "error", err)
		}
	}()

	// Create OAuth config.
	oauthCfg, err := newGoogleConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create OAuth config: %w", err)
	}

	// Cookie config (production vs dev).
	cookies := NewCookieConfig(cfg.KRevision)

	// CSRF protection.
	csrf, err := middleware.NewCSRF(cfg.CSRFKey)
	if err != nil {
		return fmt.Errorf("failed to initialize CSRF: %w", err)
	}

	// Auth dependencies.
	deps := &authDeps{
		oauthCfg:      oauthCfg,
		store:         store,
		cookies:       cookies,
		csrf:          csrf,
		allowedDomain: cfg.AllowedDomain,
	}

	// Rate limiters.
	authLimiter := middleware.NewIPRateLimiter(cfg.AuthRateLimitBurst, time.Minute)
	userLimiter := middleware.NewUserRateLimiter(cfg.UserRateLimitBurst, time.Minute, deps.emailFromSession)

	// Request log buffer.
	buf := reqlog.NewBuffer()

	// Routes.
	mux := http.NewServeMux()

	// Static files.
	staticFS, err := fs.Sub(assets.StaticFiles, "static")
	if err != nil {
		return fmt.Errorf("failed to create static sub-filesystem: %w", err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Health check (no auth).
	mux.Handle("GET /healthz", oauthhandler.Healthz())

	// Auth routes (IP rate limited).
	mux.Handle("GET /auth/login", authLimiter.Limit(deps.prodLoginHandler()))
	mux.Handle("GET /auth/callback", authLimiter.Limit(deps.prodCallbackHandler()))

	// Logout (POST-only, requires auth + CSRF).
	logoutChain := deps.requireAuth(
		csrf.RequireCSRF(deps.sessionIDFromCookie)(
			deps.prodLogoutHandler(),
		),
	)
	mux.Handle("POST /auth/logout", logoutChain)

	// Protected routes (auth with refresh + user rate limit + CSRF).
	protected := http.NewServeMux()
	protected.Handle("GET /", oauthhandler.Dashboard())
	protected.Handle("GET /token", oauthhandler.Token())
	protected.Handle("GET /gcp", oauthhandler.GCPExplorer())
	protected.Handle("GET /diagnostic", oauthhandler.Diagnostic())

	protectedChain := deps.requireAuthWithRefresh(
		userLimiter.Limit(
			csrf.RequireCSRF(deps.sessionIDFromCookie)(
				protected,
			),
		),
	)
	mux.Handle("/", protectedChain)

	// Middleware chain (outermost first).
	handler := shared.LoggingMiddleware(
		logger,
		middleware.SecurityHeaders(
			middleware.MaxBodySize(MaxBodyBytes)(
				shared.RequestLogMiddleware(buf, deps.emailFromSession, "oauth", mux),
			),
		),
	)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadTimeout:       ReadTimeout,
		ReadHeaderTimeout: ReadHeaderTimeout,
		IdleTimeout:       IdleTimeout,
	}

	slog.Info("starting production OAuth server", "port", cfg.Port, "secure", cookies.Secure)
	if err := srv.ListenAndServe(); err != nil {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}

// newGoogleConfig creates an OAuth2 config from either GOOGLE_OAUTH_CONFIG JSON
// blob (production) or individual env vars (local dev).
func newGoogleConfig(cfg Config) (*oauth2.Config, error) {
	scopes := []string{
		"openid",
		"email",
		"profile",
		"https://www.googleapis.com/auth/cloud-platform.read-only",
	}

	// Prefer full JSON blob (production path).
	if cfg.Google.OAuthConfig != "" {
		oauthCfg, err := google.ConfigFromJSON([]byte(cfg.Google.OAuthConfig), scopes...)
		if err != nil {
			return nil, fmt.Errorf("parsing GOOGLE_OAUTH_CONFIG: %w", err)
		}
		if cfg.Google.RedirectURL != "" {
			oauthCfg.RedirectURL = cfg.Google.RedirectURL
		}
		return oauthCfg, nil
	}

	// Fallback: individual env vars (local dev).
	if cfg.Google.ClientID == "" || cfg.Google.ClientSecret == "" {
		return nil, errors.New(
			"either GOOGLE_OAUTH_CONFIG or both GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET are required",
		)
	}
	return &oauth2.Config{
		ClientID:     cfg.Google.ClientID,
		ClientSecret: cfg.Google.ClientSecret,
		RedirectURL:  cfg.Google.RedirectURL,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}, nil
}
