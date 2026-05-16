// Package oauthhandler owns the HTTP handlers for the runoauth /
// runoauthprod dashboard UI: dashboard, token, gcp, diagnostic, and
// healthz. Each file is one route, mapping the authenticated UserInfo
// in context (internal/oauth) to a templ view model (oauthui).
//
// Exists so the cmd/runoauth* binaries stay thin — handler construction
// is the only thing main() needs from here; the package owns rendering,
// JSON negotiation, and the auth-context lookup conventions for OAuth
// routes.
package oauthhandler

import "net/http"

// Healthz returns a simple health check handler.
func Healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}
