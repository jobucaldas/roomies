package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AuthRateLimiter is a simple per-IP sliding window limiter for auth endpoints.
type AuthRateLimiter struct {
	mu       sync.Mutex
	window   time.Duration
	limit    int
	attempts map[string][]time.Time
	now      func() time.Time
}

func NewAuthRateLimiter(limit int, window time.Duration) *AuthRateLimiter {
	if limit <= 0 {
		limit = 20
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	return &AuthRateLimiter{
		window:   window,
		limit:    limit,
		attempts: make(map[string][]time.Time),
		now:      time.Now,
	}
}

func (l *AuthRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientIPKey(r) + "|" + r.Method + "|" + r.URL.Path
		if !l.allow(key) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"too many authentication attempts"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *AuthRateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	cutoff := now.Add(-l.window)
	kept := l.attempts[key][:0]
	for _, ts := range l.attempts[key] {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= l.limit {
		l.attempts[key] = kept
		return false
	}
	l.attempts[key] = append(kept, now)
	return true
}

func clientIPKey(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
