package models

import "time"

type GroceryItem struct {
	ID         string     `db:"id" json:"id"`
	HouseID    string     `db:"house_id" json:"house_id"`
	CreatorID  string     `db:"creator_id" json:"creator_id"`
	Name       string     `db:"name" json:"name"`
	Quantity   string     `db:"quantity" json:"quantity"`
	Unit       string     `db:"unit" json:"unit"`
	Note       string     `db:"note" json:"note"`
	AssigneeID *string    `db:"assignee_id" json:"assignee_id,omitempty"`
	Checked    bool       `db:"checked" json:"checked"`
	CheckedBy  *string    `db:"checked_by" json:"checked_by,omitempty"`
	CheckedAt  *time.Time `db:"checked_at" json:"checked_at,omitempty"`
	Position   int        `db:"position" json:"position"`
	Version    int64      `db:"version" json:"version"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at" json:"updated_at"`
}

type GroceryRequest struct {
	Name       string  `json:"name"`
	Quantity   string  `json:"quantity"`
	Unit       string  `json:"unit"`
	Note       string  `json:"note"`
	AssigneeID *string `json:"assignee_id"`
	Position   int     `json:"position"`
	Version    int64   `json:"version,omitempty"`
}

type GroceryToggleRequest struct {
	Checked bool  `json:"checked"`
	Version int64 `json:"version"`
}

type Chore struct {
	ID          string    `db:"id" json:"id"`
	HouseID     string    `db:"house_id" json:"house_id"`
	CreatorID   string    `db:"creator_id" json:"creator_id"`
	Title       string    `db:"title" json:"title"`
	Description string    `db:"description" json:"description"`
	AssigneeID  *string   `db:"assignee_id" json:"assignee_id,omitempty"`
	Timezone    string    `db:"timezone" json:"timezone"`
	DueLocal    string    `db:"due_local" json:"due_local"`
	RRule       string    `db:"rrule" json:"rrule"`
	ExDatesJSON string    `db:"exdates" json:"-"`
	ExDates     []string  `db:"-" json:"exdates"`
	Enabled     bool      `db:"enabled" json:"enabled"`
	Version     int64     `db:"version" json:"version"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type ChoreRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	AssigneeID  *string  `json:"assignee_id"`
	Timezone    string   `json:"timezone"`
	DueLocal    string   `json:"due_local"`
	RRule       string   `json:"rrule"`
	ExDates     []string `json:"exdates"`
	Enabled     *bool    `json:"enabled,omitempty"`
	Version     int64    `json:"version,omitempty"`
}

type ChoreCompletion struct {
	ID           string    `db:"id" json:"id"`
	ChoreID      string    `db:"chore_id" json:"chore_id"`
	HouseID      string    `db:"house_id" json:"house_id"`
	OccurrenceAt time.Time `db:"occurrence_at" json:"occurrence_at"`
	CompletedBy  string    `db:"completed_by" json:"completed_by"`
	CompletedAt  time.Time `db:"completed_at" json:"completed_at"`
}

type CompleteChoreRequest struct {
	OccurrenceAt time.Time `json:"occurrence_at"`
}

type CalendarEvent struct {
	ID          string    `db:"id" json:"id"`
	HouseID     string    `db:"house_id" json:"house_id"`
	CreatorID   string    `db:"creator_id" json:"creator_id"`
	Title       string    `db:"title" json:"title"`
	Description string    `db:"description" json:"description"`
	Timezone    string    `db:"timezone" json:"timezone"`
	StartLocal  string    `db:"start_local" json:"start_local"`
	EndLocal    string    `db:"end_local" json:"end_local"`
	AllDay      bool      `db:"all_day" json:"all_day"`
	RRule       string    `db:"rrule" json:"rrule"`
	ExDatesJSON string    `db:"exdates" json:"-"`
	ExDates     []string  `db:"-" json:"exdates"`
	Version     int64     `db:"version" json:"version"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type CalendarEventRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Timezone    string   `json:"timezone"`
	StartLocal  string   `json:"start_local"`
	EndLocal    string   `json:"end_local"`
	AllDay      bool     `json:"all_day"`
	RRule       string   `json:"rrule"`
	ExDates     []string `json:"exdates"`
	Version     int64    `json:"version,omitempty"`
}

type ChatMessage struct {
	Cursor     int64      `db:"cursor" json:"cursor"`
	ID         string     `db:"id" json:"id"`
	HouseID    string     `db:"house_id" json:"house_id"`
	AuthorID   string     `db:"author_id" json:"author_id"`
	Body       *string    `db:"body" json:"body,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt  *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
	RedactedAt *time.Time `db:"redacted_at" json:"redacted_at,omitempty"`
}

type ChatMessageRequest struct {
	Body string `json:"body"`
}

type ChatPage struct {
	Messages   []ChatMessage `json:"messages"`
	NextCursor string        `json:"next_cursor,omitempty"`
}
