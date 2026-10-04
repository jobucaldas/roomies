package middleware

import (
	"fmt"
	"testing"
	"time"
)

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

func TestWindowLimiterForgetsIdleKeys(t *testing.T) {
	limiter := NewWindowLimiter(time.Millisecond)
	for i := 0; i < 100; i++ {
		limiter.Allow(fmt.Sprintf("login|198.51.100.%d", i), 5)
	}
	time.Sleep(5 * time.Millisecond)
	limiter.Allow("login|203.0.113.1", 5)
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if len(limiter.hits) != 1 {
		t.Fatalf("expected idle keys to be swept, %d remain", len(limiter.hits))
	}
}
