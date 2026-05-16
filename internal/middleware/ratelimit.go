package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// ForwardedForHeader is the header set by Cloud Run's load balancer.
const ForwardedForHeader = "X-Forwarded-For"

// IPRateLimiter provides per-IP rate limiting using token buckets.
type IPRateLimiter struct {
	visitors sync.Map
	limit    rate.Limit
	burst    int
}

// NewIPRateLimiter creates a rate limiter that allows burst requests per interval per IP.
func NewIPRateLimiter(burst int, interval time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		limit: rate.Every(interval / time.Duration(burst)),
		burst: burst,
	}
}

// Limit returns middleware that rate-limits requests by client IP.
func (rl *IPRateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := ClientIP(r)
		limiter := rl.limiterFor(ip)
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *IPRateLimiter) limiterFor(ip string) *rate.Limiter {
	if v, ok := rl.visitors.Load(ip); ok {
		if limiter, ok := v.(*rate.Limiter); ok {
			return limiter
		}
	}
	limiter := rate.NewLimiter(rl.limit, rl.burst)
	actual, _ := rl.visitors.LoadOrStore(ip, limiter)
	if existing, ok := actual.(*rate.Limiter); ok {
		return existing
	}
	// Map held an unexpected type; replace with a fresh limiter.
	rl.visitors.Store(ip, limiter)
	return limiter
}

// UserRateLimiter provides per-user rate limiting using an email extractor function.
type UserRateLimiter struct {
	ExtractEmail func(*http.Request) string
	users        sync.Map
	limit        rate.Limit
	burst        int
}

// NewUserRateLimiter creates a rate limiter that allows burst requests per interval per user.
func NewUserRateLimiter(burst int, interval time.Duration, extractEmail func(*http.Request) string) *UserRateLimiter {
	return &UserRateLimiter{
		limit:        rate.Every(interval / time.Duration(burst)),
		burst:        burst,
		ExtractEmail: extractEmail,
	}
}

// Limit returns middleware that rate-limits by user email.
// Must be applied AFTER auth middleware.
func (rl *UserRateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email := rl.ExtractEmail(r)
		if email == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !rl.allow(email) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *UserRateLimiter) allow(email string) bool {
	if v, ok := rl.users.Load(email); ok {
		if limiter, ok := v.(*rate.Limiter); ok {
			return limiter.Allow()
		}
	}
	limiter := rate.NewLimiter(rl.limit, rl.burst)
	actual, _ := rl.users.LoadOrStore(email, limiter)
	if existing, ok := actual.(*rate.Limiter); ok {
		return existing.Allow()
	}
	// Map held an unexpected type; replace with a fresh limiter.
	rl.users.Store(email, limiter)
	return limiter.Allow()
}

// ClientIP extracts the client IP from the request, preferring the first
// IP in X-Forwarded-For (set by Cloud Run's load balancer).
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get(ForwardedForHeader); xff != "" {
		for i, c := range xff {
			if c == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	return r.RemoteAddr
}
