package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	roomiesclock "github.com/roomies/backend/internal/clock"
	"github.com/roomies/backend/internal/database"
	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/providers"
)

func newNotificationRepoTest(t *testing.T) (*NotificationRepository, *roomiesclock.FakeClock) {
	t.Helper()
	db, err := database.Connect("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO users(id,name,email,password_hash) VALUES('u1','One','one@test','x'),('u2','Two','two@test','x'); INSERT INTO houses(id,name) VALUES('h1','House'); INSERT INTO house_members(id,house_id,user_id,role) VALUES('m1','h1','u1','admin'),('m2','h1','u2','monitor')`)
	if err != nil {
		t.Fatal(err)
	}
	key := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	cipher, err := providers.NewNotificationDeliveryCipher("notification-v1", key, "")
	if err != nil {
		t.Fatal(err)
	}
	clock := roomiesclock.NewFake(time.Date(2025, 3, 8, 23, 0, 0, 0, time.UTC))
	return NewNotificationRepository(db, clock, cipher), clock
}
func TestSubscriptionCapabilityCanBeRegisteredAcrossHousesWithoutChangingOwnership(t *testing.T) {
	repo, _ := newNotificationRepoTest(t)
	ctx := context.Background()
	if _, err := repo.db.Exec(`INSERT INTO houses(id,name) VALUES('h2','Second'); INSERT INTO house_members(id,house_id,user_id,role) VALUES('m3','h2','u1','member')`); err != nil {
		t.Fatal(err)
	}
	first, err := repo.CreateSubscription(ctx, "h1", "u1", models.CreateNotificationSubscriptionRequest{Platform: "android_fcm", Token: "shared-device"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.CreateSubscription(ctx, "h2", "u1", models.CreateNotificationSubscriptionRequest{Platform: "android_fcm", Token: "shared-device"})
	if err != nil {
		t.Fatalf("same device in second house: %v", err)
	}
	otherOwner, err := repo.CreateSubscription(ctx, "h1", "u2", models.CreateNotificationSubscriptionRequest{Platform: "android_fcm", Token: "shared-device"})
	if err != nil {
		t.Fatalf("cross-user registration must not overwrite ownership: %v", err)
	}
	if first.ID == second.ID || first.ID == otherOwner.ID || second.ID == otherOwner.ID {
		t.Fatalf("house/user registrations must retain distinct ownership: %s %s %s", first.ID, second.ID, otherOwner.ID)
	}
	again, err := repo.CreateSubscription(ctx, "h1", "u1", models.CreateNotificationSubscriptionRequest{Platform: "android_fcm", Token: "shared-device"})
	if err != nil || again.ID != first.ID {
		t.Fatalf("same house/user/device should update one registration: %#v %v", again, err)
	}
	if err := repo.RevokeSubscription(ctx, "h1", "u2", first.ID); !errors.Is(err, ErrNotificationNotFound) {
		t.Fatalf("cross-user revoke changed ownership: %v", err)
	}
}

func TestSubscriptionSecretsEncryptedAndMetadataOwned(t *testing.T) {
	repo, _ := newNotificationRepoTest(t)
	ctx := context.Background()
	created, err := repo.CreateSubscription(ctx, "h1", "u1", models.CreateNotificationSubscriptionRequest{Platform: "android_fcm", Token: "private-device-token", DeviceLabel: "phone"})
	if err != nil {
		t.Fatal(err)
	}
	var ciphertext, hash string
	if err := repo.db.QueryRow(`SELECT ciphertext,identity_hash FROM notification_subscriptions WHERE id=$1`, created.ID).Scan(&ciphertext, &hash); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ciphertext, "private-device-token") || strings.Contains(hash, "private-device-token") {
		t.Fatal("subscription secret stored in plaintext")
	}
	values, err := repo.ListSubscriptions(ctx, "h1", "u1")
	if err != nil || len(values) != 1 {
		t.Fatalf("metadata list: %v %#v", err, values)
	}
	if err := repo.RevokeSubscription(ctx, "h1", "u2", created.ID); err == nil {
		t.Fatal("different user revoked subscription")
	}
}
func TestEnqueuePrivacyQuietHoursAndDigestDedupe(t *testing.T) {
	repo, _ := newNotificationRepoTest(t)
	ctx := context.Background()
	start, end := 22*60, 7*60
	if err := repo.PutPreferences(ctx, models.NotificationPreferences{HouseID: "h1", UserID: "u1", ExpenseCreatedEnabled: true, ReminderEnabled: true, Cadence: "immediate", Timezone: "UTC", QuietStartMinutes: &start, QuietEndMinutes: &end, DigestMinutes: 480}); err != nil {
		t.Fatal(err)
	}
	occurrence := time.Date(2025, 3, 8, 23, 30, 0, 0, time.UTC)
	if err := repo.Enqueue(ctx, "h1", "expense_created", "expense", "expense-1", occurrence); err != nil {
		t.Fatal(err)
	}
	var payload string
	var available time.Time
	if err := repo.db.QueryRow(`SELECT payload,available_at FROM outbox_messages WHERE topic='notification.delivery' AND payload LIKE '%expense-1%' LIMIT 1`).Scan(&payload, &available); err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"amount", "description", "receipt", "token"} {
		if strings.Contains(strings.ToLower(payload), private) {
			t.Fatalf("payload leaked %s: %s", private, payload)
		}
	}
	if !available.Equal(time.Date(2025, 3, 9, 7, 0, 0, 0, time.UTC)) {
		t.Fatalf("quiet delay=%s", available)
	}
	if err := repo.PutPreferences(ctx, models.NotificationPreferences{HouseID: "h1", UserID: "u2", ExpenseCreatedEnabled: true, ReminderEnabled: true, Cadence: "daily_digest", Timezone: "UTC", DigestMinutes: 480}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Enqueue(ctx, "h1", "reminder", "scheduled_event", "event-a", occurrence); err != nil {
		t.Fatal(err)
	}
	if err := repo.Enqueue(ctx, "h1", "reminder", "scheduled_event", "event-b", occurrence.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := repo.db.Get(&count, `SELECT COUNT(*) FROM outbox_messages WHERE topic='notification.delivery' AND payload LIKE '%event-%' AND payload LIKE '%"user_id":"u2"%'`); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected one durable trigger per digest row, got %d", count)
	}
}
func TestSharedExpenseAtomicallyEnqueuesPrivacySafeNotifications(t *testing.T) {
	repo, clock := newNotificationRepoTest(t)
	expenses := NewExpenseRepository(repo.db, repo)
	expense := &models.Expense{ID: "expense-private-data-test", HouseID: "h1", PayerID: "u1", Amount: 42.55, AmountCents: 4255, Description: "private groceries description", Date: "2025-03-08", Visibility: "shared", CreatedAt: clock.Now()}
	splits := []models.ExpenseSplit{{ID: "split-1", ExpenseID: expense.ID, UserID: "u1", ShareAmount: 42.55, ShareAmountCents: 4255}}
	if err := expenses.CreateAggregate(context.Background(), expense, nil, splits); err != nil {
		t.Fatal(err)
	}
	var payloads []string
	if err := repo.db.Select(&payloads, `SELECT payload FROM outbox_messages WHERE topic='notification.delivery'`); err != nil {
		t.Fatal(err)
	}
	if len(payloads) != 2 {
		t.Fatalf("expected two recipients, got %d", len(payloads))
	}
	for _, payload := range payloads {
		for _, forbidden := range []string{"42.55", "4255", "private groceries description", "receipt", "token"} {
			if strings.Contains(payload, forbidden) {
				t.Fatalf("private expense field leaked: %s", payload)
			}
		}
	}
}

func TestSchedulerOccurrenceIsFencedAndDeduped(t *testing.T) {
	repo, clock := newNotificationRepoTest(t)
	ctx := context.Background()
	event := &models.ScheduledHouseEvent{HouseID: "h1", CreatorID: "u1", Title: "Bins", Timezone: "America/New_York", DTStartLocal: "2025-03-08T18:00:00", RRule: "FREQ=DAILY;COUNT=3", Enabled: true}
	clock.Advance(-time.Minute)
	if err := repo.CreateScheduledEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	clock.Advance(2 * time.Minute)
	type result struct {
		ran bool
		err error
	}
	results := make(chan result, 2)
	for _, owner := range []string{"worker-a", "worker-b"} {
		go func() { ran, err := repo.RunDueScheduledEvent(ctx, owner, time.Minute); results <- result{ran, err} }()
	}
	ranCount := 0
	for range 2 {
		value := <-results
		if value.err != nil {
			t.Fatal(value.err)
		}
		if value.ran {
			ranCount++
		}
	}
	if ranCount != 1 {
		t.Fatalf("expected one concurrent scheduler owner, got %d", ranCount)
	}
	var count int
	if err := repo.db.Get(&count, `SELECT COUNT(*) FROM notification_deliveries WHERE resource_id=$1 AND occurrence_at=$2`, event.ID, time.Date(2025, 3, 8, 23, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected active member and monitor recipients, got %d", count)
	}
}

func TestDigestSnapshotsCategoriesAndLeavesConcurrentEnqueueForItsOwnTrigger(t *testing.T) {
	repo, clock := newNotificationRepoTest(t)
	ctx := context.Background()
	if err := repo.PutPreferences(ctx, models.NotificationPreferences{HouseID: "h1", UserID: "u2", ExpenseCreatedEnabled: true, ReminderEnabled: true, Cadence: "daily_digest", Timezone: "UTC", DigestMinutes: 480}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateSubscription(ctx, "h1", "u2", models.CreateNotificationSubscriptionRequest{Platform: "android_fcm", Token: "digest-token"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Enqueue(ctx, "h1", "expense_created", "expense", "digest-expense", clock.Now()); err != nil {
		t.Fatal(err)
	}
	if err := repo.Enqueue(ctx, "h1", "reminder", "scheduled_event", "digest-reminder", clock.Now()); err != nil {
		t.Fatal(err)
	}
	clock.Advance(10 * time.Hour)
	if err := repo.PutPreferences(ctx, models.NotificationPreferences{HouseID: "h1", UserID: "u2", ExpenseCreatedEnabled: false, ReminderEnabled: true, Cadence: "daily_digest", Timezone: "UTC", DigestMinutes: 480}); err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := repo.db.Get(&raw, `SELECT payload FROM outbox_messages WHERE payload LIKE '%digest-expense%' AND payload LIKE '%"user_id":"u2"%'`); err != nil {
		t.Fatal(err)
	}
	var trigger NotificationPayload
	if err := json.Unmarshal([]byte(raw), &trigger); err != nil {
		t.Fatal(err)
	}
	sendStarted := make(chan struct{})
	allowSend := make(chan struct{})
	done := make(chan error, 1)
	var sent NotificationPayload
	go func() {
		_, err := repo.WithAuthorizedDispatch(ctx, trigger, func(_ NotificationTarget, body []byte) (bool, error) {
			if err := json.Unmarshal(body, &sent); err != nil {
				return false, err
			}
			close(sendStarted)
			<-allowSend
			return false, nil
		})
		done <- err
	}()
	<-sendStarted
	enqueueDone := make(chan error, 1)
	go func() {
		enqueueDone <- repo.Enqueue(ctx, "h1", "reminder", "scheduled_event", "concurrent-reminder", clock.Now())
	}()
	close(allowSend)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := <-enqueueDone; err != nil {
		t.Fatal(err)
	}
	if sent.CategoryCounts["reminder"] != 1 || sent.CategoryCounts["expense_created"] != 0 {
		t.Fatalf("digest did not revalidate mixed categories: %#v", sent.CategoryCounts)
	}
	if sent.ResourceID != "" || sent.ResourceType != "" || sent.Category != "" || sent.NotificationID != "" {
		t.Fatalf("digest leaked item identifiers: %#v", sent)
	}
	var dispatched, pending int
	if err := repo.db.Get(&dispatched, `SELECT COUNT(*) FROM notification_deliveries WHERE resource_id IN ('digest-expense','digest-reminder') AND dispatched_at IS NOT NULL`); err != nil {
		t.Fatal(err)
	}
	if err := repo.db.Get(&pending, `SELECT COUNT(*) FROM notification_deliveries WHERE resource_id='concurrent-reminder' AND user_id='u2' AND dispatched_at IS NULL`); err != nil {
		t.Fatal(err)
	}
	if dispatched != 2 || pending != 1 {
		t.Fatalf("snapshot marking mismatch: dispatched=%d concurrent_pending=%d", dispatched, pending)
	}
	if err := repo.db.Get(&pending, `SELECT COUNT(*) FROM outbox_messages WHERE payload LIKE '%concurrent-reminder%' AND payload LIKE '%"user_id":"u2"%'`); err != nil || pending != 1 {
		t.Fatalf("concurrent row lost durable trigger: count=%d err=%v", pending, err)
	}
}

func TestLazyKeyRotationConvergesAndTamperIsIsolated(t *testing.T) {
	repo, clock := newNotificationRepoTest(t)
	ctx := context.Background()
	oldKey := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("o", 32)))
	oldCipher, err := providers.NewNotificationDeliveryCipher("old", oldKey, "")
	if err != nil {
		t.Fatal(err)
	}
	repo.cipher = oldCipher
	good, err := repo.CreateSubscription(ctx, "h1", "u1", models.CreateNotificationSubscriptionRequest{Platform: "android_fcm", Token: "good-token"})
	if err != nil {
		t.Fatal(err)
	}
	bad, err := repo.CreateSubscription(ctx, "h1", "u1", models.CreateNotificationSubscriptionRequest{Platform: "android_fcm", Token: "bad-token"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.db.Exec(`UPDATE notification_subscriptions SET ciphertext=ciphertext || 'tampered' WHERE id=$1`, bad.ID); err != nil {
		t.Fatal(err)
	}
	newKey := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("n", 32)))
	repo.cipher, err = providers.NewNotificationDeliveryCipher("new", newKey, "old:"+oldKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Enqueue(ctx, "h1", "expense_created", "expense", "rotation-expense", clock.Now()); err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := repo.db.Get(&raw, `SELECT payload FROM outbox_messages WHERE payload LIKE '%rotation-expense%' AND payload LIKE '%u1%'`); err != nil {
		t.Fatal(err)
	}
	var payload NotificationPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	var targets []string
	summary, err := repo.WithAuthorizedDispatch(ctx, payload, func(target NotificationTarget, _ []byte) (bool, error) {
		targets = append(targets, target.Token)
		return false, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0] != "good-token" || len(summary.InvalidTargetIDs) != 1 || summary.InvalidTargetIDs[0] != bad.ID {
		t.Fatalf("tamper isolation failed: targets=%v summary=%+v", targets, summary)
	}
	var keyID, ciphertext string
	if err := repo.db.QueryRow(`SELECT key_id,ciphertext FROM notification_subscriptions WHERE id=$1`, good.ID).Scan(&keyID, &ciphertext); err != nil {
		t.Fatal(err)
	}
	if keyID != "new" {
		t.Fatalf("old key did not converge: %q", keyID)
	}
	var secret subscriptionSecret
	if err := repo.cipher.Decrypt(good.ID, ciphertext, &secret); err != nil || secret.Token != "good-token" {
		t.Fatalf("rotated ciphertext invalid: token=%q err=%v", secret.Token, err)
	}
	var revokedAt *time.Time
	if err := repo.db.Get(&revokedAt, `SELECT revoked_at FROM notification_subscriptions WHERE id=$1`, bad.ID); err != nil || revokedAt == nil {
		t.Fatalf("tampered target was not quarantined: %v %v", revokedAt, err)
	}
	var audits int
	if err := repo.db.Get(&audits, `SELECT COUNT(*) FROM audit_events WHERE target_id=$1 AND action='notification.subscription.quarantined' AND metadata NOT LIKE '%token%'`, bad.ID); err != nil || audits != 1 {
		t.Fatalf("safe audit missing: count=%d err=%v", audits, err)
	}
}
