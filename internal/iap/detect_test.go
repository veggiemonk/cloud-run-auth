package iap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/veggiemonk/cloud-run-auth/internal/is"
)

func TestDetect_NoHeaders(t *testing.T) {
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	det := Detect(r)

	is.True(t, !det.HasJWT)
	is.True(t, !det.HasEmailHeader)
	is.True(t, !det.HasIDHeader)
	is.Equal(t, det.Warning, "", "")
}

func TestDetect_JWTOnly(t *testing.T) {
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	r.Header.Set(HeaderJWTAssertion, "header.payload.signature")
	det := Detect(r)

	is.True(t, det.HasJWT)
	is.Equal(t, det.RawJWT, "header.payload.signature", "")
	is.True(t, !det.HasEmailHeader)
	is.True(t, !det.HasIDHeader)
	is.Equal(t, det.Warning, "", "")
}

func TestDetect_EmailAndIDWithoutJWT(t *testing.T) {
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	r.Header.Set(HeaderAuthenticatedEmail, "accounts.google.com:user@example.com")
	r.Header.Set(HeaderAuthenticatedID, "accounts.google.com:12345")
	det := Detect(r)

	is.True(t, !det.HasJWT)
	is.True(t, det.HasEmailHeader)
	is.True(t, det.HasIDHeader)
	is.Equal(t, det.Email, "user@example.com", "email prefix stripped")
	is.Equal(t, det.UserID, "12345", "user ID prefix stripped")
	is.True(t, det.Warning != "")
}

func TestDetect_AllHeaders(t *testing.T) {
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	r.Header.Set(HeaderJWTAssertion, "a.b.c")
	r.Header.Set(HeaderAuthenticatedEmail, "accounts.google.com:user@example.com")
	r.Header.Set(HeaderAuthenticatedID, "accounts.google.com:12345")
	det := Detect(r)

	is.True(t, det.HasJWT && det.HasEmailHeader && det.HasIDHeader)
	is.Equal(t, det.Warning, "", "no warning when JWT present")
}

func TestDetect_EmailWithoutPrefix(t *testing.T) {
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	r.Header.Set(HeaderJWTAssertion, "a.b.c")
	r.Header.Set(HeaderAuthenticatedEmail, "user@example.com")
	det := Detect(r)

	is.Equal(t, det.Email, "user@example.com", "email unchanged without prefix")
}

func TestDetectionResult_RawJWTExcludedFromJSON(t *testing.T) {
	det := DetectionResult{
		HasJWT: true,
		RawJWT: "secret-token",
		Email:  "user@example.com",
	}

	b, err := json.Marshal(det)
	is.NoErr(t, err)

	var m map[string]any
	is.NoErr(t, json.Unmarshal(b, &m))

	_, ok := m["raw_jwt"]
	is.True(t, !ok)
}
