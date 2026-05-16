// Package middleware owns the production HTTP defenses bolted onto the
// runoauthprod binary: security response headers (CSP, X-Frame-Options,
// X-Content-Type-Options, Referrer-Policy), per-IP and per-user token
// bucket rate limiters, HMAC-derived CSRF tokens, and the body-size
// limiter.
//
// Exists as the hardening seam — runoauth (the dev binary) does not
// compose any of this, so the rules in here are exactly what changes
// when traffic is exposed to the public internet. Touching these
// constants is a security review, not an ops knob.
package middleware

import "net/http"

// Security header values.
const (
	ContentSecurityPolicy = "default-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data: https://*.googleusercontent.com; " +
		"script-src 'self'; " +
		"frame-ancestors 'none'"

	XContentTypeOptionsValue = "nosniff"
	XFrameOptionsValue       = "DENY"
	ReferrerPolicyValue      = "strict-origin-when-cross-origin"
)

// SecurityHeaders adds security headers to all responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", ContentSecurityPolicy)
		h.Set("X-Content-Type-Options", XContentTypeOptionsValue)
		h.Set("X-Frame-Options", XFrameOptionsValue)
		h.Set("Referrer-Policy", ReferrerPolicyValue)
		next.ServeHTTP(w, r)
	})
}
