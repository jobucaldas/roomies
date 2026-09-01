package repository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
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
