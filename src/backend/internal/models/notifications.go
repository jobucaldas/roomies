package models

import "time"

type NotificationPreferences struct {
	HouseID               string `db:"house_id" json:"house_id"`
	UserID                string `db:"user_id" json:"user_id"`
	ExpenseCreatedEnabled bool   `db:"expense_created_enabled" json:"expense_created_enabled"`
	ReminderEnabled       bool   `db:"reminder_enabled" json:"reminder_enabled"`
	Cadence               string `db:"cadence" json:"cadence"`
	Timezone              string `db:"timezone" json:"timezone"`
	QuietStartMinutes     *int   `db:"quiet_start_minutes" json:"quiet_start_minutes,omitempty"`
	QuietEndMinutes       *int   `db:"quiet_end_minutes" json:"quiet_end_minutes,omitempty"`
	DigestMinutes         int    `db:"digest_minutes" json:"digest_minutes"`
}

type NotificationSubscription struct {
	ID          string     `db:"id" json:"id"`
	HouseID     string     `db:"house_id" json:"house_id"`
	UserID      string     `db:"user_id" json:"user_id"`
	Platform    string     `db:"platform" json:"platform"`
	DeviceLabel string     `db:"device_label" json:"device_label"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	LastSeenAt  time.Time  `db:"last_seen_at" json:"last_seen_at"`
	RevokedAt   *time.Time `db:"revoked_at" json:"revoked_at,omitempty"`
}

type CreateNotificationSubscriptionRequest struct {
	Platform    string `json:"platform"`
	DeviceLabel string `json:"device_label"`
	Endpoint    string `json:"endpoint,omitempty"`
	P256DH      string `json:"p256dh,omitempty"`
	Auth        string `json:"auth,omitempty"`
	Token       string `json:"token,omitempty"`
}

type ScheduledHouseEvent struct {
	ID               string     `db:"id" json:"id"`
	HouseID          string     `db:"house_id" json:"house_id"`
	CreatorID        string     `db:"creator_id" json:"creator_id"`
	Title            string     `db:"title" json:"title"`
	Timezone         string     `db:"timezone" json:"timezone"`
	DTStartLocal     string     `db:"dtstart_local" json:"dtstart_local"`
	RRule            string     `db:"rrule" json:"rrule"`
	ExDatesJSON      string     `db:"exdates" json:"-"`
	ExDates          []string   `db:"-" json:"exdates"`
	NextOccurrenceAt *time.Time `db:"next_occurrence_at" json:"next_occurrence_at,omitempty"`
	Enabled          bool       `db:"enabled" json:"enabled"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
}

type ScheduledEventRequest struct {
	Title        string   `json:"title"`
	Timezone     string   `json:"timezone"`
	DTStartLocal string   `json:"dtstart_local"`
	RRule        string   `json:"rrule"`
	ExDates      []string `json:"exdates"`
	Enabled      *bool    `json:"enabled,omitempty"`
}
