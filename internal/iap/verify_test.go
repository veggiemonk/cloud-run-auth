package iap

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/veggiemonk/cloud-run-auth/internal/is"
)

// makeJWT creates a test JWT with the given header and payload maps.
func makeJWT(t *testing.T, header, payload map[string]any) string {
	t.Helper()
	h, err := json.Marshal(header)
	is.NoErr(t, err)
	p, err := json.Marshal(payload)
	is.NoErr(t, err)
	return base64.RawURLEncoding.EncodeToString(h) + "." +
		base64.RawURLEncoding.EncodeToString(p) + "." +
		base64.RawURLEncoding.EncodeToString([]byte("fake-signature"))
}

func TestDecode_ValidJWT(t *testing.T) {
	v := &Verifier{}
	jwt := makeJWT(
		t,
		map[string]any{"alg": "ES256", "typ": "JWT"},
		map[string]any{
			"iss":   "https://cloud.google.com/iap",
			"sub":   "12345",
			"email": "user@example.com",
			"aud":   "/projects/123/global/backendServices/456",
			"iat":   1700000000.0,
			"exp":   1700003600.0,
		},
	)

	result := v.Decode(jwt)

	is.Equal(t, result.Error, "", "no decode error")
	is.True(t, !result.Valid)
	is.True(t, result.Claims != nil)
	is.Equal(t, result.Claims.Email, "user@example.com", "")
	is.Equal(t, result.Claims.Issuer, "https://cloud.google.com/iap", "")
	is.Equal(t, result.Claims.Subject, "12345", "")
	is.True(t, result.Header != nil)
	is.Equal(t, result.Header["alg"], any("ES256"), "")
	is.True(t, result.SignatureB64 != "")
}

func TestDecode_InvalidFormat(t *testing.T) {
	v := &Verifier{}

	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"no dots", "nodots", "invalid JWT format"},
		{"one dot", "one.dot", "invalid JWT format"},
		{"bad base64 header", "!!!.cGF5bG9hZA.sig", "failed to decode JWT header"},
		{"bad base64 payload", "aGVhZGVy.!!!.sig", "failed to decode JWT payload"},
		{
			"invalid header json",
			base64.RawURLEncoding.EncodeToString([]byte("not json")) + ".cA.sig",
			"failed to parse JWT header JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.Decode(tt.token)
			is.True(t, result.Error != "")
			is.True(t, strings.Contains(result.Error, tt.want))
		})
	}
}

func TestVerify_NoAudience(t *testing.T) {
	v := &Verifier{expectedAudience: ""}
	jwt := makeJWT(
		t,
		map[string]any{"alg": "ES256"},
		map[string]any{"email": "user@example.com"},
	)

	result := v.Verify(t.Context(), jwt)

	is.True(t, !result.Valid)
	is.True(t, strings.Contains(result.Error, "no IAP_AUDIENCE configured"))
	// Claims should still be decoded even though verification failed.
	is.True(t, result.Claims != nil)
}

func TestExpectedAudience(t *testing.T) {
	v := &Verifier{expectedAudience: "test-audience"}
	is.Equal(t, v.ExpectedAudience(), "test-audience", "")
}

func TestParseClaims_AllFields(t *testing.T) {
	payload := map[string]any{
		"iss":   "https://cloud.google.com/iap",
		"sub":   "subject-123",
		"email": "user@example.com",
		"hd":    "example.com",
		"aud":   "/projects/123/global/backendServices/456",
		"iat":   1700000000.0,
		"exp":   1700003600.0,
		"google": map[string]any{
			"access_levels": []any{"level1", "level2"},
		},
	}

	c := parseClaims(payload)

	is.Equal(t, c.Issuer, "https://cloud.google.com/iap", "")
	is.Equal(t, c.Subject, "subject-123", "")
	is.Equal(t, c.Email, "user@example.com", "")
	is.Equal(t, c.HostedDomain, "example.com", "")
	is.Equal(t, c.Audience, "/projects/123/global/backendServices/456", "")
	is.Equal(t, len(c.AccessLevels), 2, "")
	is.Equal(t, c.AccessLevels[0], "level1", "")
	is.Equal(t, c.IssuedAt.Unix(), int64(1700000000), "")
	is.Equal(t, c.ExpiresAt.Unix(), int64(1700003600), "")
}

func TestParseClaims_EmptyPayload(t *testing.T) {
	c := parseClaims(map[string]any{})
	is.Equal(t, c.Issuer, "", "")
	is.Equal(t, c.Email, "", "")
	is.Equal(t, c.Subject, "", "")
}
