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
	"github.com/veggiemonk/cloud-run-auth/internal/handler/iaphandler"
	"github.com/veggiemonk/cloud-run-auth/internal/iap"
	"github.com/veggiemonk/cloud-run-auth/internal/log"
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
	cfg, help, err := config.LoadIAP()
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

	verifier := iap.NewVerifier(cfg.Audience)

	// Warn at startup if IAP_AUDIENCE is not configured on Cloud Run.
	if verifier.ExpectedAudience() == "" {
		if cfg.KService != "" {
			slog.Error(
				"IAP_AUDIENCE environment variable is not set — JWT verification is disabled. Set IAP_AUDIENCE to enable signature verification.",
			)
		} else {
			slog.Warn("IAP_AUDIENCE not set — running in local/dev mode, JWT verification disabled")
		}
	}

	buf := reqlog.NewBuffer()

	mux := http.NewServeMux()

	// Static files (no auth required).
	staticFS, err := fs.Sub(assets.StaticFiles, "static")
	if err != nil {
		return fmt.Errorf("static sub-filesystem: %w", err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Health check (no auth required).
	mux.Handle("GET /healthz", iaphandler.Healthz())

	// Protected routes — wrapped with IAP auth middleware.
	protected := http.NewServeMux()
	protected.Handle("GET /", iaphandler.Dashboard(verifier))
	protected.Handle("GET /headers", iaphandler.Headers())
	protected.Handle("GET /jwt", iaphandler.JWT(verifier))
	protected.Handle("GET /audience", iaphandler.Audience(verifier))
	protected.Handle("POST /audience", iaphandler.Audience(verifier))
	protected.Handle("GET /log", iaphandler.Log(buf))
	protected.Handle("GET /diagnostic", iaphandler.Diagnostic(verifier))
	mux.Handle("/", requireIAP(verifier, protected))

	// IAP-specific email extractor for request log middleware.
	iapEmailExtractor := func(r *http.Request) string {
		det := iap.Detect(r)
		return det.Email
	}

	// Wrap with middleware.
	wrapped := shared.LoggingMiddleware(logger, shared.RequestLogMiddleware(buf, iapEmailExtractor, "iap", mux))

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

// requireIAP rejects requests that don't have a valid IAP JWT.
// When IAP_AUDIENCE is configured, the JWT signature is verified.
// When running locally (no IAP_AUDIENCE), only the presence of the JWT header is checked.
func requireIAP(verifier *iap.Verifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		det := iap.Detect(r)
		if !det.HasJWT {
			http.Error(w, "Unauthorized: no IAP JWT present", http.StatusUnauthorized)
			return
		}

		if verifier.ExpectedAudience() != "" {
			result := verifier.Verify(r.Context(), det.RawJWT)
			if !result.Valid {
				http.Error(w, "Forbidden: invalid IAP JWT", http.StatusForbidden)
				return
			}
		}

		// Store detection result in context so handlers don't re-detect.
		next.ServeHTTP(w, iap.WithDetectionResult(r, det))
	})
}
