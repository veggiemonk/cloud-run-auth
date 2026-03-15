package main

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/veggiemonk/cloud-run-iap/internal/handler"
	"github.com/veggiemonk/cloud-run-iap/internal/iap"
	"github.com/veggiemonk/cloud-run-iap/internal/reqlog"
)

//go:embed static
var staticFiles embed.FS

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	verifier := iap.NewVerifier()

	// Warn at startup if IAP_AUDIENCE is not configured on Cloud Run.
	if verifier.ExpectedAudience() == "" {
		if os.Getenv("K_SERVICE") != "" {
			slog.Error("IAP_AUDIENCE environment variable is not set — JWT verification is disabled. Set IAP_AUDIENCE to enable signature verification.")
		} else {
			slog.Warn("IAP_AUDIENCE not set — running in local/dev mode, JWT verification disabled")
		}
	}

	buf := reqlog.NewBuffer()

	mux := http.NewServeMux()

	// Static files (no auth required).
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		slog.Error("failed to create static sub-filesystem", "error", err)
		os.Exit(1)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Health check (no auth required).
	mux.Handle("GET /healthz", handler.Healthz())

	// Protected routes — wrapped with IAP auth middleware.
	protected := http.NewServeMux()
	protected.Handle("GET /", handler.Dashboard(verifier))
	protected.Handle("GET /headers", handler.Headers())
	protected.Handle("GET /jwt", handler.JWT(verifier))
	protected.Handle("GET /audience", handler.Audience(verifier))
	protected.Handle("POST /audience", handler.Audience(verifier))
	protected.Handle("GET /log", handler.Log(buf))
	protected.Handle("GET /diagnostic", handler.Diagnostic(verifier))
	mux.Handle("/", requireIAP(verifier, protected))

	// Wrap with middleware.
	wrapped := loggingMiddleware(logger, requestLogMiddleware(buf, mux))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("starting server", "port", port)
	if err := http.ListenAndServe(":"+port, wrapped); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// loggingMiddleware logs each request using structured logging.
func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}

		next.ServeHTTP(sw, r)

		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// requestLogMiddleware records each request into the ring buffer.
func requestLogMiddleware(buf *reqlog.Buffer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		det := iap.Detect(r)

		buf.Add(reqlog.Entry{
			Timestamp:  time.Now(),
			Method:     r.Method,
			Path:       r.URL.Path,
			Email:      det.Email,
			HasIAP:     det.HasJWT,
			RemoteAddr: r.RemoteAddr,
		})

		next.ServeHTTP(w, r)
	})
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
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
