package middleware

import "testing"

func TestWindowLimiterBlocksAfterLimit(t *testing.T) {
	limiter := NewWindowLimiter(0)
	if !limiter.Allow("login|203.0.113.10", 2) || !limiter.Allow("login|203.0.113.10", 2) {
		t.Fatal("expected first two hits to be allowed")
	}
	if limiter.Allow("login|203.0.113.10", 2) {
		t.Fatal("expected third hit to be blocked")
	}
	if !limiter.Allow("login|203.0.113.11", 2) {
		t.Fatal("expected a different key to have its own budget")
	}
}
