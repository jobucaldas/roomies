package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	roomiesclock "github.com/roomies/backend/internal/clock"
	"github.com/roomies/backend/internal/database"
	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/providers"
)

func TestNotificationAuthorizationBarrierSQLite(t *testing.T) {
	db, err := database.Connect("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	runNotificationAuthorizationBarrierTests(t, db)
}

func TestNotificationAuthorizationBarrierPostgres(t *testing.T) {
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
	runNotificationAuthorizationBarrierTests(t, db)
}

func runNotificationAuthorizationBarrierTests(t *testing.T, db *sqlx.DB) {
	for _, operation := range []string{"remove", "revoke"} {
		t.Run(operation, func(t *testing.T) {
			repo, houseID, memberID, subscriptionID, payload := seedNotificationBarrierFixture(t, db)
			sendStarted := make(chan struct{})
			allowSend := make(chan struct{})
			dispatchDone := make(chan error, 1)
			go func() {
				_, err := repo.WithAuthorizedDispatch(context.Background(), payload, func(NotificationTarget, []byte) (bool, error) {
					close(sendStarted)
					<-allowSend
					return false, nil
				})
				dispatchDone <- err
			}()
			<-sendStarted

			operationDone := make(chan error, 1)
			go func() {
				if operation == "remove" {
					operationDone <- NewHouseRepository(db).RemoveMember(context.Background(), houseID, memberID)
					return
				}
				operationDone <- repo.RevokeSubscription(context.Background(), houseID, memberID, subscriptionID)
			}()
			select {
			case err := <-operationDone:
				t.Fatalf("%s committed while provider dispatch held authorization locks: %v", operation, err)
			case <-time.After(100 * time.Millisecond):
			}
			close(allowSend)
			if err := <-dispatchDone; err != nil {
				t.Fatal(err)
			}
			if err := <-operationDone; err != nil {
				t.Fatalf("%s failed: %v", operation, err)
			}

			laterID := "later-" + strings.ToLower(models.NewID())
			if _, err := db.Exec(`INSERT INTO notification_deliveries(id,house_id,user_id,category,resource_type,resource_id,occurrence_at,cadence,available_at,created_at) VALUES($1,$2,$3,'expense_created','expense',$1,$4,'immediate',$4,$4)`, laterID, houseID, memberID, repo.clock.Now()); err != nil {
				t.Fatal(err)
			}
			payload.NotificationID = laterID
			called := false
			_, err := repo.WithAuthorizedDispatch(context.Background(), payload, func(NotificationTarget, []byte) (bool, error) {
				called = true
				return false, nil
			})
			if err != nil || called {
				t.Fatalf("dispatch after %s used target: called=%t err=%v", operation, called, err)
			}
		})
	}
}

func seedNotificationBarrierFixture(t *testing.T, db *sqlx.DB) (*NotificationRepository, string, string, string, NotificationPayload) {
	t.Helper()
	suffix := strings.ToLower(models.NewID())
	houseID, adminID, memberID := "house-"+suffix, "admin-"+suffix, "member-"+suffix
	if _, err := db.Exec(`INSERT INTO users(id,name,email,password_hash) VALUES($1,'Admin',$2,'x'),($3,'Member',$4,'x')`, adminID, adminID+"@test", memberID, memberID+"@test"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO houses(id,name) VALUES($1,'House')`, houseID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO house_members(id,house_id,user_id,role) VALUES($1,$2,$3,'admin'),($4,$2,$5,'member')`, "admin-membership-"+suffix, houseID, adminID, "member-membership-"+suffix, memberID); err != nil {
		t.Fatal(err)
	}
	key := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("b", 32)))
	cipher, err := providers.NewNotificationDeliveryCipher("current", key, "")
	if err != nil {
		t.Fatal(err)
	}
	clk := roomiesclock.NewFake(time.Now().UTC())
	repo := NewNotificationRepository(db, clk, cipher)
	subscription, err := repo.CreateSubscription(context.Background(), houseID, memberID, models.CreateNotificationSubscriptionRequest{Platform: "android_fcm", Token: "token-" + suffix})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Enqueue(context.Background(), houseID, "expense_created", "expense", "expense-"+suffix, clk.Now()); err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := db.Get(&raw, `SELECT payload FROM outbox_messages WHERE topic='notification.delivery' AND payload LIKE $1 AND payload LIKE $2`, "%expense-"+suffix+"%", "%"+memberID+"%"); err != nil {
		t.Fatal(err)
	}
	var payload NotificationPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	return repo, houseID, memberID, subscription.ID, payload
}

func TestNotificationDispatchDeadlineBoundsLockAndCallback(t *testing.T) {
	repo, _ := newNotificationRepoTest(t)
	ctx := context.Background()
	subscription, err := repo.CreateSubscription(ctx, "h1", "u2", models.CreateNotificationSubscriptionRequest{Platform: "android_fcm", Token: "deadline-token"})
	if err != nil {
		t.Fatal(err)
	}
	_ = subscription
	if err := repo.Enqueue(ctx, "h1", "expense_created", "expense", "deadline-expense", repo.clock.Now()); err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := repo.db.Get(&raw, `SELECT payload FROM outbox_messages WHERE payload LIKE '%deadline-expense%' AND payload LIKE '%"user_id":"u2"%'`); err != nil {
		t.Fatal(err)
	}
	var payload NotificationPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	bounded, cancel := context.WithTimeout(ctx, 25*time.Millisecond)
	defer cancel()
	_, err = repo.WithAuthorizedDispatch(bounded, payload, func(NotificationTarget, []byte) (bool, error) {
		<-bounded.Done()
		return false, bounded.Err()
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected bounded dispatch deadline, got %v", err)
	}
}
