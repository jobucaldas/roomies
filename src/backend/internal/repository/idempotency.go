package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/roomies/backend/internal/models"
)

var (
	ErrIdempotencyConflict   = errors.New("idempotency key reuse conflict")
	ErrIdempotencyInProgress = errors.New("idempotency request in progress")
)

type IdempotencyRecord struct {
	ID                  string         `db:"id"`
	UserID              string         `db:"user_id"`
	IdempotencyKey      string         `db:"idempotency_key"`
	Method              string         `db:"method"`
	Path                string         `db:"path"`
	BodyHash            string         `db:"body_hash"`
	State               string         `db:"state"`
	StatusCode          sql.NullInt64  `db:"status_code"`
	ResponseBody        sql.NullString `db:"response_body"`
	ResponseContentType sql.NullString `db:"response_content_type"`
	CreatedAt           time.Time      `db:"created_at"`
	UpdatedAt           time.Time      `db:"updated_at"`
	CompletedAt         sql.NullTime   `db:"completed_at"`
}

type StoredHTTPResponse struct {
	StatusCode  int
	Body        string
	ContentType string
}

func (r *ReliabilityRepository) StartIdempotentRequest(ctx context.Context, userID, key, method, path, bodyHash string) (*StoredHTTPResponse, error) {
	now := r.clock.Now()
	_, err := r.db.ExecContext(ctx, `INSERT INTO idempotency_keys
		(id, user_id, idempotency_key, method, path, body_hash, state, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'processing', $7, $8)`,
		models.NewID(), userID, key, method, path, bodyHash, now, now)
	if err == nil {
		return nil, nil
	}
	if !isUniqueViolation(err) {
		return nil, err
	}

	record, getErr := r.GetIdempotencyRecord(ctx, userID, key)
	if getErr != nil {
		return nil, getErr
	}
	if record.Method != method || record.Path != path || record.BodyHash != bodyHash {
		return nil, ErrIdempotencyConflict
	}
	if record.State != "completed" {
		return nil, ErrIdempotencyInProgress
	}
	if !record.StatusCode.Valid {
		return nil, fmt.Errorf("completed idempotency record missing status code")
	}
	return &StoredHTTPResponse{
		StatusCode:  int(record.StatusCode.Int64),
		Body:        record.ResponseBody.String,
		ContentType: record.ResponseContentType.String,
	}, nil
}

func (r *ReliabilityRepository) CompleteIdempotentRequest(ctx context.Context, userID, key string, response StoredHTTPResponse) error {
	now := r.clock.Now()
	_, err := r.db.ExecContext(ctx, `UPDATE idempotency_keys
		SET state = 'completed', status_code = $1, response_body = $2, response_content_type = $3,
			updated_at = $4, completed_at = $5
		WHERE user_id = $6 AND idempotency_key = $7`,
		response.StatusCode, response.Body, response.ContentType, now, now, userID, key)
	return err
}

func (r *ReliabilityRepository) AbortIdempotentRequest(ctx context.Context, userID, key string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM idempotency_keys WHERE user_id = $1 AND idempotency_key = $2 AND state = 'processing'`, userID, key)
	return err
}

func (r *ReliabilityRepository) GetIdempotencyRecord(ctx context.Context, userID, key string) (*IdempotencyRecord, error) {
	var record IdempotencyRecord
	if err := r.db.GetContext(ctx, &record, `SELECT id, user_id, idempotency_key, method, path, body_hash, state,
		status_code, response_body, response_content_type, created_at, updated_at, completed_at
		FROM idempotency_keys WHERE user_id = $1 AND idempotency_key = $2`, userID, key); err != nil {
		return nil, err
	}
	return &record, nil
}
