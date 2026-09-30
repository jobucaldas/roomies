package models

import "time"

type CreateInvitationRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type AcceptInvitationRequest struct {
	Token string `json:"token"`
}

type HouseInvitation struct {
	ID                  string     `db:"id" json:"id"`
	HouseID             string     `db:"house_id" json:"house_id"`
	Email               string     `db:"email_normalized" json:"email"`
	Role                string     `db:"role" json:"role"`
	Status              string     `db:"status" json:"status"`
	CreatedBy           string     `db:"created_by" json:"created_by"`
	CreatedAt           time.Time  `db:"created_at" json:"created_at"`
	ExpiresAt           time.Time  `db:"expires_at" json:"expires_at"`
	AcceptedBy          *string    `db:"accepted_by" json:"accepted_by,omitempty"`
	AcceptedAt          *time.Time `db:"accepted_at" json:"accepted_at,omitempty"`
	RevokedBy           *string    `db:"revoked_by" json:"revoked_by,omitempty"`
	RevokedAt           *time.Time `db:"revoked_at" json:"revoked_at,omitempty"`
	ManualAcceptanceURL string     `json:"manual_acceptance_url,omitempty"`
}

type InvitationAcceptanceResponse struct {
	Invitation HouseInvitation `json:"invitation"`
}

type HouseEvent struct {
	Cursor       string         `json:"cursor"`
	EventType    string         `json:"event_type"`
	ActorID      string         `json:"actor_id,omitempty"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	Payload      map[string]any `json:"payload"`
	CreatedAt    time.Time      `json:"created_at"`
}

type HouseEventsResponse struct {
	Events     []HouseEvent `json:"events"`
	NextCursor string       `json:"next_cursor"`
}

type AuditEvent struct {
	ID         string    `db:"id" json:"id"`
	ActorID    *string   `db:"actor_id" json:"actor_id,omitempty"`
	HouseID    *string   `db:"house_id" json:"house_id,omitempty"`
	Action     string    `db:"action" json:"action"`
	TargetType string    `db:"target_type" json:"target_type"`
	TargetID   string    `db:"target_id" json:"target_id"`
	Metadata   string    `db:"metadata" json:"metadata"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}
