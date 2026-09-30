package repository

import (
	"testing"
	"time"
)

func TestJobBackoffCapsDeterministically(t *testing.T) {
	cases := map[int]time.Duration{
		0: time.Second,
		1: time.Second,
		2: 2 * time.Second,
		3: 4 * time.Second,
		4: 8 * time.Second,
		5: 16 * time.Second,
		6: 32 * time.Second,
		7: 32 * time.Second,
	}
	for attempt, want := range cases {
		if got := jobBackoff(attempt); got != want {
			t.Fatalf("attempt %d: expected %s, got %s", attempt, want, got)
		}
	}
}
