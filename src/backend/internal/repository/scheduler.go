package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/roomies/backend/internal/models"
	"github.com/roomies/backend/internal/recurrence"
)

var ErrScheduleLeaseLost = errors.New("scheduled event lease lost")

type ScheduledEventLease struct {
	ID           string    `db:"id"`
	HouseID      string    `db:"house_id"`
	Timezone     string    `db:"timezone"`
	DTSTARTLocal string    `db:"dtstart_local"`
	RRule        string    `db:"rrule"`
	ExDatesJSON  string    `db:"exdates"`
	Next         time.Time `db:"next_occurrence_at"`
	Generation   int64     `db:"lease_generation"`
}

// RunDueScheduledEvent claims at most one occurrence. Both owner and monotonically
// increasing generation fence completion, so an expired worker cannot advance it.
func (r *NotificationRepository) RunDueScheduledEvent(ctx context.Context, owner string, leaseDuration time.Duration) (bool, error) {
	now := r.clock.Now()
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	query := `SELECT id,house_id,timezone,dtstart_local,rrule,exdates,next_occurrence_at,lease_generation FROM scheduled_house_events WHERE enabled=TRUE AND next_occurrence_at<=$1 AND (lease_expires_at IS NULL OR lease_expires_at<=$1) ORDER BY next_occurrence_at,id LIMIT 1`
	if r.db.DriverName() == "postgres" {
		query += " FOR UPDATE SKIP LOCKED"
	}
	var event ScheduledEventLease
	if err := tx.GetContext(ctx, &event, query, now); errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE scheduled_house_events SET lease_owner=$1,lease_expires_at=$2,lease_generation=lease_generation+1,updated_at=$3 WHERE id=$4 AND lease_generation=$5 AND (lease_expires_at IS NULL OR lease_expires_at<=$3)`, owner, now.Add(leaseDuration), now, event.ID, event.Generation)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return false, nil
	}
	event.Generation++
	var exdates []string
	if err := json.Unmarshal([]byte(event.ExDatesJSON), &exdates); err != nil {
		return false, err
	}
	spec := recurrence.Spec{Timezone: event.Timezone, DTStartLocal: event.DTSTARTLocal, Rule: event.RRule, ExDates: exdates}
	next, err := recurrence.Next(spec, event.Next)
	if err != nil {
		return false, err
	}
	if err := r.EnqueueTx(ctx, tx, event.HouseID, "reminder", "scheduled_event", event.ID, event.Next); err != nil {
		return false, err
	}
	payload, _ := json.Marshal(map[string]any{"category": "reminder", "occurrence_at": event.Next.UTC()})
	_, err = tx.ExecContext(ctx, `INSERT INTO house_events(id,house_id,event_type,resource_type,resource_id,payload,created_at) VALUES($1,$2,'scheduled_event.due','scheduled_event',$3,$4,$5)`, models.NewID(), event.HouseID, event.ID, string(payload), now)
	if err != nil {
		return false, err
	}
	result, err = tx.ExecContext(ctx, `UPDATE scheduled_house_events SET next_occurrence_at=$1,enabled=$2,lease_owner=NULL,lease_expires_at=NULL,updated_at=$3 WHERE id=$4 AND lease_owner=$5 AND lease_generation=$6 AND lease_expires_at>$3`, next, next != nil, now, event.ID, owner, event.Generation)
	if err != nil {
		return false, err
	}
	n, _ = result.RowsAffected()
	if n != 1 {
		return false, ErrScheduleLeaseLost
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}
