package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	roomiesclock "github.com/roomies/backend/internal/clock"
	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/recurrence"
)

var (
	ErrHouseholdNotFound = errors.New("household resource not found")
	ErrVersionConflict   = errors.New("resource version conflict")
	ErrInvalidOccurrence = errors.New("occurrence is not part of the chore recurrence")
	ErrAlreadyCompleted  = errors.New("chore occurrence already completed")
)

const chatBodyLimit = 4000

type HouseholdRepository struct {
	db    *sqlx.DB
	clock roomiesclock.Clock
}

func NewHouseholdRepository(db *sqlx.DB, clk roomiesclock.Clock) *HouseholdRepository {
	if clk == nil {
		clk = roomiesclock.RealClock{}
	}
	return &HouseholdRepository{db: db, clock: clk}
}

func (r *HouseholdRepository) ListGroceries(ctx context.Context, houseID string) ([]models.GroceryItem, error) {
	var out []models.GroceryItem
	err := r.db.SelectContext(ctx, &out, `SELECT id,house_id,creator_id,name,quantity,unit,note,assignee_id,checked,checked_by,checked_at,position,version,created_at,updated_at FROM grocery_items WHERE house_id=$1 ORDER BY position,created_at,id`, houseID)
	return out, err
}

func (r *HouseholdRepository) GetGrocery(ctx context.Context, houseID, id string) (*models.GroceryItem, error) {
	var item models.GroceryItem
	err := r.db.GetContext(ctx, &item, `SELECT id,house_id,creator_id,name,quantity,unit,note,assignee_id,checked,checked_by,checked_at,position,version,created_at,updated_at FROM grocery_items WHERE house_id=$1 AND id=$2`, houseID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrHouseholdNotFound
	}
	return &item, err
}

func (r *HouseholdRepository) CreateGrocery(ctx context.Context, item *models.GroceryItem, actorID string) error {
	item.ID, item.CreatorID, item.Version = models.NewID(), actorID, 1
	item.CreatedAt = r.clock.Now()
	item.UpdatedAt = item.CreatedAt
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := ensureActiveAssignee(ctx, tx, item.HouseID, item.AssigneeID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO grocery_items(id,house_id,creator_id,name,quantity,unit,note,assignee_id,position,version,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,1,$10,$10)`, item.ID, item.HouseID, item.CreatorID, item.Name, item.Quantity, item.Unit, item.Note, item.AssigneeID, item.Position, item.CreatedAt)
	if err != nil {
		return err
	}
	if err = r.recordMutation(ctx, tx, item.HouseID, actorID, "grocery.created", "grocery", item.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *HouseholdRepository) UpdateGrocery(ctx context.Context, item *models.GroceryItem, actorID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := ensureActiveAssignee(ctx, tx, item.HouseID, item.AssigneeID); err != nil {
		return err
	}
	now := r.clock.Now()
	res, err := tx.ExecContext(ctx, `UPDATE grocery_items SET name=$1,quantity=$2,unit=$3,note=$4,assignee_id=$5,position=$6,version=version+1,updated_at=$7 WHERE house_id=$8 AND id=$9 AND version=$10`, item.Name, item.Quantity, item.Unit, item.Note, item.AssigneeID, item.Position, now, item.HouseID, item.ID, item.Version)
	if err != nil {
		return err
	}
	if err = requireVersionedRow(ctx, tx, res, "grocery_items", item.HouseID, item.ID); err != nil {
		return err
	}
	item.Version++
	item.UpdatedAt = now
	if err = r.recordMutation(ctx, tx, item.HouseID, actorID, "grocery.updated", "grocery", item.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *HouseholdRepository) ToggleGrocery(ctx context.Context, houseID, id, actorID string, checked bool, version int64) (*models.GroceryItem, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := r.clock.Now()
	var by any
	var at any
	if checked {
		by = actorID
		at = now
	}
	res, err := tx.ExecContext(ctx, `UPDATE grocery_items SET checked=$1,checked_by=$2,checked_at=$3,version=version+1,updated_at=$4 WHERE house_id=$5 AND id=$6 AND version=$7`, checked, by, at, now, houseID, id, version)
	if err != nil {
		return nil, err
	}
	if err = requireVersionedRow(ctx, tx, res, "grocery_items", houseID, id); err != nil {
		return nil, err
	}
	if err = r.recordMutation(ctx, tx, houseID, actorID, "grocery.toggled", "grocery", id); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetGrocery(ctx, houseID, id)
}

func (r *HouseholdRepository) DeleteGrocery(ctx context.Context, houseID, id, actorID string, version int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `DELETE FROM grocery_items WHERE house_id=$1 AND id=$2 AND version=$3`, houseID, id, version)
	if err != nil {
		return err
	}
	if err = requireVersionedRow(ctx, tx, res, "grocery_items", houseID, id); err != nil {
		return err
	}
	if err = r.recordMutation(ctx, tx, houseID, actorID, "grocery.deleted", "grocery", id); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *HouseholdRepository) ListChores(ctx context.Context, houseID string) ([]models.Chore, error) {
	var out []models.Chore
	err := r.db.SelectContext(ctx, &out, `SELECT id,house_id,creator_id,title,description,assignee_id,timezone,due_local,rrule,exdates,enabled,version,created_at,updated_at FROM chores WHERE house_id=$1 ORDER BY created_at,id`, houseID)
	for i := range out {
		if e := json.Unmarshal([]byte(out[i].ExDatesJSON), &out[i].ExDates); e != nil {
			return nil, e
		}
	}
	return out, err
}
func (r *HouseholdRepository) GetChore(ctx context.Context, houseID, id string) (*models.Chore, error) {
	var out models.Chore
	err := r.db.GetContext(ctx, &out, `SELECT id,house_id,creator_id,title,description,assignee_id,timezone,due_local,rrule,exdates,enabled,version,created_at,updated_at FROM chores WHERE house_id=$1 AND id=$2`, houseID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrHouseholdNotFound
	}
	if err == nil {
		err = json.Unmarshal([]byte(out.ExDatesJSON), &out.ExDates)
	}
	return &out, err
}
func (r *HouseholdRepository) SaveChore(ctx context.Context, chore *models.Chore, actorID string, create bool) error {
	if err := recurrence.Validate(recurrence.Spec{Timezone: chore.Timezone, DTStartLocal: chore.DueLocal, Rule: chore.RRule, ExDates: chore.ExDates}, r.clock.Now()); err != nil {
		return err
	}
	body, _ := json.Marshal(chore.ExDates)
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = ensureActiveAssignee(ctx, tx, chore.HouseID, chore.AssigneeID); err != nil {
		return err
	}
	now := r.clock.Now()
	action := "chore.updated"
	if create {
		chore.ID = models.NewID()
		chore.CreatorID = actorID
		chore.Version = 1
		chore.CreatedAt = now
		chore.UpdatedAt = now
		_, err = tx.ExecContext(ctx, `INSERT INTO chores(id,house_id,creator_id,title,description,assignee_id,timezone,due_local,rrule,exdates,enabled,version,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,1,$12,$12)`, chore.ID, chore.HouseID, chore.CreatorID, chore.Title, chore.Description, chore.AssigneeID, chore.Timezone, chore.DueLocal, chore.RRule, string(body), chore.Enabled, now)
		action = "chore.created"
	} else {
		var res sql.Result
		res, err = tx.ExecContext(ctx, `UPDATE chores SET title=$1,description=$2,assignee_id=$3,timezone=$4,due_local=$5,rrule=$6,exdates=$7,enabled=$8,version=version+1,updated_at=$9 WHERE house_id=$10 AND id=$11 AND version=$12`, chore.Title, chore.Description, chore.AssigneeID, chore.Timezone, chore.DueLocal, chore.RRule, string(body), chore.Enabled, now, chore.HouseID, chore.ID, chore.Version)
		if err == nil {
			err = requireVersionedRow(ctx, tx, res, "chores", chore.HouseID, chore.ID)
		}
		chore.Version++
		chore.UpdatedAt = now
	}
	if err != nil {
		return err
	}
	if err = r.recordMutation(ctx, tx, chore.HouseID, actorID, action, "chore", chore.ID); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *HouseholdRepository) DeleteChore(ctx context.Context, houseID, id, actorID string, version int64) error {
	return r.deleteVersioned(ctx, "chores", houseID, id, actorID, version, "chore.deleted", "chore")
}
func (r *HouseholdRepository) CompleteChore(ctx context.Context, chore *models.Chore, actorID string, occurrence time.Time) (*models.ChoreCompletion, error) {
	spec := recurrence.Spec{Timezone: chore.Timezone, DTStartLocal: chore.DueLocal, Rule: chore.RRule, ExDates: chore.ExDates}
	next, err := recurrence.Next(spec, occurrence.Add(-time.Nanosecond))
	if err != nil || next == nil || !next.Equal(occurrence.UTC()) {
		return nil, ErrInvalidOccurrence
	}
	completion := &models.ChoreCompletion{ID: models.NewID(), ChoreID: chore.ID, HouseID: chore.HouseID, OccurrenceAt: occurrence.UTC(), CompletedBy: actorID, CompletedAt: r.clock.Now()}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO chore_completions(id,chore_id,house_id,occurrence_at,completed_by,completed_at) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(chore_id,occurrence_at) DO NOTHING`, completion.ID, completion.ChoreID, completion.HouseID, completion.OccurrenceAt, completion.CompletedBy, completion.CompletedAt)
	if err != nil {
		return nil, err
	}
	var found models.ChoreCompletion
	if err = tx.GetContext(ctx, &found, `SELECT id,chore_id,house_id,occurrence_at,completed_by,completed_at FROM chore_completions WHERE chore_id=$1 AND occurrence_at=$2`, chore.ID, occurrence.UTC()); err != nil {
		return nil, err
	}
	if found.ID != completion.ID {
		return nil, ErrAlreadyCompleted
	}
	if err = r.recordMutation(ctx, tx, chore.HouseID, actorID, "chore.completed", "chore", chore.ID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return completion, nil
}
func (r *HouseholdRepository) ListChoreCompletions(ctx context.Context, houseID, choreID string) ([]models.ChoreCompletion, error) {
	var out []models.ChoreCompletion
	err := r.db.SelectContext(ctx, &out, `SELECT id,chore_id,house_id,occurrence_at,completed_by,completed_at FROM chore_completions WHERE house_id=$1 AND chore_id=$2 ORDER BY occurrence_at DESC`, houseID, choreID)
	return out, err
}

func (r *HouseholdRepository) ListCalendar(ctx context.Context, houseID string) ([]models.CalendarEvent, error) {
	var out []models.CalendarEvent
	err := r.db.SelectContext(ctx, &out, `SELECT id,house_id,creator_id,title,description,timezone,start_local,end_local,all_day,rrule,exdates,version,created_at,updated_at FROM calendar_events WHERE house_id=$1 ORDER BY created_at,id`, houseID)
	for i := range out {
		if e := json.Unmarshal([]byte(out[i].ExDatesJSON), &out[i].ExDates); e != nil {
			return nil, e
		}
	}
	return out, err
}
func (r *HouseholdRepository) GetCalendar(ctx context.Context, houseID, id string) (*models.CalendarEvent, error) {
	var out models.CalendarEvent
	err := r.db.GetContext(ctx, &out, `SELECT id,house_id,creator_id,title,description,timezone,start_local,end_local,all_day,rrule,exdates,version,created_at,updated_at FROM calendar_events WHERE house_id=$1 AND id=$2`, houseID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrHouseholdNotFound
	}
	if err == nil {
		err = json.Unmarshal([]byte(out.ExDatesJSON), &out.ExDates)
	}
	return &out, err
}
func (r *HouseholdRepository) SaveCalendar(ctx context.Context, event *models.CalendarEvent, actorID string, create bool) error {
	spec := recurrence.Spec{Timezone: event.Timezone, DTStartLocal: event.StartLocal, Rule: event.RRule, ExDates: event.ExDates}
	if err := recurrence.Validate(spec, r.clock.Now()); err != nil {
		return err
	}
	loc, _ := time.LoadLocation(event.Timezone)
	start, e1 := time.ParseInLocation("2006-01-02T15:04:05", event.StartLocal, loc)
	end, e2 := time.ParseInLocation("2006-01-02T15:04:05", event.EndLocal, loc)
	if e1 != nil || e2 != nil || !end.After(start) {
		return errors.New("end_local must be after start_local")
	}
	body, _ := json.Marshal(event.ExDates)
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := r.clock.Now()
	action := "calendar.updated"
	if create {
		event.ID = models.NewID()
		event.CreatorID = actorID
		event.Version = 1
		event.CreatedAt = now
		event.UpdatedAt = now
		_, err = tx.ExecContext(ctx, `INSERT INTO calendar_events(id,house_id,creator_id,title,description,timezone,start_local,end_local,all_day,rrule,exdates,version,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,1,$12,$12)`, event.ID, event.HouseID, event.CreatorID, event.Title, event.Description, event.Timezone, event.StartLocal, event.EndLocal, event.AllDay, event.RRule, string(body), now)
		action = "calendar.created"
	} else {
		var res sql.Result
		res, err = tx.ExecContext(ctx, `UPDATE calendar_events SET title=$1,description=$2,timezone=$3,start_local=$4,end_local=$5,all_day=$6,rrule=$7,exdates=$8,version=version+1,updated_at=$9 WHERE house_id=$10 AND id=$11 AND version=$12`, event.Title, event.Description, event.Timezone, event.StartLocal, event.EndLocal, event.AllDay, event.RRule, string(body), now, event.HouseID, event.ID, event.Version)
		if err == nil {
			err = requireVersionedRow(ctx, tx, res, "calendar_events", event.HouseID, event.ID)
		}
		event.Version++
		event.UpdatedAt = now
	}
	if err != nil {
		return err
	}
	if err = r.recordMutation(ctx, tx, event.HouseID, actorID, action, "calendar", event.ID); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *HouseholdRepository) DeleteCalendar(ctx context.Context, houseID, id, actorID string, version int64) error {
	return r.deleteVersioned(ctx, "calendar_events", houseID, id, actorID, version, "calendar.deleted", "calendar")
}

func (r *HouseholdRepository) ListChat(ctx context.Context, houseID string, before int64, limit int) (models.ChatPage, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if before <= 0 {
		before = 1 << 62
	}
	var out []models.ChatMessage
	err := r.db.SelectContext(ctx, &out, `SELECT cursor,id,house_id,author_id,body,created_at,updated_at,deleted_at,redacted_at FROM chat_messages WHERE house_id=$1 AND cursor<$2 ORDER BY cursor DESC LIMIT $3`, houseID, before, limit)
	if out == nil {
		out = []models.ChatMessage{}
	}
	page := models.ChatPage{Messages: out}
	if len(out) == limit {
		page.NextCursor = strconv.FormatInt(out[len(out)-1].Cursor, 10)
	}
	return page, err
}
func (r *HouseholdRepository) GetChat(ctx context.Context, houseID, id string) (*models.ChatMessage, error) {
	var out models.ChatMessage
	err := r.db.GetContext(ctx, &out, `SELECT cursor,id,house_id,author_id,body,created_at,updated_at,deleted_at,redacted_at FROM chat_messages WHERE house_id=$1 AND id=$2`, houseID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrHouseholdNotFound
	}
	return &out, err
}
func (r *HouseholdRepository) CreateChat(ctx context.Context, houseID, actorID, body string) (*models.ChatMessage, error) {
	body = strings.TrimSpace(body)
	if body == "" || len([]rune(body)) > chatBodyLimit {
		return nil, errors.New("body must contain 1 to 4000 characters")
	}
	id := models.NewID()
	now := r.clock.Now()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO chat_messages(id,house_id,author_id,body,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$5)`, id, houseID, actorID, body, now)
	if err != nil {
		return nil, err
	}
	if err = r.recordMutation(ctx, tx, houseID, actorID, "chat.created", "chat", id); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetChat(ctx, houseID, id)
}
func (r *HouseholdRepository) EditChat(ctx context.Context, houseID, id, actorID, body string) (*models.ChatMessage, error) {
	body = strings.TrimSpace(body)
	if body == "" || len([]rune(body)) > chatBodyLimit {
		return nil, errors.New("body must contain 1 to 4000 characters")
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE chat_messages SET body=$1,updated_at=$2 WHERE house_id=$3 AND id=$4 AND author_id=$5 AND deleted_at IS NULL AND redacted_at IS NULL`, body, r.clock.Now(), houseID, id, actorID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrHouseholdNotFound
	}
	if err = r.recordMutation(ctx, tx, houseID, actorID, "chat.updated", "chat", id); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetChat(ctx, houseID, id)
}
func (r *HouseholdRepository) DeleteChat(ctx context.Context, houseID, id, actorID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := r.clock.Now()
	res, err := tx.ExecContext(ctx, `UPDATE chat_messages SET body=NULL,deleted_at=$1,updated_at=$1 WHERE house_id=$2 AND id=$3 AND author_id=$4 AND deleted_at IS NULL`, now, houseID, id, actorID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrHouseholdNotFound
	}
	if err = r.recordMutation(ctx, tx, houseID, actorID, "chat.deleted", "chat", id); err != nil {
		return err
	}
	return tx.Commit()
}

// RunChatRetention claims the singleton durable lease, redacts at most batch bodies,
// and fences completion with the claimed generation so an expired worker cannot
// overwrite a newer lease holder's schedule.
func (r *HouseholdRepository) RunChatRetention(ctx context.Context, owner string, lease time.Duration, batch int) (int, error) {
	if batch <= 0 || batch > 500 {
		batch = 100
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	now := r.clock.Now()
	res, err := tx.ExecContext(ctx, `UPDATE chat_retention_jobs SET lease_owner=$1,lease_expires_at=$2,lease_generation=lease_generation+1,updated_at=$3 WHERE id='chat-body-retention' AND available_at<=$3 AND (lease_expires_at IS NULL OR lease_expires_at<=$3)`, owner, now.Add(lease), now)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return 0, tx.Commit()
	}
	var generation int64
	if err = tx.GetContext(ctx, &generation, `SELECT lease_generation FROM chat_retention_jobs WHERE id='chat-body-retention' AND lease_owner=$1`, owner); err != nil {
		return 0, err
	}
	type retentionTarget struct {
		ID      string `db:"id"`
		HouseID string `db:"house_id"`
	}
	var targets []retentionTarget
	if err = tx.SelectContext(ctx, &targets, `SELECT id,house_id FROM chat_messages WHERE body IS NOT NULL AND created_at<=$1 ORDER BY created_at,id LIMIT $2`, now.AddDate(-1, 0, 0), batch); err != nil {
		return 0, err
	}
	for _, target := range targets {
		if _, err = tx.ExecContext(ctx, `UPDATE chat_messages SET body=NULL,redacted_at=$1,updated_at=$1 WHERE id=$2 AND body IS NOT NULL`, now, target.ID); err != nil {
			return 0, err
		}
		if err = r.recordSystemMutation(ctx, tx, target.HouseID, "chat.redacted", "chat", target.ID); err != nil {
			return 0, err
		}
	}
	next := now.Add(24 * time.Hour)
	if len(targets) == batch {
		next = now
	}
	res, err = tx.ExecContext(ctx, `UPDATE chat_retention_jobs SET available_at=$1,lease_owner=NULL,lease_expires_at=NULL,last_completed_at=$2,updated_at=$2 WHERE id='chat-body-retention' AND lease_owner=$3 AND lease_generation=$4`, next, now, owner, generation)
	if err != nil {
		return 0, err
	}
	n, _ = res.RowsAffected()
	if n == 0 {
		return 0, ErrVersionConflict
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return len(targets), nil
}

func (r *HouseholdRepository) deleteVersioned(ctx context.Context, table, houseID, id, actorID string, version int64, action, resource string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE house_id=$1 AND id=$2 AND version=$3`, houseID, id, version)
	if err != nil {
		return err
	}
	if err = requireVersionedRow(ctx, tx, res, table, houseID, id); err != nil {
		return err
	}
	if err = r.recordMutation(ctx, tx, houseID, actorID, action, resource, id); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *HouseholdRepository) recordMutation(ctx context.Context, tx *sqlx.Tx, houseID, actorID, action, resource, id string) error {
	return r.recordMutationWithActor(ctx, tx, houseID, &actorID, action, resource, id)
}

func (r *HouseholdRepository) recordSystemMutation(ctx context.Context, tx *sqlx.Tx, houseID, action, resource, id string) error {
	return r.recordMutationWithActor(ctx, tx, houseID, nil, action, resource, id)
}

func (r *HouseholdRepository) recordMutationWithActor(ctx context.Context, tx *sqlx.Tx, houseID string, actorID *string, action, resource, id string) error {
	now := r.clock.Now()
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_events(id,actor_id,house_id,action,target_type,target_id,metadata,created_at) VALUES($1,$2,$3,$4,$5,$6,'{}',$7)`, models.NewID(), actorID, houseID, action, resource, id, now)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]string{"category": resource})
	_, err = tx.ExecContext(ctx, `INSERT INTO house_events(id,house_id,event_type,actor_id,resource_type,resource_id,payload,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, models.NewID(), houseID, action, actorID, resource, id, string(payload), now)
	return err
}
func ensureActiveAssignee(ctx context.Context, tx *sqlx.Tx, houseID string, userID *string) error {
	if userID == nil {
		return nil
	}
	var n int
	if err := tx.GetContext(ctx, &n, `SELECT COUNT(*) FROM house_members WHERE house_id=$1 AND user_id=$2`, houseID, *userID); err != nil {
		return err
	}
	if n == 0 {
		return errors.New("assignee must be an active house member")
	}
	return nil
}
func requireVersionedRow(ctx context.Context, tx *sqlx.Tx, res sql.Result, table, houseID, id string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	var exists int
	if err = tx.GetContext(ctx, &exists, `SELECT COUNT(*) FROM `+table+` WHERE house_id=$1 AND id=$2`, houseID, id); err != nil {
		return err
	}
	if exists == 0 {
		return ErrHouseholdNotFound
	}
	return ErrVersionConflict
}
