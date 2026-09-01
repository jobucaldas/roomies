package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type DurableJob struct {
	ID             string         `db:"id"`
	Kind           string         `db:"kind"`
	DedupeKey      string         `db:"dedupe_key"`
	Payload        string         `db:"payload"`
	AvailableAt    time.Time      `db:"available_at"`
	LeaseOwner     sql.NullString `db:"lease_owner"`
	LeaseExpiresAt sql.NullTime   `db:"lease_expires_at"`
	Attempts       int            `db:"attempts"`
	MaxAttempts    int            `db:"max_attempts"`
	LastError      sql.NullString `db:"last_error"`
	DeadLetteredAt sql.NullTime   `db:"dead_lettered_at"`
	CompletedAt    sql.NullTime   `db:"completed_at"`
	CreatedAt      time.Time      `db:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at"`
}

func (r *ReliabilityRepository) AcquireNextJob(ctx context.Context, owner string, leaseDuration time.Duration) (*DurableJob, error) {
	now := r.clock.Now()
	leaseExpiresAt := now.Add(leaseDuration)
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	candidate, err := r.selectJobCandidateTx(ctx, tx, now)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE durable_jobs
		SET lease_owner = $1, lease_expires_at = $2, attempts = attempts + 1, updated_at = $3
		WHERE id = $4 AND completed_at IS NULL AND dead_lettered_at IS NULL AND available_at <= $5
			AND (lease_expires_at IS NULL OR lease_expires_at <= $6)`,
		owner, leaseExpiresAt, now, candidate.ID, now, now)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, nil
	}
	job, err := r.getJobTx(ctx, tx, candidate.ID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return job, nil
}

func (r *ReliabilityRepository) CompleteOutboxJob(ctx context.Context, jobID, outboxMessageID string) error {
	now := r.clock.Now()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE durable_jobs
		SET completed_at = $1, lease_owner = NULL, lease_expires_at = NULL, updated_at = $2, last_error = NULL
		WHERE id = $3`, now, now, jobID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE outbox_messages
		SET status = 'dispatched', dispatched_at = $1, updated_at = $2, last_error = NULL
		WHERE id = $3`, now, now, outboxMessageID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ReliabilityRepository) FailOutboxJob(ctx context.Context, job *DurableJob, outboxMessageID string, cause error) error {
	now := r.clock.Now()
	backoff := jobBackoff(job.Attempts)
	deadLetter := job.Attempts >= job.MaxAttempts
	message := cause.Error()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if deadLetter {
		if _, err := tx.ExecContext(ctx, `UPDATE durable_jobs
			SET dead_lettered_at = $1, lease_owner = NULL, lease_expires_at = NULL, last_error = $2, updated_at = $3
			WHERE id = $4`, now, message, now, job.ID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE outbox_messages
			SET status = 'dead_lettered', last_error = $1, updated_at = $2
			WHERE id = $3`, message, now, outboxMessageID); err != nil {
			return err
		}
	} else {
		if _, err := tx.ExecContext(ctx, `UPDATE durable_jobs
			SET available_at = $1, lease_owner = NULL, lease_expires_at = NULL, last_error = $2, updated_at = $3
			WHERE id = $4`, now.Add(backoff), message, now, job.ID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE outbox_messages
			SET last_error = $1, updated_at = $2
			WHERE id = $3`, message, now, outboxMessageID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *ReliabilityRepository) GetOutboxMessage(ctx context.Context, id string) (*OutboxMessage, error) {
	var message OutboxMessage
	if err := r.db.GetContext(ctx, &message, `SELECT id, topic, dedupe_key, payload, status, available_at,
		dispatched_at, last_error, created_at, updated_at
		FROM outbox_messages WHERE id = $1`, id); err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *ReliabilityRepository) GetJob(ctx context.Context, id string) (*DurableJob, error) {
	var job DurableJob
	if err := r.db.GetContext(ctx, &job, `SELECT id, kind, dedupe_key, payload, available_at, lease_owner, lease_expires_at,
		attempts, max_attempts, last_error, dead_lettered_at, completed_at, created_at, updated_at
		FROM durable_jobs WHERE id = $1`, id); err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *ReliabilityRepository) DecodeJobPayload(job *DurableJob) (string, error) {
	var payload jobPayload
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
		return "", err
	}
	if payload.OutboxMessageID == "" {
		return "", fmt.Errorf("job payload missing outbox message id")
	}
	return payload.OutboxMessageID, nil
}

func (r *ReliabilityRepository) selectJobCandidateTx(ctx context.Context, tx *sqlx.Tx, now time.Time) (*DurableJob, error) {
	query := `SELECT id, kind, dedupe_key, payload, available_at, lease_owner, lease_expires_at,
		attempts, max_attempts, last_error, dead_lettered_at, completed_at, created_at, updated_at
		FROM durable_jobs WHERE completed_at IS NULL AND dead_lettered_at IS NULL AND available_at <= $1
			AND (lease_expires_at IS NULL OR lease_expires_at <= $1)
		ORDER BY available_at ASC, created_at ASC, id ASC LIMIT 1`
	if r.db.DriverName() == "postgres" {
		query += ` FOR UPDATE`
	}
	var job DurableJob
	if err := tx.GetContext(ctx, &job, query, now); err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *ReliabilityRepository) getJobTx(ctx context.Context, tx *sqlx.Tx, id string) (*DurableJob, error) {
	var job DurableJob
	if err := tx.GetContext(ctx, &job, `SELECT id, kind, dedupe_key, payload, available_at, lease_owner, lease_expires_at,
		attempts, max_attempts, last_error, dead_lettered_at, completed_at, created_at, updated_at
		FROM durable_jobs WHERE id = $1`, id); err != nil {
		return nil, err
	}
	return &job, nil
}

func jobBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	backoff := time.Duration(1<<min(attempt-1, 5)) * time.Second
	if backoff > 32*time.Second {
		return 32 * time.Second
	}
	return backoff
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
