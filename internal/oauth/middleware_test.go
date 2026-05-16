package oauth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"

	"github.com/veggiemonk/cloud-run-auth/internal/is"
)

func TestRequireAuth_NoCookie(t *testing.T) {
	store := NewSessionStore(nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	handler := RequireAuth(store, next)
	req := httptest.NewRequestWithContext(t.Context(), "GET", "/protected", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	is.Equal(t, rr.Code, http.StatusFound, "")
	is.Equal(t, rr.Header().Get("Location"), "/auth/login", "")
}

func TestRequireAuth_InvalidSession(t *testing.T) {
	store := NewSessionStore(nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	handler := RequireAuth(store, next)
	req := httptest.NewRequestWithContext(t.Context(), "GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "nonexistent"})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	is.Equal(t, rr.Code, http.StatusFound, "")
}

func TestRequireAuth_ValidSession(t *testing.T) {
	store := NewSessionStore(nil)
	token := &oauth2.Token{AccessToken: "test-token"}
	session := store.Create("user@example.com", "Test User", "pic.jpg", token)

	var gotUser *UserInfo
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireAuth(store, next)
	req := httptest.NewRequestWithContext(t.Context(), "GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: session.ID})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	is.Equal(t, rr.Code, http.StatusOK, "")
	is.True(t, gotUser != nil)
	is.Equal(t, gotUser.Email, "user@example.com", "")
	is.Equal(t, gotUser.Name, "Test User", "")
}

func TestUserFromContext_NilWhenMissing(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), "GET", "/", nil)
	user := UserFromContext(req.Context())
	is.True(t, user == nil)
}
