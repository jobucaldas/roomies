package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/roomies/backend/internal/models"
)

func TestHouseholdVersionsRecurrenceCompletionAndRetention(t *testing.T) {
	reliability, clk, cleanup := setupReliabilityRepoTest(t)
	defer cleanup()
	houseID, userID := seedHouseFixture(t, reliability)
	repo := NewHouseholdRepository(reliability.db, clk)

	item := &models.GroceryItem{HouseID: houseID, Name: "Rice", Quantity: "1", Unit: "bag"}
	if err := repo.CreateGrocery(t.Context(), item, userID); err != nil {
		t.Fatal(err)
	}
	stale := *item
	item.Name = "Brown rice"
	if err := repo.UpdateGrocery(t.Context(), item, userID); err != nil {
		t.Fatal(err)
	}
	stale.Name = "stale write"
	if err := repo.UpdateGrocery(t.Context(), &stale, userID); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("want version conflict, got %v", err)
	}

	chore := &models.Chore{HouseID: houseID, Title: "Bins", Timezone: "America/New_York", DueLocal: "2025-03-08T02:30:00", RRule: "FREQ=DAILY;COUNT=4", ExDates: []string{"2025-03-10T02:30:00"}, Enabled: true}
	if err := repo.SaveChore(t.Context(), chore, userID, true); err != nil {
		t.Fatal(err)
	}
	occurrence := time.Date(2025, 3, 8, 7, 30, 0, 0, time.UTC)
	if _, err := repo.CompleteChore(t.Context(), chore, userID, occurrence); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CompleteChore(t.Context(), chore, userID, occurrence); !errors.Is(err, ErrAlreadyCompleted) {
		t.Fatalf("want unique completion, got %v", err)
	}
	excluded := time.Date(2025, 3, 10, 6, 30, 0, 0, time.UTC)
	if _, err := repo.CompleteChore(t.Context(), chore, userID, excluded); !errors.Is(err, ErrInvalidOccurrence) {
		t.Fatalf("excluded occurrence accepted: %v", err)
	}

	message, err := repo.CreateChat(t.Context(), houseID, userID, "retained secret")
	if err != nil {
		t.Fatal(err)
	}
	clk.Advance(366 * 24 * time.Hour)
	count, err := repo.RunChatRetention(t.Context(), "retention-test", time.Minute, 100)
	if err != nil || count != 1 {
		t.Fatalf("retention count=%d err=%v", count, err)
	}
	message, err = repo.GetChat(t.Context(), houseID, message.ID)
	if err != nil {
		t.Fatal(err)
	}
	if message.Body != nil || message.RedactedAt == nil || message.DeletedAt != nil {
		t.Fatalf("expected redacted metadata tombstone: %+v", message)
	}
	if count, err = repo.RunChatRetention(t.Context(), "retention-test", time.Minute, 100); err != nil || count != 0 {
		t.Fatalf("bounded rescan count=%d err=%v", count, err)
	}
}
