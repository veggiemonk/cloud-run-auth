package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/veggiemonk/cloud-run-auth/internal/is"
	"github.com/veggiemonk/cloud-run-auth/internal/middleware"
)

func TestCSRF_TokenGeneration(t *testing.T) {
	csrf, err := middleware.NewCSRF("")
	is.NoErr(t, err)

	token1 := csrf.Token("session-1")
	token2 := csrf.Token("session-1")
	is.Equal(t, token1, token2, "same session should produce same token")

	token3 := csrf.Token("session-2")
	is.True(t, token1 != token3)
}

func TestCSRF_ValidToken(t *testing.T) {
	csrf, err := middleware.NewCSRF("")
	is.NoErr(t, err)

	token := csrf.Token("sess-abc")
	is.True(t, csrf.ValidToken("sess-abc", token))
	is.True(t, !csrf.ValidToken("sess-abc", "wrong-token"))
}

func TestCSRF_RequireCSRF_BlocksWithoutToken(t *testing.T) {
	csrf, err := middleware.NewCSRF("")
	is.NoErr(t, err)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	getSessionID := func(r *http.Request) string {
		if c, err := r.Cookie("session"); err == nil {
			return c.Value
		}
		return ""
	}

	handler := csrf.RequireCSRF(getSessionID)(inner)

	// POST without CSRF token should be rejected.
	form := url.Values{}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/action", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session", Value: "sess-123"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	is.Equal(t, rec.Code, http.StatusForbidden, "POST without CSRF")
}

func TestCSRF_RequireCSRF_AllowsValidToken(t *testing.T) {
	csrf, err := middleware.NewCSRF("")
	is.NoErr(t, err)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	getSessionID := func(r *http.Request) string {
		if c, err := r.Cookie("session"); err == nil {
			return c.Value
		}
		return ""
	}

	handler := csrf.RequireCSRF(getSessionID)(inner)
	token := csrf.Token("sess-123")

	form := url.Values{middleware.CSRFFormFieldName: {token}}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/action", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "session", Value: "sess-123"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	is.Equal(t, rec.Code, http.StatusOK, "POST with valid CSRF")
}

func TestCSRF_RequireCSRF_AllowsGET(t *testing.T) {
	csrf, err := middleware.NewCSRF("")
	is.NoErr(t, err)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	getSessionID := func(r *http.Request) string { return "sess-123" }
	handler := csrf.RequireCSRF(getSessionID)(inner)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/page", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	is.Equal(t, rec.Code, http.StatusOK, "GET should pass through")
}
