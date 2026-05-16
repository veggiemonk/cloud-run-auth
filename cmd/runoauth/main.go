package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/veggiemonk/cloud-run-auth/internal/assets"
	"github.com/veggiemonk/cloud-run-auth/internal/config"
	"github.com/veggiemonk/cloud-run-auth/internal/handler/oauthhandler"
	"github.com/veggiemonk/cloud-run-auth/internal/log"
	"github.com/veggiemonk/cloud-run-auth/internal/oauth"
	"github.com/veggiemonk/cloud-run-auth/internal/shared"
	"github.com/veggiemonk/cloud-run-auth/internal/shared/reqlog"
	"github.com/veggiemonk/cloud-run-auth/internal/version"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, help, err := config.LoadOAuth()
	if err != nil {
		return err
	}
	if help != "" {
		fmt.Println(help)
		return nil
	}

	logger := log.New(os.Stdout, cfg.LogOptions())
	slog.SetDefault(logger)

	if dump, err := config.String(&cfg); err == nil {
		slog.Info("startup", "version", version.Get(), "config", dump)
	}

	oauthCfg := oauth.NewGoogleConfig(cfg.ClientID, cfg.ClientSecret, cfg.RedirectURL)
	sessions := oauth.NewSessionStore(oauthCfg)
	sessions.StartCleanup(5 * time.Minute)
	buf := reqlog.NewBuffer()

	mux := http.NewServeMux()

	// Static files.
	staticFS, err := fs.Sub(assets.StaticFiles, "static")
	if err != nil {
		return fmt.Errorf("static sub-filesystem: %w", err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Health check (no auth required).
	mux.Handle("GET /healthz", oauthhandler.Healthz())

	// Auth routes (public).
	mux.Handle("GET /auth/login", oauth.LoginHandler(oauthCfg, sessions))
	mux.Handle("GET /auth/callback", oauth.CallbackHandler(oauthCfg, sessions))
	mux.Handle("GET /auth/logout", oauth.LogoutHandler(sessions))

	// Protected routes.
	protected := http.NewServeMux()
	protected.Handle("GET /", oauthhandler.Dashboard())
	protected.Handle("GET /token", oauthhandler.Token())
	protected.Handle("GET /gcp", oauthhandler.GCPExplorer())
	protected.Handle("GET /diagnostic", oauthhandler.Diagnostic())
	mux.Handle("/", oauth.RequireAuth(sessions, protected))

	// OAuth-specific email extractor reads directly from session store via cookie,
	// so it works regardless of middleware ordering.
	oauthEmailExtractor := func(r *http.Request) string {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			return ""
		}
		session := sessions.Get(cookie.Value)
		if session == nil {
			return ""
		}
		return session.Email
	}

	wrapped := shared.LoggingMiddleware(logger, shared.RequestLogMiddleware(buf, oauthEmailExtractor, "oauth", mux))

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           wrapped,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.Info("starting server", "port", cfg.Port)
	if err := srv.ListenAndServe(); err != nil {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}
