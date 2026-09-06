package repository

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	roomiesclock "github.com/roomies/backend/internal/clock"
	"github.com/roomies/backend/internal/database"
	"github.com/roomies/backend/internal/models"
)

func TestPostgresConcurrentInvitationAcceptAndRevoke(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for PostgreSQL lock regression test")
	}
	db, err := database.Connect(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if db.DriverName() != "postgres" {
		t.Skip("DATABASE_URL is not PostgreSQL")
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	clk := roomiesclock.NewFake(time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC))
	repo := NewReliabilityRepository(db, clk)
	for i := 0; i < 10; i++ {
		houseID, adminID, inviteeID := seedPostgresInvitationFixture(t, repo)
		inviteeEmail := inviteeID + "@example.test"
		invite, token, err := repo.CreateInvitation(context.Background(), CreateInvitationParams{
			HouseID: houseID, ActorID: adminID, Email: inviteeEmail, Role: "member", ExpiresAt: clk.Now().Add(24 * time.Hour),
		})
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		start := make(chan struct{})
		var acceptErr, revokeErr error
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, acceptErr = repo.AcceptInvitation(ctx, token, inviteeID, inviteeEmail)
		}()
		go func() {
			defer wg.Done()
			<-start
			_, revokeErr = repo.RevokeInvitation(ctx, houseID, invite.ID, adminID)
		}()
		close(start)
		wg.Wait()
		cancel()

		var status string
		if err := db.Get(&status, `SELECT status FROM house_invitations WHERE id = $1`, invite.ID); err != nil {
			t.Fatal(err)
		}
		switch status {
		case "accepted":
			if acceptErr != nil || !errors.Is(revokeErr, ErrInvitationStateConflict) {
				t.Fatalf("accepted invitation has accept=%v revoke=%v", acceptErr, revokeErr)
			}
		case "revoked":
			if revokeErr != nil || !errors.Is(acceptErr, ErrInvitationUnavailable) {
				t.Fatalf("revoked invitation has accept=%v revoke=%v", acceptErr, revokeErr)
			}
		default:
			t.Fatalf("unexpected status %q (accept=%v revoke=%v)", status, acceptErr, revokeErr)
		}
	}
}

func TestPostgresHouseEventDeliveryBarrierSerializesRemovalAndSend(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for PostgreSQL event delivery lock regression test")
	}
	db, err := database.Connect(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if db.DriverName() != "postgres" {
		t.Skip("DATABASE_URL is not PostgreSQL")
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	repo := NewReliabilityRepository(db, roomiesclock.RealClock{})
	houseID, adminID, memberID := seedPostgresInvitationFixture(t, repo)
	if _, err := db.Exec(`INSERT INTO house_members (id, house_id, user_id, role, joined_at) VALUES ($1, $2, $3, 'member', CURRENT_TIMESTAMP)`, models.NewID(), houseID, memberID); err != nil {
		t.Fatal(err)
	}
	if err := repo.AppendHouseEvent(context.Background(), houseID, "test.event", &adminID, "test", "event-1", nil); err != nil {
		t.Fatal(err)
	}
	runHouseEventDeliveryBarrierTest(t, repo, houseID, adminID, memberID)
}

func seedPostgresInvitationFixture(t *testing.T, repo *ReliabilityRepository) (houseID, adminID, inviteeID string) {
	t.Helper()
	houseID = models.NewID()
	adminID = models.NewID()
	inviteeID = models.NewID()
	for _, user := range []struct{ id, email string }{
		{adminID, adminID + "@example.test"},
		{inviteeID, inviteeID + "@example.test"},
	} {
		if _, err := repo.db.Exec(`INSERT INTO users (id, name, email, password_hash, created_at) VALUES ($1, $2, $3, 'hash', CURRENT_TIMESTAMP)`, user.id, user.id, user.email); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repo.db.Exec(`INSERT INTO houses (id, name, created_at) VALUES ($1, 'PostgreSQL lock test', CURRENT_TIMESTAMP)`, houseID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.db.Exec(`INSERT INTO house_members (id, house_id, user_id, role, joined_at) VALUES ($1, $2, $3, 'admin', CURRENT_TIMESTAMP)`, models.NewID(), houseID, adminID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := repo.db.Exec(`DELETE FROM houses WHERE id = $1`, houseID); err != nil {
			t.Errorf("delete PostgreSQL invitation fixture: %v", err)
		}
		if _, err := repo.db.Exec(`DELETE FROM users WHERE id IN ($1, $2)`, adminID, inviteeID); err != nil {
			t.Errorf("delete PostgreSQL invitation users: %v", err)
		}
	})
	return houseID, adminID, inviteeID
}
