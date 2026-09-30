package repository

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStaleLeaseCannotCompleteOrFailReclaimedJob(t *testing.T) {
	repo, clk, cleanup := setupReliabilityRepoTest(t)
	defer cleanup()
	job, outboxID := createInvitationJob(t, repo, clk, "lease-reclaim@example.com")

	stale, err := repo.AcquireNextJob(context.Background(), "worker-a", time.Second)
	if err != nil || stale == nil || stale.ID != job.ID {
		t.Fatalf("acquire stale lease: job=%+v err=%v", stale, err)
	}
	clk.Advance(2 * time.Second)
	current, err := repo.AcquireNextJob(context.Background(), "worker-b", time.Second)
	if err != nil || current == nil || current.ID != job.ID {
		t.Fatalf("reclaim job: job=%+v err=%v", current, err)
	}
	if err := repo.CompleteOutboxJob(context.Background(), current, outboxID); err != nil {
		t.Fatalf("complete current lease: %v", err)
	}
	if err := repo.CompleteOutboxJob(context.Background(), stale, outboxID); !errors.Is(err, ErrJobLeaseLost) {
		t.Fatalf("stale completion error = %v, want ErrJobLeaseLost", err)
	}
	if err := repo.FailOutboxJob(context.Background(), stale, outboxID, errors.New("late failure")); !errors.Is(err, ErrJobLeaseLost) {
		t.Fatalf("stale failure error = %v, want ErrJobLeaseLost", err)
	}

	storedJob, err := repo.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !storedJob.CompletedAt.Valid || storedJob.DeadLetteredAt.Valid {
		t.Fatalf("stale worker changed terminal job state: %+v", storedJob)
	}
	message, err := repo.GetOutboxMessage(context.Background(), outboxID)
	if err != nil {
		t.Fatal(err)
	}
	if message.Status != "dispatched" || message.LastError.Valid {
		t.Fatalf("stale worker corrupted outbox state: %+v", message)
	}
}

func TestStaleLeaseCannotOverwriteReclaimedFailure(t *testing.T) {
	repo, clk, cleanup := setupReliabilityRepoTest(t)
	defer cleanup()
	job, outboxID := createInvitationJob(t, repo, clk, "lease-failure@example.com")

	stale, err := repo.AcquireNextJob(context.Background(), "worker-a", time.Second)
	if err != nil || stale == nil {
		t.Fatalf("acquire stale lease: job=%+v err=%v", stale, err)
	}
	clk.Advance(2 * time.Second)
	current, err := repo.AcquireNextJob(context.Background(), "worker-b", time.Second)
	if err != nil || current == nil {
		t.Fatalf("reclaim job: job=%+v err=%v", current, err)
	}
	if err := repo.FailOutboxJob(context.Background(), current, outboxID, errors.New("current failure")); err != nil {
		t.Fatalf("fail current lease: %v", err)
	}
	if err := repo.CompleteOutboxJob(context.Background(), stale, outboxID); !errors.Is(err, ErrJobLeaseLost) {
		t.Fatalf("stale completion error = %v, want ErrJobLeaseLost", err)
	}
	if err := repo.FailOutboxJob(context.Background(), stale, outboxID, errors.New("late failure")); !errors.Is(err, ErrJobLeaseLost) {
		t.Fatalf("stale failure error = %v, want ErrJobLeaseLost", err)
	}

	storedJob, err := repo.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedJob.CompletedAt.Valid || storedJob.DeadLetteredAt.Valid || storedJob.LastError.String != "current failure" {
		t.Fatalf("stale worker changed failed job state: %+v", storedJob)
	}
	message, err := repo.GetOutboxMessage(context.Background(), outboxID)
	if err != nil {
		t.Fatal(err)
	}
	if message.Status != "pending" || message.LastError.String != "current failure" {
		t.Fatalf("stale worker corrupted failed outbox state: %+v", message)
	}
}

func TestLeaseGenerationFencesSameOwnerReclaim(t *testing.T) {
	repo, clk, cleanup := setupReliabilityRepoTest(t)
	defer cleanup()
	_, outboxID := createInvitationJob(t, repo, clk, "lease-generation@example.com")

	first, err := repo.AcquireNextJob(context.Background(), "worker", time.Second)
	if err != nil || first == nil {
		t.Fatalf("acquire first lease: job=%+v err=%v", first, err)
	}
	clk.Advance(2 * time.Second)
	second, err := repo.AcquireNextJob(context.Background(), "worker", time.Second)
	if err != nil || second == nil {
		t.Fatalf("reacquire lease: job=%+v err=%v", second, err)
	}
	if second.LeaseGeneration != first.LeaseGeneration+1 {
		t.Fatalf("lease generation = %d, want %d", second.LeaseGeneration, first.LeaseGeneration+1)
	}
	if err := repo.CompleteOutboxJob(context.Background(), first, outboxID); !errors.Is(err, ErrJobLeaseLost) {
		t.Fatalf("same-owner stale completion error = %v, want ErrJobLeaseLost", err)
	}
	if err := repo.CompleteOutboxJob(context.Background(), second, outboxID); err != nil {
		t.Fatalf("complete current generation: %v", err)
	}
}

func createInvitationJob(t *testing.T, repo *ReliabilityRepository, clk interface{ Now() time.Time }, email string) (*DurableJob, string) {
	t.Helper()
	houseID, adminID := seedHouseFixture(t, repo)
	invite, _, err := repo.CreateInvitation(context.Background(), CreateInvitationParams{
		HouseID: houseID, ActorID: adminID, Email: email, Role: "member", ExpiresAt: clk.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	var jobID, outboxID string
	if err := repo.db.QueryRow(`SELECT id FROM durable_jobs WHERE dedupe_key = $1`, "job:outbox:invitation:"+invite.ID).Scan(&jobID); err != nil {
		t.Fatal(err)
	}
	job, err := repo.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatal(err)
	}
	outboxID, err = repo.DecodeJobPayload(job)
	if err != nil {
		t.Fatal(err)
	}
	return job, outboxID
}
