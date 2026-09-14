package recurrence

import (
	"testing"
	"time"
)

func TestExpandPreservesLocalTimeAcrossDSTAndAppliesEXDATE(t *testing.T) {
	spec := Spec{Timezone: "America/New_York", DTStartLocal: "2025-03-08T09:00:00", Rule: "FREQ=DAILY;COUNT=4", ExDates: []string{"2025-03-09T09:00:00"}}
	if err := Validate(spec, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	values, err := Expand(spec, time.Date(2025, 3, 7, 0, 0, 0, 0, time.UTC), time.Date(2025, 3, 13, 0, 0, 0, 0, time.UTC), 366)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 3 {
		t.Fatalf("expected 3 values, got %v", values)
	}
	want := []string{"2025-03-08T14:00:00Z", "2025-03-10T13:00:00Z", "2025-03-11T13:00:00Z"}
	for i := range want {
		if got := values[i].Format(time.RFC3339); got != want[i] {
			t.Fatalf("value %d: got %s want %s", i, got, want[i])
		}
	}
}
func TestValidateBounds(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []Spec{{Timezone: "UTC", DTStartLocal: "2025-01-01T09:00:00", Rule: "FREQ=HOURLY;COUNT=2"}, {Timezone: "UTC", DTStartLocal: "2025-01-01T09:00:00", Rule: "FREQ=DAILY;INTERVAL=367;COUNT=2"}, {Timezone: "UTC", DTStartLocal: "2025-01-01T09:00:00", Rule: "FREQ=WEEKLY;COUNT=367"}, {Timezone: "UTC", DTStartLocal: "2025-01-01T09:00:00", Rule: "FREQ=MONTHLY"}, {Timezone: "UTC", DTStartLocal: "2025-01-01T09:00:00", Rule: "FREQ=DAILY;UNTIL=20270101T000000Z"}}
	for _, spec := range tests {
		if err := Validate(spec, now); err == nil {
			t.Fatalf("expected rejection for %s", spec.Rule)
		}
	}
}
func TestExpandCapsOutputs(t *testing.T) {
	spec := Spec{Timezone: "UTC", DTStartLocal: "2025-01-01T09:00:00", Rule: "FREQ=DAILY;COUNT=366"}
	values, err := Expand(spec, time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 10 {
		t.Fatalf("expected bounded 10 outputs, got %d", len(values))
	}
}
