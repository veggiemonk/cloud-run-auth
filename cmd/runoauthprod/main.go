package main

import (
	"context"
	"encoding/base64"
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := MustParse()

	// Decode encryption key (base64 → raw bytes).
	encKey, err := base64.StdEncoding.DecodeString(cfg.SessionEncryptionKey)
	if err != nil || len(encKey) != 32 {
		slog.Error("SESSION_ENCRYPTION_KEY must be a base64-encoded 32-byte key")
		os.Exit(1)
	}

	// Initialize Firestore session store with encryption.
	ctx := context.Background()
	store, err := session.NewStore(ctx, cfg.ProjectID, cfg.FirestoreDB, encKey)
	if err != nil {
		slog.Error("failed to initialize session store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Create OAuth config.
	oauthCfg := newGoogleConfig(cfg.Google.ClientID, cfg.Google.ClientSecret, cfg.Google.RedirectURL)

	// Cookie config (production vs dev).
	cookies := NewCookieConfig(cfg.KRevision)

	// CSRF protection.
	csrf, err := middleware.NewCSRF(cfg.CSRFKey)
	if err != nil {
		slog.Error("failed to initialize CSRF", "error", err)
		os.Exit(1)
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
		slog.Error("failed to create static sub-filesystem", "error", err)
		os.Exit(1)
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
	handler := shared.LoggingMiddleware(logger,
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
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// newGoogleConfig creates an OAuth2 config. We inline this rather than importing
// oauth.NewGoogleConfig to avoid pulling in the in-memory session store dependency.
func newGoogleConfig(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"openid",
			"email",
			"profile",
			"https://www.googleapis.com/auth/cloud-platform.read-only",
		},
		Endpoint: google.Endpoint,
	}
}
