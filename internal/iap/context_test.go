package iap

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/veggiemonk/cloud-run-auth/internal/is"
)

func TestDetectionResultFromContext_WithStoredResult(t *testing.T) {
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

	det := DetectionResult{
		HasJWT: true,
		Email:  "stored@example.com",
		RawJWT: "stored-token",
	}

	r = WithDetectionResult(r, det)
	got := DetectionResultFromContext(r)

	is.Equal(t, got.Email, "stored@example.com", "")
	is.Equal(t, got.RawJWT, "stored-token", "")
}

func TestDetectionResultFromContext_FallsBackToDetect(t *testing.T) {
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	r.Header.Set(HeaderJWTAssertion, "a.b.c")
	r.Header.Set(HeaderAuthenticatedEmail, "accounts.google.com:fallback@example.com")

	// No WithDetectionResult call — should fall back to Detect(r).
	got := DetectionResultFromContext(r)

	is.True(t, got.HasJWT)
	is.Equal(t, got.Email, "fallback@example.com", "")
}
