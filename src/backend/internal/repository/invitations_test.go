package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/roomies/backend/internal/models"
)

func TestConcurrentAcceptAndRevokeLeaveConsistentInvitationState(t *testing.T) {
	repo, clk, cleanup := setupReliabilityRepoTest(t)
	defer cleanup()
	houseID, adminID := seedHouseFixture(t, repo)
	if _, err := repo.db.Exec(`INSERT INTO users (id, name, email, password_hash, created_at) VALUES
		('user-invitee', 'Invitee', 'invitee@example.com', 'hash', CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	invite, token, err := repo.CreateInvitation(context.Background(), CreateInvitationParams{
		HouseID: houseID, ActorID: adminID, Email: "invitee@example.com", Role: "member", ExpiresAt: clk.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	var acceptErr, revokeErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, acceptErr = repo.AcceptInvitation(context.Background(), token, "user-invitee", "invitee@example.com")
	}()
	go func() {
		defer wg.Done()
		<-start
		_, revokeErr = repo.RevokeInvitation(context.Background(), houseID, invite.ID, adminID)
	}()
	close(start)
	wg.Wait()

	var status string
	if err := repo.db.Get(&status, `SELECT status FROM house_invitations WHERE id = $1`, invite.ID); err != nil {
		t.Fatal(err)
	}
	var memberCount int
	if err := repo.db.Get(&memberCount, `SELECT COUNT(*) FROM house_members WHERE house_id = $1 AND user_id = $2`, houseID, "user-invitee"); err != nil {
		t.Fatal(err)
	}
	switch status {
	case "accepted":
		if acceptErr != nil {
			t.Fatalf("accept should succeed when final status is accepted: %v", acceptErr)
		}
		if !errors.Is(revokeErr, ErrInvitationStateConflict) {
			t.Fatalf("expected revoke state conflict after accept, got %v", revokeErr)
		}
		if memberCount != 1 {
			t.Fatalf("expected accepted invitee to be a member, got %d", memberCount)
		}
	case "revoked":
		if revokeErr != nil {
			t.Fatalf("revoke should succeed when final status is revoked: %v", revokeErr)
		}
		if !errors.Is(acceptErr, ErrInvitationUnavailable) {
			t.Fatalf("expected accept to fail when revoke won, got %v", acceptErr)
		}
		if memberCount != 0 {
			t.Fatalf("expected revoked invitee not to join, got %d members", memberCount)
		}
	default:
		t.Fatalf("unexpected final invitation status %q", status)
	}
}

func TestHouseEventDeliveryBarrierSerializesRemovalAndSendSQLite(t *testing.T) {
	repo, _, cleanup := setupReliabilityRepoTest(t)
	defer cleanup()
	houseID, adminID := seedHouseFixture(t, repo)
	memberID := "user-stream-member"
	if _, err := repo.db.Exec(`INSERT INTO users (id, name, email, password_hash, created_at) VALUES ($1, 'Stream Member', 'stream-member@example.com', 'hash', CURRENT_TIMESTAMP)`, memberID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.db.Exec(`INSERT INTO house_members (id, house_id, user_id, role, joined_at) VALUES ('member-stream', $1, $2, 'member', CURRENT_TIMESTAMP)`, houseID, memberID); err != nil {
		t.Fatal(err)
	}
	if err := repo.AppendHouseEvent(context.Background(), houseID, "test.event", &adminID, "test", "event-1", nil); err != nil {
		t.Fatal(err)
	}
	runHouseEventDeliveryBarrierTest(t, repo, houseID, adminID, memberID)
}

// runHouseEventDeliveryBarrierTest establishes the interleaving explicitly: the
// callback has fetched and is about to send an event while removal is attempted.
// Removal must not commit until the bounded send callback releases the house barrier.
func runHouseEventDeliveryBarrierTest(t *testing.T, repo *ReliabilityRepository, houseID, adminID, memberID string) {
	t.Helper()
	sendStarted := make(chan struct{})
	allowSend := make(chan struct{})
	snapshotDone := make(chan error, 1)
	go func() {
		snapshotDone <- repo.WithAuthorizedHouseEvents(context.Background(), houseID, memberID, 0, 100, func(events []models.HouseEvent) error {
			if len(events) != 1 || events[0].ResourceID != "event-1" {
				return fmt.Errorf("unexpected event snapshot: %#v", events)
			}
			close(sendStarted)
			<-allowSend
			return nil
		})
	}()
	<-sendStarted

	removeDone := make(chan error, 1)
	go func() {
		removeDone <- NewHouseRepository(repo.db).RemoveMember(context.Background(), houseID, memberID)
	}()
	select {
	case err := <-removeDone:
		t.Fatalf("removal committed while an authorized send held the barrier: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(allowSend)
	if err := <-snapshotDone; err != nil {
		t.Fatalf("authorized send failed: %v", err)
	}
	if err := <-removeDone; err != nil {
		t.Fatalf("remove member failed: %v", err)
	}

	called := false
	err := repo.WithAuthorizedHouseEvents(context.Background(), houseID, memberID, 0, 100, func([]models.HouseEvent) error {
		called = true
		return nil
	})
	if !errors.Is(err, ErrHouseMembershipUnavailable) || called {
		t.Fatalf("event snapshot after removal was authorized: err=%v callback=%t", err, called)
	}
}

func TestExpiredInvitationCanBeReissued(t *testing.T) {
	repo, clk, cleanup := setupReliabilityRepoTest(t)
	defer cleanup()
	houseID, adminID := seedHouseFixture(t, repo)
	first, _, err := repo.CreateInvitation(context.Background(), CreateInvitationParams{
		HouseID: houseID, ActorID: adminID, Email: "expired@example.com", Role: "member", ExpiresAt: clk.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	clk.Advance(2 * time.Hour)
	second, _, err := repo.CreateInvitation(context.Background(), CreateInvitationParams{
		HouseID: houseID, ActorID: adminID, Email: "expired@example.com", Role: "member", ExpiresAt: clk.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID {
		t.Fatal("expected expired invite reissue to create a new record")
	}
	var statuses []string
	if err := repo.db.Select(&statuses, `SELECT status FROM house_invitations WHERE house_id = $1 ORDER BY created_at ASC, id ASC`, houseID); err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 2 || statuses[0] != "expired" || statuses[1] != "pending" {
		t.Fatalf("unexpected invitation statuses after reissue: %#v", statuses)
	}
}
