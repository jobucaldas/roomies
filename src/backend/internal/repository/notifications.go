package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	roomiesclock "github.com/roomies/backend/internal/clock"
	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/providers"
	"github.com/roomies/backend/internal/recurrence"
)

var ErrNotificationNotFound = errors.New("notification resource not found")

type NotificationRepository struct {
	db     *sqlx.DB
	clock  roomiesclock.Clock
	cipher *providers.NotificationDeliveryCipher
}
type subscriptionSecret struct {
	Endpoint string `json:"endpoint,omitempty"`
	P256DH   string `json:"p256dh,omitempty"`
	Auth     string `json:"auth,omitempty"`
	Token    string `json:"token,omitempty"`
}
type NotificationTarget struct {
	ID       string
	Platform string
	Endpoint string
	P256DH   string
	Auth     string
	Token    string
}

type NotificationPayload struct {
	NotificationID string         `json:"notification_id,omitempty"`
	HouseID        string         `json:"house_id"`
	UserID         string         `json:"user_id"`
	Category       string         `json:"category,omitempty"`
	ResourceType   string         `json:"resource_type,omitempty"`
	ResourceID     string         `json:"resource_id,omitempty"`
	Text           string         `json:"text"`
	Digest         bool           `json:"digest,omitempty"`
	CategoryCounts map[string]int `json:"category_counts,omitempty"`
}

type NotificationDispatch func(NotificationTarget, []byte) (revoke bool, err error)

type NotificationDispatchSummary struct {
	InvalidTargetIDs []string
	Dispatched       bool
}

func NewNotificationRepository(db *sqlx.DB, clk roomiesclock.Clock, cipher *providers.NotificationDeliveryCipher) *NotificationRepository {
	if clk == nil {
		clk = roomiesclock.RealClock{}
	}
	return &NotificationRepository{db: db, clock: clk, cipher: cipher}
}

func (r *NotificationRepository) GetPreferences(ctx context.Context, houseID, userID string) (*models.NotificationPreferences, error) {
	var p models.NotificationPreferences
	err := r.db.GetContext(ctx, &p, `SELECT house_id,user_id,expense_created_enabled,reminder_enabled,cadence,timezone,quiet_start_minutes,quiet_end_minutes,digest_minutes FROM notification_preferences WHERE house_id=$1 AND user_id=$2`, houseID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return &models.NotificationPreferences{HouseID: houseID, UserID: userID, ExpenseCreatedEnabled: true, ReminderEnabled: true, Cadence: "immediate", Timezone: "UTC", DigestMinutes: 480}, nil
	}
	return &p, err
}
func (r *NotificationRepository) PutPreferences(ctx context.Context, p models.NotificationPreferences) error {
	_, err := time.LoadLocation(p.Timezone)
	if err != nil {
		return errors.New("invalid IANA timezone")
	}
	if p.Cadence != "immediate" && p.Cadence != "daily_digest" {
		return errors.New("invalid cadence")
	}
	if (p.QuietStartMinutes == nil) != (p.QuietEndMinutes == nil) {
		return errors.New("quiet hours require both start and end")
	}
	valid := func(v int) bool { return v >= 0 && v < 1440 }
	if p.QuietStartMinutes != nil && (!valid(*p.QuietStartMinutes) || !valid(*p.QuietEndMinutes)) {
		return errors.New("quiet hour minutes must be 0..1439")
	}
	if !valid(p.DigestMinutes) {
		return errors.New("digest minutes must be 0..1439")
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO notification_preferences(house_id,user_id,expense_created_enabled,reminder_enabled,cadence,timezone,quiet_start_minutes,quiet_end_minutes,digest_minutes,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(house_id,user_id) DO UPDATE SET expense_created_enabled=excluded.expense_created_enabled,reminder_enabled=excluded.reminder_enabled,cadence=excluded.cadence,timezone=excluded.timezone,quiet_start_minutes=excluded.quiet_start_minutes,quiet_end_minutes=excluded.quiet_end_minutes,digest_minutes=excluded.digest_minutes,updated_at=excluded.updated_at`, p.HouseID, p.UserID, p.ExpenseCreatedEnabled, p.ReminderEnabled, p.Cadence, p.Timezone, p.QuietStartMinutes, p.QuietEndMinutes, p.DigestMinutes, r.clock.Now())
	return err
}

func (r *NotificationRepository) CreateSubscription(ctx context.Context, houseID, userID string, req models.CreateNotificationSubscriptionRequest) (*models.NotificationSubscription, error) {
	if r.cipher == nil {
		return nil, errors.New("notification delivery encryption is not configured")
	}
	identity := ""
	secret := subscriptionSecret{}
	switch req.Platform {
	case "web_push":
		if req.Endpoint == "" || req.P256DH == "" || req.Auth == "" {
			return nil, errors.New("web push endpoint and keys are required")
		}
		endpoint, err := url.Parse(req.Endpoint)
		if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" || endpoint.User != nil || len(req.Endpoint) > 2048 || len(req.P256DH) > 256 || len(req.Auth) > 256 {
			return nil, errors.New("web push subscription is invalid")
		}
		identity = req.Endpoint
		secret = subscriptionSecret{Endpoint: req.Endpoint, P256DH: req.P256DH, Auth: req.Auth}
	case "android_fcm":
		if req.Token == "" || len(req.Token) > 4096 {
			return nil, errors.New("valid FCM token is required")
		}
		identity = req.Token
		secret.Token = req.Token
	default:
		return nil, errors.New("platform must be web_push or android_fcm")
	}
	sum := sha256.Sum256([]byte(identity))
	hash := hex.EncodeToString(sum[:])
	id := models.NewID()
	var existingID string
	if err := r.db.GetContext(ctx, &existingID, `SELECT id FROM notification_subscriptions WHERE house_id=$1 AND user_id=$2 AND platform=$3 AND identity_hash=$4`, houseID, userID, req.Platform, hash); err == nil {
		id = existingID
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	encrypted, err := r.cipher.Encrypt(id, secret)
	if err != nil {
		return nil, err
	}
	now := r.clock.Now()
	_, err = r.db.ExecContext(ctx, `INSERT INTO notification_subscriptions(id,house_id,user_id,platform,identity_hash,key_id,ciphertext,device_label,created_at,last_seen_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9) ON CONFLICT(house_id,user_id,platform,identity_hash) DO UPDATE SET ciphertext=excluded.ciphertext,key_id=excluded.key_id,device_label=excluded.device_label,last_seen_at=excluded.last_seen_at,revoked_at=NULL`, id, houseID, userID, req.Platform, hash, r.cipher.CurrentKeyID(), encrypted, truncate(strings.TrimSpace(req.DeviceLabel), 120), now)
	if err != nil {
		return nil, err
	}
	var out models.NotificationSubscription
	err = r.db.GetContext(ctx, &out, `SELECT id,house_id,user_id,platform,device_label,created_at,last_seen_at,revoked_at FROM notification_subscriptions WHERE house_id=$1 AND user_id=$2 AND platform=$3 AND identity_hash=$4`, houseID, userID, req.Platform, hash)
	return &out, err
}
func (r *NotificationRepository) ListSubscriptions(ctx context.Context, houseID, userID string) ([]models.NotificationSubscription, error) {
	var out []models.NotificationSubscription
	err := r.db.SelectContext(ctx, &out, `SELECT id,house_id,user_id,platform,device_label,created_at,last_seen_at,revoked_at FROM notification_subscriptions WHERE house_id=$1 AND user_id=$2 ORDER BY created_at`, houseID, userID)
	return out, err
}
func (r *NotificationRepository) RevokeSubscription(ctx context.Context, houseID, userID, id string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := r.lockHouseForNotificationDelivery(ctx, tx, houseID); err != nil {
		return err
	}
	if err := r.lockSubscription(ctx, tx, houseID, userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotificationNotFound
		}
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE notification_subscriptions SET revoked_at=$1,ciphertext='',key_id='' WHERE id=$2 AND house_id=$3 AND user_id=$4 AND revoked_at IS NULL`, r.clock.Now(), id, houseID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotificationNotFound
	}
	return tx.Commit()
}

func (r *NotificationRepository) CreateScheduledEvent(ctx context.Context, event *models.ScheduledHouseEvent) error {
	return r.saveScheduledEvent(ctx, event, false)
}
func (r *NotificationRepository) UpdateScheduledEvent(ctx context.Context, event *models.ScheduledHouseEvent) error {
	return r.saveScheduledEvent(ctx, event, true)
}
func (r *NotificationRepository) saveScheduledEvent(ctx context.Context, event *models.ScheduledHouseEvent, update bool) error {
	spec := recurrence.Spec{Timezone: event.Timezone, DTStartLocal: event.DTStartLocal, Rule: event.RRule, ExDates: event.ExDates}
	if err := recurrence.Validate(spec, r.clock.Now()); err != nil {
		return err
	}
	next, err := recurrence.Next(spec, r.clock.Now().Add(-time.Nanosecond))
	if err != nil {
		return err
	}
	event.NextOccurrenceAt = next
	body, _ := json.Marshal(event.ExDates)
	now := r.clock.Now()
	if update {
		res, err := r.db.ExecContext(ctx, `UPDATE scheduled_house_events SET title=$1,timezone=$2,dtstart_local=$3,rrule=$4,exdates=$5,next_occurrence_at=$6,enabled=$7,updated_at=$8 WHERE id=$9 AND house_id=$10`, event.Title, event.Timezone, event.DTStartLocal, event.RRule, string(body), next, event.Enabled, now, event.ID, event.HouseID)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return ErrNotificationNotFound
		}
		return nil
	}
	event.ID = models.NewID()
	event.CreatedAt = now
	event.UpdatedAt = now
	_, err = r.db.ExecContext(ctx, `INSERT INTO scheduled_house_events(id,house_id,creator_id,title,timezone,dtstart_local,rrule,exdates,next_occurrence_at,enabled,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)`, event.ID, event.HouseID, event.CreatorID, event.Title, event.Timezone, event.DTStartLocal, event.RRule, string(body), next, event.Enabled, now)
	return err
}
func (r *NotificationRepository) GetScheduledEvent(ctx context.Context, houseID, id string) (*models.ScheduledHouseEvent, error) {
	var event models.ScheduledHouseEvent
	err := r.db.GetContext(ctx, &event, `SELECT id,house_id,creator_id,title,timezone,dtstart_local,rrule,exdates,next_occurrence_at,enabled,created_at,updated_at FROM scheduled_house_events WHERE house_id=$1 AND id=$2`, houseID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotificationNotFound
	}
	if err == nil {
		err = json.Unmarshal([]byte(event.ExDatesJSON), &event.ExDates)
	}
	return &event, err
}
func (r *NotificationRepository) ListScheduledEvents(ctx context.Context, houseID string) ([]models.ScheduledHouseEvent, error) {
	var out []models.ScheduledHouseEvent
	err := r.db.SelectContext(ctx, &out, `SELECT id,house_id,creator_id,title,timezone,dtstart_local,rrule,exdates,next_occurrence_at,enabled,created_at,updated_at FROM scheduled_house_events WHERE house_id=$1 ORDER BY created_at`, houseID)
	for i := range out {
		if e := json.Unmarshal([]byte(out[i].ExDatesJSON), &out[i].ExDates); e != nil {
			return nil, e
		}
	}
	return out, err
}
func (r *NotificationRepository) DeleteScheduledEvent(ctx context.Context, houseID, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM scheduled_house_events WHERE house_id=$1 AND id=$2`, houseID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

// WithAuthorizedDispatch holds the house membership and subscription row locks
// through bounded provider dispatch. Provider acceptance is the linearization point:
// accepted pushes cannot be recalled, but after removal or revocation returns no later
// dispatch can use that capability.
func (r *NotificationRepository) WithAuthorizedDispatch(ctx context.Context, requested NotificationPayload, dispatch NotificationDispatch) (NotificationDispatchSummary, error) {
	var summary NotificationDispatchSummary
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return summary, err
	}
	defer tx.Rollback()
	if err := r.lockHouseForNotificationDelivery(ctx, tx, requested.HouseID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return summary, nil
		}
		return summary, err
	}
	var memberCount int
	if err := tx.GetContext(ctx, &memberCount, `SELECT COUNT(*) FROM house_members WHERE house_id=$1 AND user_id=$2`, requested.HouseID, requested.UserID); err != nil {
		return summary, err
	}
	if memberCount == 0 {
		return summary, tx.Commit()
	}
	preferences, err := getPreferencesTx(ctx, tx, requested.HouseID, requested.UserID)
	if err != nil {
		return summary, err
	}
	type deliveryRow struct {
		ID       string `db:"id"`
		Category string `db:"category"`
	}
	var deliveries []deliveryRow
	query := `SELECT id,category FROM notification_deliveries WHERE house_id=$1 AND user_id=$2 AND dispatched_at IS NULL AND id=$3`
	args := []any{requested.HouseID, requested.UserID, requested.NotificationID}
	if requested.Digest {
		query = `SELECT id,category FROM notification_deliveries WHERE house_id=$1 AND user_id=$2 AND cadence='daily_digest' AND dispatched_at IS NULL AND available_at<=$3 ORDER BY id`
		args = []any{requested.HouseID, requested.UserID, r.clock.Now()}
	}
	if r.db.DriverName() == "postgres" {
		query += ` FOR UPDATE`
	}
	if err := tx.SelectContext(ctx, &deliveries, query, args...); err != nil {
		return summary, err
	}
	if len(deliveries) == 0 {
		return summary, tx.Commit()
	}

	counts := map[string]int{}
	ids := make([]string, 0, len(deliveries))
	for _, delivery := range deliveries {
		ids = append(ids, delivery.ID)
		if notificationCategoryEnabled(preferences, delivery.Category) {
			counts[delivery.Category]++
		}
	}
	payload := requested
	if requested.Digest {
		payload.NotificationID = ""
		payload.Category = ""
		payload.ResourceType = ""
		payload.ResourceID = ""
		payload.CategoryCounts = counts
		payload.Text = fmt.Sprintf("You have %d household updates", totalNotificationCount(counts))
	}
	if len(counts) > 0 || !requested.Digest && notificationCategoryEnabled(preferences, requested.Category) {
		body, err := json.Marshal(payload)
		if err != nil {
			return summary, err
		}
		if r.cipher == nil {
			return summary, errors.New("notification delivery encryption is not configured")
		}
		var targets []struct {
			ID         string `db:"id"`
			Platform   string `db:"platform"`
			KeyID      string `db:"key_id"`
			Ciphertext string `db:"ciphertext"`
		}
		targetQuery := `SELECT id,platform,key_id,ciphertext FROM notification_subscriptions WHERE house_id=$1 AND user_id=$2 AND revoked_at IS NULL ORDER BY id`
		if r.db.DriverName() == "postgres" {
			targetQuery += ` FOR UPDATE`
		}
		if err := tx.SelectContext(ctx, &targets, targetQuery, requested.HouseID, requested.UserID); err != nil {
			return summary, err
		}
		for _, row := range targets {
			var secret subscriptionSecret
			if err := r.cipher.Decrypt(row.ID, row.Ciphertext, &secret); err != nil {
				summary.InvalidTargetIDs = append(summary.InvalidTargetIDs, row.ID)
				if err := r.quarantineInvalidTarget(ctx, tx, requested.HouseID, row.ID, row.KeyID); err != nil {
					return summary, err
				}
				continue
			}
			if row.KeyID != r.cipher.CurrentKeyID() {
				rotated, err := r.cipher.Encrypt(row.ID, secret)
				if err != nil {
					return summary, err
				}
				if _, err := tx.ExecContext(ctx, `UPDATE notification_subscriptions SET key_id=$1,ciphertext=$2 WHERE id=$3 AND revoked_at IS NULL`, r.cipher.CurrentKeyID(), rotated, row.ID); err != nil {
					return summary, err
				}
			}
			target := NotificationTarget{ID: row.ID, Platform: row.Platform, Endpoint: secret.Endpoint, P256DH: secret.P256DH, Auth: secret.Auth, Token: secret.Token}
			revoke, err := dispatch(target, body)
			if err != nil {
				return summary, err
			}
			summary.Dispatched = true
			if revoke {
				if _, err := tx.ExecContext(ctx, `UPDATE notification_subscriptions SET revoked_at=$1,ciphertext='',key_id='' WHERE id=$2`, r.clock.Now(), row.ID); err != nil {
					return summary, err
				}
			}
		}
	}
	markQuery, markArgs, err := sqlx.In(`UPDATE notification_deliveries SET dispatched_at=? WHERE id IN (?)`, r.clock.Now(), ids)
	if err != nil {
		return summary, err
	}
	if _, err := tx.ExecContext(ctx, r.db.Rebind(markQuery), markArgs...); err != nil {
		return summary, err
	}
	return summary, tx.Commit()
}

func notificationCategoryEnabled(p models.NotificationPreferences, category string) bool {
	return category == "expense_created" && p.ExpenseCreatedEnabled || category == "reminder" && p.ReminderEnabled
}

func totalNotificationCount(counts map[string]int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}

func (r *NotificationRepository) quarantineInvalidTarget(ctx context.Context, tx *sqlx.Tx, houseID, targetID, keyID string) error {
	if _, err := tx.ExecContext(ctx, `UPDATE notification_subscriptions SET revoked_at=$1,ciphertext='',key_id='' WHERE id=$2`, r.clock.Now(), targetID); err != nil {
		return err
	}
	metadata, _ := json.Marshal(map[string]string{"key_id": keyID, "reason": "decrypt_failed"})
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_events(id,actor_id,house_id,action,target_type,target_id,metadata,created_at) VALUES($1,NULL,$2,'notification.subscription.quarantined','notification_subscription',$3,$4,$5)`, models.NewID(), houseID, targetID, string(metadata), r.clock.Now())
	return err
}

func (r *NotificationRepository) lockHouseForNotificationDelivery(ctx context.Context, tx *sqlx.Tx, houseID string) error {
	if r.db.DriverName() == "sqlite3" {
		result, err := tx.ExecContext(ctx, `UPDATE houses SET name=name WHERE id=$1`, houseID)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil || n == 0 {
			if err != nil {
				return err
			}
			return sql.ErrNoRows
		}
		return nil
	}
	var id string
	return tx.GetContext(ctx, &id, `SELECT id FROM houses WHERE id=$1 FOR UPDATE`, houseID)
}

func (r *NotificationRepository) lockSubscription(ctx context.Context, tx *sqlx.Tx, houseID, userID, id string) error {
	query := `SELECT id FROM notification_subscriptions WHERE house_id=$1 AND user_id=$2 AND id=$3 AND revoked_at IS NULL`
	if r.db.DriverName() == "postgres" {
		query += ` FOR UPDATE`
	}
	var found string
	return tx.GetContext(ctx, &found, query, houseID, userID, id)
}

func (r *NotificationRepository) Enqueue(ctx context.Context, houseID, category, resourceType, resourceID string, occurrence time.Time) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = r.EnqueueTx(ctx, tx, houseID, category, resourceType, resourceID, occurrence); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *NotificationRepository) EnqueueTx(ctx context.Context, tx *sqlx.Tx, houseID, category, resourceType, resourceID string, occurrence time.Time) error {
	var users []string
	enabledColumn := "expense_created_enabled"
	if category == "reminder" {
		enabledColumn = "reminder_enabled"
	}
	if err := tx.SelectContext(ctx, &users, fmt.Sprintf(`SELECT hm.user_id FROM house_members hm LEFT JOIN notification_preferences np ON np.house_id=hm.house_id AND np.user_id=hm.user_id WHERE hm.house_id=$1 AND COALESCE(np.%s,TRUE)=TRUE`, enabledColumn), houseID); err != nil {
		return err
	}
	for _, userID := range users {
		p, err := getPreferencesTx(ctx, tx, houseID, userID)
		if err != nil {
			return err
		}
		deliveryTime := occurrence
		if now := r.clock.Now(); now.After(deliveryTime) {
			deliveryTime = now
		}
		available := notificationAvailableAt(deliveryTime, p)
		id := models.NewID()
		res, err := tx.ExecContext(ctx, `INSERT INTO notification_deliveries(id,house_id,user_id,category,resource_type,resource_id,occurrence_at,cadence,available_at,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(house_id,user_id,category,resource_type,resource_id,occurrence_at) DO NOTHING`, id, houseID, userID, category, resourceType, resourceID, occurrence.UTC(), p.Cadence, available, r.clock.Now())
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			continue
		}
		payload := NotificationPayload{NotificationID: id, HouseID: houseID, UserID: userID, Category: category, ResourceType: resourceType, ResourceID: resourceID, Text: genericText(category), Digest: p.Cadence == "daily_digest"}
		body, _ := json.Marshal(payload)
		// Each row keeps its own durable trigger. At dispatch, a digest trigger
		// snapshots all then-due rows; later triggers become harmless no-ops. This
		// prevents an enqueue concurrent with a snapshot from being stranded behind
		// an already-completed daily dedupe key.
		dedupe := "notification:" + id
		messageID := models.NewID()
		result, err := tx.ExecContext(ctx, `INSERT INTO outbox_messages(id,topic,dedupe_key,payload,status,available_at,created_at,updated_at) VALUES($1,'notification.delivery',$2,$3,'pending',$4,$5,$5) ON CONFLICT(dedupe_key) DO NOTHING`, messageID, dedupe, string(body), available, r.clock.Now())
		if err != nil {
			return err
		}
		inserted, _ := result.RowsAffected()
		if inserted == 0 {
			continue
		}
		jobBody, _ := json.Marshal(jobPayload{OutboxMessageID: messageID})
		_, err = tx.ExecContext(ctx, `INSERT INTO durable_jobs(id,kind,dedupe_key,payload,available_at,attempts,max_attempts,created_at,updated_at) VALUES($1,'dispatch_notification',$2,$3,$4,0,8,$5,$5)`, models.NewID(), "job:"+dedupe, string(jobBody), available, r.clock.Now())
		if err != nil {
			return err
		}
	}
	return nil
}

func getPreferencesTx(ctx context.Context, tx *sqlx.Tx, houseID, userID string) (models.NotificationPreferences, error) {
	var p models.NotificationPreferences
	err := tx.GetContext(ctx, &p, `SELECT house_id,user_id,expense_created_enabled,reminder_enabled,cadence,timezone,quiet_start_minutes,quiet_end_minutes,digest_minutes FROM notification_preferences WHERE house_id=$1 AND user_id=$2`, houseID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return models.NotificationPreferences{HouseID: houseID, UserID: userID, ExpenseCreatedEnabled: true, ReminderEnabled: true, Cadence: "immediate", Timezone: "UTC", DigestMinutes: 480}, nil
	}
	return p, err
}
func notificationAvailableAt(at time.Time, p models.NotificationPreferences) time.Time {
	loc := mustLocation(p.Timezone)
	local := at.In(loc)
	if p.Cadence == "daily_digest" {
		candidate := time.Date(local.Year(), local.Month(), local.Day(), p.DigestMinutes/60, p.DigestMinutes%60, 0, 0, loc)
		if !candidate.After(local) {
			candidate = candidate.AddDate(0, 0, 1)
		}
		return candidate.UTC()
	}
	if p.QuietStartMinutes == nil {
		return at.UTC()
	}
	minute := local.Hour()*60 + local.Minute()
	start, end := *p.QuietStartMinutes, *p.QuietEndMinutes
	quiet := false
	if start < end {
		quiet = minute >= start && minute < end
	} else {
		quiet = minute >= start || minute < end
	}
	if !quiet {
		return at.UTC()
	}
	days := 0
	if start >= end && minute >= start {
		days = 1
	}
	return time.Date(local.Year(), local.Month(), local.Day()+days, end/60, end%60, 0, 0, loc).UTC()
}
func mustLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}
func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func genericText(category string) string {
	if category == "reminder" {
		return "A scheduled household reminder is due"
	}
	return "A shared expense was added"
}
