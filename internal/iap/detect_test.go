package iap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDetect_NoHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	det := Detect(r)

	if det.HasJWT {
		t.Error("expected HasJWT=false with no headers")
	}
	if det.HasEmailHeader {
		t.Error("expected HasEmailHeader=false with no headers")
	}
	if det.HasIDHeader {
		t.Error("expected HasIDHeader=false with no headers")
	}
	if det.Warning != "" {
		t.Errorf("expected no warning, got: %s", det.Warning)
	}
}

func TestDetect_JWTOnly(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(HeaderJWTAssertion, "header.payload.signature")
	det := Detect(r)

	if !det.HasJWT {
		t.Error("expected HasJWT=true")
	}
	if det.RawJWT != "header.payload.signature" {
		t.Errorf("expected RawJWT to be set, got: %s", det.RawJWT)
	}
	if det.HasEmailHeader || det.HasIDHeader {
		t.Error("expected no email/ID headers")
	}
	if det.Warning != "" {
		t.Errorf("expected no warning, got: %s", det.Warning)
	}
}

func TestDetect_EmailAndIDWithoutJWT(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(HeaderAuthenticatedEmail, "accounts.google.com:user@example.com")
	r.Header.Set(HeaderAuthenticatedID, "accounts.google.com:12345")
	det := Detect(r)

	if det.HasJWT {
		t.Error("expected HasJWT=false")
	}
	if !det.HasEmailHeader {
		t.Error("expected HasEmailHeader=true")
	}
	if !det.HasIDHeader {
		t.Error("expected HasIDHeader=true")
	}
	if det.Email != "user@example.com" {
		t.Errorf("expected email prefix stripped, got: %s", det.Email)
	}
	if det.UserID != "12345" {
		t.Errorf("expected user ID prefix stripped, got: %s", det.UserID)
	}
	if det.Warning == "" {
		t.Error("expected bypass warning when headers present without JWT")
	}
}

func TestDetect_AllHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(HeaderJWTAssertion, "a.b.c")
	r.Header.Set(HeaderAuthenticatedEmail, "accounts.google.com:user@example.com")
	r.Header.Set(HeaderAuthenticatedID, "accounts.google.com:12345")
	det := Detect(r)

	if !det.HasJWT || !det.HasEmailHeader || !det.HasIDHeader {
		t.Error("expected all header flags to be true")
	}
	if det.Warning != "" {
		t.Error("expected no warning when JWT is present alongside headers")
	}
}

func TestDetect_EmailWithoutPrefix(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(HeaderJWTAssertion, "a.b.c")
	r.Header.Set(HeaderAuthenticatedEmail, "user@example.com")
	det := Detect(r)

	if det.Email != "user@example.com" {
		t.Errorf("expected email unchanged without prefix, got: %s", det.Email)
	}
}

func TestDetectionResult_RawJWTExcludedFromJSON(t *testing.T) {
	det := DetectionResult{
		HasJWT: true,
		RawJWT: "secret-token",
		Email:  "user@example.com",
	}

	b, err := json.Marshal(det)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, ok := m["raw_jwt"]; ok {
		t.Error("RawJWT should not appear in JSON output (json:\"-\" tag)")
	}
}
