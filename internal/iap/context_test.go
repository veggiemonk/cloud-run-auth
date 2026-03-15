package iap

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDetectionResultFromContext_WithStoredResult(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	det := DetectionResult{
		HasJWT: true,
		Email:  "stored@example.com",
		RawJWT: "stored-token",
	}

	r = WithDetectionResult(r, det)
	got := DetectionResultFromContext(r)

	if got.Email != "stored@example.com" {
		t.Errorf("expected stored email, got: %s", got.Email)
	}
	if got.RawJWT != "stored-token" {
		t.Errorf("expected stored RawJWT, got: %s", got.RawJWT)
	}
}

func TestDetectionResultFromContext_FallsBackToDetect(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(HeaderJWTAssertion, "a.b.c")
	r.Header.Set(HeaderAuthenticatedEmail, "accounts.google.com:fallback@example.com")

	// No WithDetectionResult call — should fall back to Detect(r).
	got := DetectionResultFromContext(r)

	if !got.HasJWT {
		t.Error("expected HasJWT=true from fallback Detect")
	}
	if got.Email != "fallback@example.com" {
		t.Errorf("expected fallback email, got: %s", got.Email)
	}
}
