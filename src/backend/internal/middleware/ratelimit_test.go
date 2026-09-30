package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuthRateLimiterBlocksAfterLimit(t *testing.T) {
	limiter := NewAuthRateLimiter(3, time.Minute)
	now := time.Unix(1_700_000_000, 0)
	limiter.now = func() time.Time { return now }

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		req.RemoteAddr = "203.0.113.10:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("attempt %d status = %d", i+1, w.Code)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
}
