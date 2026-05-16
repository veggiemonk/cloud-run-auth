// Package iaphandler owns the HTTP handlers for the runiap diagnostic
// UI: dashboard, headers, JWT, audience, log, diagnostic, and healthz.
// Each file is one route, mapping IAP detection results (internal/iap)
// to the templ view models (internal/components/iapui).
//
// Exists so cmd/runiap stays a thin wiring layer — handler construction
// (route → handler factory) is the only thing main() needs to know
// about; the package owns rendering, JSON negotiation, and the
// auth-context lookup conventions for IAP routes.
package iaphandler

import "net/http"

// Healthz returns a simple health check handler.
func Healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}
