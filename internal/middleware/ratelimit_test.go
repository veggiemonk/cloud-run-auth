package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/veggiemonk/cloud-run-auth/internal/is"
	"github.com/veggiemonk/cloud-run-auth/internal/middleware"
)

func TestIPRateLimiter_BurstEnforced(t *testing.T) {
	rl := middleware.NewIPRateLimiter(3, time.Minute)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := rl.Limit(inner)

	for i := range 3 {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		is.Equal(t, rec.Code, http.StatusOK, fmt.Sprintf("request %d", i))
	}

	// Next request should be rate limited.
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	is.Equal(t, rec.Code, http.StatusTooManyRequests, "request after burst")
}

func TestIPRateLimiter_PerIPIsolation(t *testing.T) {
	rl := middleware.NewIPRateLimiter(1, time.Minute)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := rl.Limit(inner)

	// Exhaust limit for IP A.
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = "1.1.1.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	is.Equal(t, rec.Code, http.StatusOK, "IP A first request")

	// IP B should still be allowed.
	req = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = "2.2.2.2:1234"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	is.Equal(t, rec.Code, http.StatusOK, "IP B first request")
}

func TestClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	is.Equal(t, middleware.ClientIP(req), "10.0.0.1", "")
}

func TestClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:4321"
	is.Equal(t, middleware.ClientIP(req), "192.168.1.1:4321", "")
}
