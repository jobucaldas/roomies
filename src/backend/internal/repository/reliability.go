package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	roomiesclock "github.com/roomies/backend/internal/clock"
	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/providers"
)

var (
	ErrInvitationPendingExists    = errors.New("pending invitation already exists")
	ErrInvitationUnavailable      = errors.New("invitation unavailable")
	ErrInvitationStateConflict    = errors.New("invitation state change rejected")
	ErrHouseMembershipUnavailable = errors.New("active house membership is required")
)

type ReliabilityRepository struct {
	db    *sqlx.DB
	clock roomiesclock.Clock
}

type CreateInvitationParams struct {
	HouseID   string
	ActorID   string
	Email     string
	Role      string
	ExpiresAt time.Time
}

type jobPayload struct {
	OutboxMessageID string `json:"outbox_message_id"`
}

type OutboxMessage struct {
	ID           string         `db:"id"`
	Topic        string         `db:"topic"`
	DedupeKey    string         `db:"dedupe_key"`
	Payload      string         `db:"payload"`
	Status       string         `db:"status"`
	AvailableAt  time.Time      `db:"available_at"`
	DispatchedAt sql.NullTime   `db:"dispatched_at"`
	LastError    sql.NullString `db:"last_error"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

func NewReliabilityRepository(db *sqlx.DB, clk roomiesclock.Clock) *ReliabilityRepository {
	if clk == nil {
		clk = roomiesclock.RealClock{}
	}
	return &ReliabilityRepository{db: db, clock: clk}
}

func (r *ReliabilityRepository) CreateInvitation(ctx context.Context, params CreateInvitationParams) (*models.HouseInvitation, string, error) {
	now := r.clock.Now()
	token := models.NewID()
	tokenHash := hashToken(token)
	invite := &models.HouseInvitation{
		ID:        models.NewID(),
		HouseID:   params.HouseID,
		Email:     strings.ToLower(strings.TrimSpace(params.Email)),
		Role:      params.Role,
		Status:    "pending",
		CreatedBy: params.ActorID,
		CreatedAt: now,
		ExpiresAt: params.ExpiresAt.UTC(),
	}
	if invite.Email == "" {
		return nil, "", fmt.Errorf("email is required")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback()

	if err := r.lockHouse(ctx, tx, invite.HouseID); err != nil {
		return nil, "", err
	}
	if err := r.expirePendingInvitationsTx(ctx, tx, invite.HouseID, now); err != nil {
		return nil, "", err
	}
	var existingID string
	if err := tx.GetContext(ctx, &existingID, `SELECT id FROM house_invitations
		WHERE house_id = $1 AND email_normalized = $2 AND status = 'pending'
		LIMIT 1`, invite.HouseID, invite.Email); err == nil {
		return nil, "", ErrInvitationPendingExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, "", err
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO house_invitations
		(id, house_id, email_normalized, role, token_hash, status, created_by, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		invite.ID, invite.HouseID, invite.Email, invite.Role, tokenHash, invite.Status, invite.CreatedBy, invite.CreatedAt, invite.ExpiresAt); err != nil {
		if isUniqueViolation(err) {
			return nil, "", ErrInvitationPendingExists
		}
		return nil, "", err
	}

	notification := providers.InvitationNotification{
		Topic:        "house.invitation.created",
		InvitationID: invite.ID,
		HouseID:      invite.HouseID,
		Email:        invite.Email,
		Role:         invite.Role,
		Status:       invite.Status,
		OccurredAt:   now,
	}
	if err := r.enqueueInvitationNotificationTx(ctx, tx, invite.ID, notification); err != nil {
		return nil, "", err
	}
	if err := r.insertAuditEventTx(ctx, tx, now, auditEventParams{
		ActorID:    &invite.CreatedBy,
		HouseID:    &invite.HouseID,
		Action:     "house.invitation.created",
		TargetType: "house_invitation",
		TargetID:   invite.ID,
		Metadata: map[string]string{
			"email": invite.Email,
			"role":  invite.Role,
		},
	}); err != nil {
		return nil, "", err
	}
	if err := r.insertHouseEventTx(ctx, tx, now, houseEventParams{
		HouseID:      invite.HouseID,
		EventType:    "house.invitation.created",
		ActorID:      &invite.CreatedBy,
		ResourceType: "house_invitation",
		ResourceID:   invite.ID,
		Payload: map[string]any{
			"role":   invite.Role,
			"status": invite.Status,
		},
	}); err != nil {
		return nil, "", err
	}

	if err := tx.Commit(); err != nil {
		return nil, "", err
	}
	return invite, token, nil
}

func (r *ReliabilityRepository) ListInvitations(ctx context.Context, houseID string) ([]models.HouseInvitation, error) {
	if err := r.expirePendingInvitations(ctx, houseID); err != nil {
		return nil, err
	}
	var invites []models.HouseInvitation
	err := r.db.SelectContext(ctx, &invites, `SELECT id, house_id, email_normalized, role, status, created_by,
		created_at, expires_at, accepted_by, accepted_at, revoked_by, revoked_at
		FROM house_invitations WHERE house_id = $1 ORDER BY created_at DESC`, houseID)
	return invites, err
}

func (r *ReliabilityRepository) RevokeInvitation(ctx context.Context, houseID, inviteID, actorID string) (*models.HouseInvitation, error) {
	now := r.clock.Now()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := r.lockHouse(ctx, tx, houseID); err != nil {
		return nil, err
	}
	invite, err := r.getInvitationForUpdateTx(ctx, tx, "house_id = $1 AND id = $2", houseID, inviteID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvitationUnavailable
		}
		return nil, err
	}
	if invite.Status == "revoked" {
		return invite, tx.Commit()
	}
	if invite.Status == "accepted" {
		return nil, ErrInvitationStateConflict
	}
	if invite.Status == "pending" && invite.ExpiresAt.Before(now) {
		if err := r.markInvitationExpiredTx(ctx, tx, invite.ID); err != nil {
			return nil, err
		}
		return nil, ErrInvitationStateConflict
	}
	if invite.Status == "expired" {
		return nil, ErrInvitationStateConflict
	}

	if _, err := tx.ExecContext(ctx, `UPDATE house_invitations
		SET status = 'revoked', revoked_by = $1, revoked_at = $2
		WHERE id = $3 AND house_id = $4 AND status = 'pending'`, actorID, now, invite.ID, houseID); err != nil {
		return nil, err
	}
	invite.Status = "revoked"
	invite.RevokedBy = &actorID
	invite.RevokedAt = timePointer(now)

	notification := providers.InvitationNotification{
		Topic:        "house.invitation.revoked",
		InvitationID: invite.ID,
		HouseID:      invite.HouseID,
		Email:        invite.Email,
		Role:         invite.Role,
		Status:       invite.Status,
		OccurredAt:   now,
	}
	if err := r.enqueueInvitationNotificationTx(ctx, tx, invite.ID+":revoke", notification); err != nil {
		return nil, err
	}
	if err := r.insertAuditEventTx(ctx, tx, now, auditEventParams{
		ActorID:    &actorID,
		HouseID:    &houseID,
		Action:     "house.invitation.revoked",
		TargetType: "house_invitation",
		TargetID:   invite.ID,
		Metadata: map[string]string{
			"email": invite.Email,
			"role":  invite.Role,
		},
	}); err != nil {
		return nil, err
	}
	if err := r.insertHouseEventTx(ctx, tx, now, houseEventParams{
		HouseID:      houseID,
		EventType:    "house.invitation.revoked",
		ActorID:      &actorID,
		ResourceType: "house_invitation",
		ResourceID:   invite.ID,
		Payload: map[string]any{
			"role":   invite.Role,
			"status": invite.Status,
		},
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return invite, nil
}

func (r *ReliabilityRepository) AcceptInvitation(ctx context.Context, token, actorID, actorEmail string) (*models.HouseInvitation, error) {
	now := r.clock.Now()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	houseID, err := r.getInvitationHouseIDByTokenTx(ctx, tx, hashToken(token))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvitationUnavailable
		}
		return nil, err
	}
	if err := r.lockHouse(ctx, tx, houseID); err != nil {
		return nil, err
	}
	// Re-read after locking the house so accept and revoke always lock in the
	// same order and all invitation state is revalidated under both locks.
	invite, err := r.getInvitationForUpdateTx(ctx, tx, "token_hash = $1", hashToken(token))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvitationUnavailable
		}
		return nil, err
	}
	actorEmail = strings.ToLower(strings.TrimSpace(actorEmail))
	if invite.Email != actorEmail {
		return nil, ErrInvitationUnavailable
	}
	if invite.Status == "accepted" {
		if invite.AcceptedBy != nil && *invite.AcceptedBy == actorID {
			if err := tx.Commit(); err != nil {
				return nil, err
			}
			return invite, nil
		}
		return nil, ErrInvitationUnavailable
	}
	if invite.Status == "revoked" || invite.Status == "expired" {
		return nil, ErrInvitationUnavailable
	}
	if invite.ExpiresAt.Before(now) {
		if err := r.markInvitationExpiredTx(ctx, tx, invite.ID); err != nil {
			return nil, err
		}
		return nil, ErrInvitationUnavailable
	}

	memberID := models.NewID()
	if _, err := tx.ExecContext(ctx, `INSERT INTO house_members (id, house_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (house_id, user_id) DO NOTHING`, memberID, invite.HouseID, actorID, invite.Role, now); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE house_invitations
		SET status = 'accepted', accepted_by = $1, accepted_at = $2
		WHERE id = $3 AND status = 'pending'`, actorID, now, invite.ID); err != nil {
		return nil, err
	}
	invite.Status = "accepted"
	invite.AcceptedBy = &actorID
	invite.AcceptedAt = timePointer(now)

	notification := providers.InvitationNotification{
		Topic:        "house.invitation.accepted",
		InvitationID: invite.ID,
		HouseID:      invite.HouseID,
		Email:        invite.Email,
		Role:         invite.Role,
		Status:       invite.Status,
		OccurredAt:   now,
		Metadata: map[string]string{
			"accepted_by": actorID,
		},
	}
	if err := r.enqueueInvitationNotificationTx(ctx, tx, invite.ID+":accepted", notification); err != nil {
		return nil, err
	}
	if err := r.insertAuditEventTx(ctx, tx, now, auditEventParams{
		ActorID:    &actorID,
		HouseID:    &invite.HouseID,
		Action:     "house.invitation.accepted",
		TargetType: "house_invitation",
		TargetID:   invite.ID,
		Metadata: map[string]string{
			"email": invite.Email,
			"role":  invite.Role,
		},
	}); err != nil {
		return nil, err
	}
	if err := r.insertHouseEventTx(ctx, tx, now, houseEventParams{
		HouseID:      invite.HouseID,
		EventType:    "house.invitation.accepted",
		ActorID:      &actorID,
		ResourceType: "house_invitation",
		ResourceID:   invite.ID,
		Payload: map[string]any{
			"role":   invite.Role,
			"status": invite.Status,
		},
	}); err != nil {
		return nil, err
	}
	if err := r.insertHouseEventTx(ctx, tx, now, houseEventParams{
		HouseID:      invite.HouseID,
		EventType:    "house.member.joined",
		ActorID:      &actorID,
		ResourceType: "house_member",
		ResourceID:   actorID,
		Payload: map[string]any{
			"role": invite.Role,
		},
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return invite, nil
}

func (r *ReliabilityRepository) AppendHouseEvent(ctx context.Context, houseID, eventType string, actorID *string, resourceType, resourceID string, payload map[string]any) error {
	return r.insertHouseEvent(ctx, r.clock.Now(), houseEventParams{
		HouseID:      houseID,
		EventType:    eventType,
		ActorID:      actorID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Payload:      payload,
	})
}

func (r *ReliabilityRepository) ListHouseEvents(ctx context.Context, houseID string, afterCursor int64, limit int) ([]models.HouseEvent, string, error) {
	return listHouseEvents(ctx, r.db, houseID, afterCursor, limit)
}

// WithAuthorizedHouseEvents serializes the membership authorization, event snapshot,
// and callback against member removal by locking the house row. The callback must make
// only bounded work (SSE writes use a deadline); it runs before the lock is released.
// Its successful return is the authorization linearization point: a snapshot begun
// after a removal commit cannot be authorized, while bytes already sent cannot be
// recalled.
func (r *ReliabilityRepository) WithAuthorizedHouseEvents(ctx context.Context, houseID, userID string, afterCursor int64, limit int, send func([]models.HouseEvent) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := r.lockHouseForEventDelivery(ctx, tx, houseID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrHouseMembershipUnavailable
		}
		return err
	}
	var memberCount int
	if err := tx.GetContext(ctx, &memberCount, `SELECT COUNT(*) FROM house_members WHERE house_id = $1 AND user_id = $2`, houseID, userID); err != nil {
		return err
	}
	if memberCount == 0 {
		return ErrHouseMembershipUnavailable
	}
	events, _, err := listHouseEvents(ctx, tx, houseID, afterCursor, limit)
	if err != nil {
		return err
	}
	if send != nil {
		if err := send(events); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *ReliabilityRepository) lockHouseForEventDelivery(ctx context.Context, tx *sqlx.Tx, houseID string) error {
	if r.db.DriverName() == "sqlite3" {
		// SQLite has no row-level SELECT FOR UPDATE. This no-op update takes its
		// writer lock before authorization so removal cannot commit during send.
		result, err := tx.ExecContext(ctx, `UPDATE houses SET name = name WHERE id = $1`, houseID)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return sql.ErrNoRows
		}
		return nil
	}
	return r.lockHouse(ctx, tx, houseID)
}

type houseEventRow struct {
	Cursor       int64     `db:"cursor"`
	EventType    string    `db:"event_type"`
	ActorID      *string   `db:"actor_id"`
	ResourceType string    `db:"resource_type"`
	ResourceID   string    `db:"resource_id"`
	Payload      string    `db:"payload"`
	CreatedAt    time.Time `db:"created_at"`
}

func listHouseEvents(ctx context.Context, queryer sqlx.QueryerContext, houseID string, afterCursor int64, limit int) ([]models.HouseEvent, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	var rows []houseEventRow
	if err := sqlx.SelectContext(ctx, queryer, &rows, `SELECT cursor, event_type, actor_id, resource_type, resource_id, payload, created_at
		FROM house_events WHERE house_id = $1 AND cursor > $2 ORDER BY cursor ASC LIMIT $3`, houseID, afterCursor, limit); err != nil {
		return nil, "", err
	}
	result := make([]models.HouseEvent, 0, len(rows))
	nextCursor := fmt.Sprintf("%d", afterCursor)
	for _, row := range rows {
		payload := map[string]any{}
		if err := json.Unmarshal([]byte(row.Payload), &payload); err != nil {
			return nil, "", err
		}
		event := models.HouseEvent{
			Cursor:       fmt.Sprintf("%d", row.Cursor),
			EventType:    row.EventType,
			ResourceType: row.ResourceType,
			ResourceID:   row.ResourceID,
			Payload:      payload,
			CreatedAt:    row.CreatedAt,
		}
		if row.ActorID != nil {
			event.ActorID = *row.ActorID
		}
		result = append(result, event)
		nextCursor = event.Cursor
	}
	return result, nextCursor, nil
}

func (r *ReliabilityRepository) ListAuditEvents(ctx context.Context, houseID string) ([]models.AuditEvent, error) {
	var events []models.AuditEvent
	err := r.db.SelectContext(ctx, &events, `SELECT id, actor_id, house_id, action, target_type, target_id, metadata, created_at
		FROM audit_events WHERE house_id = $1 ORDER BY created_at ASC`, houseID)
	return events, err
}

func (r *ReliabilityRepository) expirePendingInvitations(ctx context.Context, houseID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE house_invitations SET status = 'expired'
		WHERE house_id = $1 AND status = 'pending' AND expires_at < $2`, houseID, r.clock.Now())
	return err
}

func (r *ReliabilityRepository) expirePendingInvitationsTx(ctx context.Context, tx *sqlx.Tx, houseID string, now time.Time) error {
	_, err := tx.ExecContext(ctx, `UPDATE house_invitations SET status = 'expired'
		WHERE house_id = $1 AND status = 'pending' AND expires_at < $2`, houseID, now)
	return err
}

func (r *ReliabilityRepository) markInvitationExpiredTx(ctx context.Context, tx *sqlx.Tx, inviteID string) error {
	_, err := tx.ExecContext(ctx, `UPDATE house_invitations SET status = 'expired'
		WHERE id = $1 AND status = 'pending'`, inviteID)
	return err
}

func (r *ReliabilityRepository) getInvitationHouseIDByTokenTx(ctx context.Context, tx *sqlx.Tx, tokenHash string) (string, error) {
	var houseID string
	err := tx.GetContext(ctx, &houseID, `SELECT house_id FROM house_invitations WHERE token_hash = $1`, tokenHash)
	return houseID, err
}

func (r *ReliabilityRepository) getInvitationForUpdateTx(ctx context.Context, tx *sqlx.Tx, where string, args ...any) (*models.HouseInvitation, error) {
	query := `SELECT id, house_id, email_normalized, role, status, created_by, created_at, expires_at,
		accepted_by, accepted_at, revoked_by, revoked_at FROM house_invitations WHERE ` + where
	if r.db.DriverName() == "postgres" {
		query += ` FOR UPDATE`
	}
	var invite models.HouseInvitation
	if err := tx.GetContext(ctx, &invite, query, args...); err != nil {
		return nil, err
	}
	return &invite, nil
}

type auditEventParams struct {
	ActorID    *string
	HouseID    *string
	Action     string
	TargetType string
	TargetID   string
	Metadata   map[string]string
}

type houseEventParams struct {
	HouseID      string
	EventType    string
	ActorID      *string
	ResourceType string
	ResourceID   string
	Payload      map[string]any
}

func (r *ReliabilityRepository) insertAuditEventTx(ctx context.Context, tx *sqlx.Tx, now time.Time, params auditEventParams) error {
	metadata := params.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}
	body, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_events
		(id, actor_id, house_id, action, target_type, target_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		models.NewID(), nullableString(params.ActorID), nullableString(params.HouseID),
		params.Action, params.TargetType, params.TargetID, string(body), now)
	return err
}

func (r *ReliabilityRepository) insertHouseEvent(ctx context.Context, now time.Time, params houseEventParams) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := r.insertHouseEventTx(ctx, tx, now, params); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ReliabilityRepository) insertHouseEventTx(ctx context.Context, tx *sqlx.Tx, now time.Time, params houseEventParams) error {
	payload := params.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO house_events
		(id, house_id, event_type, actor_id, resource_type, resource_id, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		models.NewID(), params.HouseID, params.EventType, nullableString(params.ActorID), params.ResourceType, params.ResourceID, string(body), now)
	return err
}

func (r *ReliabilityRepository) enqueueInvitationNotificationTx(ctx context.Context, tx *sqlx.Tx, dedupeSuffix string, notification providers.InvitationNotification) error {
	payload, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	messageID := models.NewID()
	now := r.clock.Now()
	dedupeKey := "outbox:invitation:" + dedupeSuffix
	if _, err := tx.ExecContext(ctx, `INSERT INTO outbox_messages
		(id, topic, dedupe_key, payload, status, available_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'pending', $5, $6, $7)`,
		messageID, notification.Topic, dedupeKey, string(payload), now, now, now); err != nil {
		return err
	}
	jobBody, err := json.Marshal(jobPayload{OutboxMessageID: messageID})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO durable_jobs
		(id, kind, dedupe_key, payload, available_at, attempts, max_attempts, created_at, updated_at)
		VALUES ($1, 'dispatch_outbox_message', $2, $3, $4, 0, 5, $5, $6)`,
		models.NewID(), "job:"+dedupeKey, string(jobBody), now, now, now)
	return err
}

func (r *ReliabilityRepository) lockHouse(ctx context.Context, tx *sqlx.Tx, houseID string) error {
	query := `SELECT id FROM houses WHERE id = $1`
	if r.db.DriverName() == "postgres" {
		query += ` FOR UPDATE`
	}
	var id string
	return tx.GetContext(ctx, &id, query, houseID)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func timePointer(value time.Time) *time.Time {
	copyValue := value
	return &copyValue
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint") || strings.Contains(message, "duplicate key")
}
