package models

import (
	"testing"

	"github.com/oklog/ulid/v2"
)

func TestNewIDReturnsULID(t *testing.T) {
	id := NewID()
	if len(id) != 26 {
		t.Fatalf("expected 26-character ULID, got %q", id)
	}
	if _, err := ulid.ParseStrict(id); err != nil {
		t.Fatalf("NewID returned invalid ULID %q: %v", id, err)
	}
}
