package worker_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	roomiesclock "github.com/roomies/backend/internal/clock"
	"github.com/roomies/backend/internal/database"
	"github.com/roomies/backend/internal/providers"
	"github.com/roomies/backend/internal/repository"
	"github.com/roomies/backend/internal/worker"
)

func TestConcurrentWorkersDispatchJobOnce(t *testing.T) {
	repo, db, clk, cleanup := setupWorkerTest(t)
	defer cleanup()
	houseID, adminID := seedHouseFixture(t, db)
	if _, _, err := repo.CreateInvitation(context.Background(), repository.CreateInvitationParams{
		HouseID: houseID, ActorID: adminID, Email: "invitee@example.com", Role: "member", ExpiresAt: clk.Now().Add(24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	provider := &providers.FakeInvitationProvider{}
	workerA := worker.New(nil, repo, provider, "worker-a", time.Millisecond, time.Second)
	workerB := worker.New(nil, repo, provider, "worker-b", time.Millisecond, time.Second)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := workerA.RunOnce(context.Background()); err != nil {
			t.Errorf("worker A: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := workerB.RunOnce(context.Background()); err != nil {
			t.Errorf("worker B: %v", err)
		}
	}()
	wg.Wait()

	if got := len(provider.Snapshot()); got != 1 {
		t.Fatalf("expected exactly one dispatch, got %d", got)
	}
	job := loadOnlyJob(t, repo, db)
	if !job.CompletedAt.Valid {
		t.Fatalf("expected job to be completed: %+v", job)
	}
}

func TestLeaseExpiryAllowsCrashRetry(t *testing.T) {
	repo, db, clk, cleanup := setupWorkerTest(t)
	defer cleanup()
	houseID, adminID := seedHouseFixture(t, db)
	if _, _, err := repo.CreateInvitation(context.Background(), repository.CreateInvitationParams{
		HouseID: houseID, ActorID: adminID, Email: "retry@example.com", Role: "member", ExpiresAt: clk.Now().Add(24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	job := loadOnlyJob(t, repo, db)
	claimed, err := repo.AcquireNextJob(context.Background(), "worker-a", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if claimed == nil || claimed.ID != job.ID {
		t.Fatalf("expected initial lease claim for %s, got %+v", job.ID, claimed)
	}

	clk.Advance(2 * time.Second)
	provider := &providers.FakeInvitationProvider{}
	workerB := worker.New(nil, repo, provider, "worker-b", time.Millisecond, time.Second)
	if err := workerB.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	job = loadOnlyJob(t, repo, db)
	if job.Attempts != 2 {
		t.Fatalf("expected second attempt after lease expiry, got %d", job.Attempts)
	}
	if !job.CompletedAt.Valid {
		t.Fatalf("expected retried job to complete: %+v", job)
	}
	if len(provider.Snapshot()) != 1 {
		t.Fatalf("expected one successful retried dispatch, got %d", len(provider.Snapshot()))
	}
}

func TestWorkerRetriesThenDeadLetters(t *testing.T) {
	repo, db, clk, cleanup := setupWorkerTest(t)
	defer cleanup()
	houseID, adminID := seedHouseFixture(t, db)
	if _, _, err := repo.CreateInvitation(context.Background(), repository.CreateInvitationParams{
		HouseID: houseID, ActorID: adminID, Email: "deadletter@example.com", Role: "member", ExpiresAt: clk.Now().Add(24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	provider := &providers.FakeInvitationProvider{FailuresLeft: 5}
	processor := worker.New(nil, repo, provider, "worker-a", time.Millisecond, time.Second)
	for i := 0; i < 5; i++ {
		if err := processor.RunOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
		clk.Advance(backoffForAttempt(i + 1))
	}
	job := loadOnlyJob(t, repo, db)
	if !job.DeadLetteredAt.Valid {
		t.Fatalf("expected job to be dead-lettered: %+v", job)
	}
	outboxID := decodeOnlyOutboxID(t, job)
	message, err := repo.GetOutboxMessage(context.Background(), outboxID)
	if err != nil {
		t.Fatal(err)
	}
	if message.Status != "dead_lettered" {
		t.Fatalf("expected dead-lettered outbox message, got %s", message.Status)
	}
}

func setupWorkerTest(t *testing.T) (*repository.ReliabilityRepository, *sqlx.DB, *roomiesclock.FakeClock, func()) {
	t.Helper()
	db, err := database.Connect("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	clk := roomiesclock.NewFake(time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC))
	return repository.NewReliabilityRepository(db, clk), db, clk, func() { _ = db.Close() }
}

func seedHouseFixture(t *testing.T, db *sqlx.DB) (houseID, adminID string) {
	t.Helper()
	adminID = "user-admin"
	houseID = "house-1"
	if _, err := db.Exec(`INSERT INTO users (id, name, email, password_hash, created_at) VALUES
		('user-admin', 'Admin', 'admin@example.com', 'hash', CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO houses (id, name, created_at) VALUES ('house-1', 'Test House', CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO house_members (id, house_id, user_id, role, joined_at) VALUES
		('member-admin', 'house-1', 'user-admin', 'admin', CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	return houseID, adminID
}

func loadOnlyJob(t *testing.T, repo *repository.ReliabilityRepository, db *sqlx.DB) *repository.DurableJob {
	t.Helper()
	var ids []string
	if err := db.Select(&ids, `SELECT id FROM durable_jobs ORDER BY created_at ASC, id ASC`); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected exactly one job, got %d", len(ids))
	}
	job, err := repo.GetJob(context.Background(), ids[0])
	if err != nil {
		t.Fatal(err)
	}
	return job
}

func decodeOnlyOutboxID(t *testing.T, job *repository.DurableJob) string {
	t.Helper()
	var payload struct {
		OutboxMessageID string `json:"outbox_message_id"`
	}
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.OutboxMessageID == "" {
		t.Fatal("missing outbox_message_id")
	}
	return payload.OutboxMessageID
}

func backoffForAttempt(attempt int) time.Duration {
	switch {
	case attempt <= 1:
		return time.Second
	case attempt == 2:
		return 2 * time.Second
	case attempt == 3:
		return 4 * time.Second
	case attempt == 4:
		return 8 * time.Second
	default:
		return 16 * time.Second
	}
}
