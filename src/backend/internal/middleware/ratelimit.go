package middleware

import (
	"sync"
	"time"
)

// WindowLimiter is an in-process fixed window used for auth endpoints.
// One replica is enough for the current deployment; a second process has its own counters.
type WindowLimiter struct {
	mu        sync.Mutex
	window    time.Duration
	hits      map[string][]time.Time
	lastSweep time.Time
}

func NewWindowLimiter(window time.Duration) *WindowLimiter {
	if window <= 0 {
		window = time.Minute
	}
	return &WindowLimiter{window: window, hits: map[string][]time.Time{}}
}

// Allow records one hit and reports whether it is still under limit.
func (l *WindowLimiter) Allow(key string, limit int) bool {
	if l == nil || limit <= 0 || key == "" {
		return true
	}
	now := time.Now()
	cutoff := now.Add(-l.window)
	l.mu.Lock()
	defer l.mu.Unlock()
	// Forget idle keys once per window so spoofed or one-off keys cannot grow
	// the map without bound.
	if now.Sub(l.lastSweep) >= l.window {
		for k, stamps := range l.hits {
			if len(stamps) == 0 || !stamps[len(stamps)-1].After(cutoff) {
				delete(l.hits, k)
			}
		}
		l.lastSweep = now
	}
	prev := l.hits[key]
	kept := make([]time.Time, 0, len(prev)+1)
	for _, ts := range prev {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= limit {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}
