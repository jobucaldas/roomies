package providers

import (
	"context"
	"errors"
	"sync"
	"time"
)

type InvitationNotification struct {
	Topic        string            `json:"topic"`
	InvitationID string            `json:"invitation_id"`
	HouseID      string            `json:"house_id"`
	Email        string            `json:"email"`
	Role         string            `json:"role"`
	Status       string            `json:"status"`
	OccurredAt   time.Time         `json:"occurred_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// InvitationProvider dispatches notifications with at-least-once semantics.
// A lease can expire after a provider accepts a request but before the worker
// records completion, so providers should deduplicate by Topic and InvitationID
// when they support an idempotency key. The worker never promises exactly-once
// external delivery.
type InvitationProvider interface {
	DispatchInvitation(context.Context, InvitationNotification) error
}

type FakeInvitationProvider struct {
	mu            sync.Mutex
	FailuresLeft  int
	Failure       error
	Notifications []InvitationNotification
}

func (p *FakeInvitationProvider) DispatchInvitation(_ context.Context, notification InvitationNotification) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.FailuresLeft > 0 {
		p.FailuresLeft--
		if p.Failure != nil {
			return p.Failure
		}
		return errors.New("fake invitation provider failure")
	}
	p.Notifications = append(p.Notifications, notification)
	return nil
}

func (p *FakeInvitationProvider) Snapshot() []InvitationNotification {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]InvitationNotification, len(p.Notifications))
	copy(result, p.Notifications)
	return result
}
