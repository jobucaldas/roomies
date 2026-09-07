package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/roomies/backend/internal/providers"
	"github.com/roomies/backend/internal/repository"
)

type Worker struct {
	logger               *slog.Logger
	repo                 *repository.ReliabilityRepository
	provider             providers.InvitationProvider
	owner                string
	pollInterval         time.Duration
	leaseDuration        time.Duration
	deliveryCipher       *providers.InvitationDeliveryCipher
	notificationRepo     *repository.NotificationRepository
	notificationProvider providers.NotificationProvider
	ready                atomic.Bool
}

func New(logger *slog.Logger, repo *repository.ReliabilityRepository, provider providers.InvitationProvider, owner string, pollInterval, leaseDuration time.Duration, deliveryCipher ...*providers.InvitationDeliveryCipher) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	if leaseDuration <= 0 {
		leaseDuration = 30 * time.Second
	}
	var cipher *providers.InvitationDeliveryCipher
	if len(deliveryCipher) > 0 {
		cipher = deliveryCipher[0]
	}
	return &Worker{
		logger:         logger,
		repo:           repo,
		provider:       provider,
		owner:          owner,
		pollInterval:   pollInterval,
		leaseDuration:  leaseDuration,
		deliveryCipher: cipher,
	}
}

func (w *Worker) ConfigureNotifications(repo *repository.NotificationRepository, provider providers.NotificationProvider) {
	w.notificationRepo = repo
	w.notificationProvider = provider
}

func (w *Worker) Ready() bool {
	return w.ready.Load()
}

func (w *Worker) Run(ctx context.Context) error {
	w.ready.Store(true)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		if err := w.RunOnce(ctx); err != nil {
			w.logger.Error("worker_iteration_failed", slog.String("error", err.Error()))
		}
		select {
		case <-ctx.Done():
			w.ready.Store(false)
			return nil
		case <-ticker.C:
		}
	}
}

func (w *Worker) RunOnce(ctx context.Context) error {
	if w.notificationRepo != nil {
		if _, err := w.notificationRepo.RunDueScheduledEvent(ctx, w.owner, w.leaseDuration); err != nil {
			return err
		}
	}
	job, err := w.repo.AcquireNextJob(ctx, w.owner, w.leaseDuration)
	if err != nil || job == nil {
		return err
	}
	outboxMessageID, err := w.repo.DecodeJobPayload(job)
	if err != nil {
		return w.repo.FailOutboxJob(ctx, job, outboxMessageID, err)
	}
	message, err := w.repo.GetOutboxMessage(ctx, outboxMessageID)
	if err != nil {
		return w.repo.FailOutboxJob(ctx, job, outboxMessageID, err)
	}
	if message.Status == "dispatched" || message.Status == "dead_lettered" {
		return w.repo.CompleteOutboxJob(ctx, job, "")
	}
	if message.Topic == "notification.delivery" {
		return w.dispatchNotification(ctx, job, message)
	}
	var notification providers.InvitationNotification
	if err := json.Unmarshal([]byte(message.Payload), &notification); err != nil {
		return w.repo.FailOutboxJob(ctx, job, message.ID, err)
	}
	if w.provider == nil {
		return w.repo.FailOutboxJob(ctx, job, message.ID, errors.New("invitation provider is not configured"))
	}
	if notification.Delivery != nil {
		if err := w.repo.InvitationDeliveryAvailable(ctx, notification.InvitationID); err != nil {
			if errors.Is(err, repository.ErrInvitationUnavailable) {
				return w.repo.CompleteOutboxJob(ctx, job, message.ID)
			}
			return w.repo.FailOutboxJob(ctx, job, message.ID, err)
		}
		if w.deliveryCipher == nil {
			return w.repo.FailOutboxJob(ctx, job, message.ID, errors.New("invitation delivery cipher is not configured"))
		}
		acceptanceURL, err := w.deliveryCipher.Decrypt(notification.Delivery, notification.InvitationID, notification.HouseID, job.ID)
		if err != nil {
			return w.repo.FailOutboxJob(ctx, job, message.ID, err)
		}
		notification.AcceptanceURL = acceptanceURL
		defer func() { notification.AcceptanceURL = "" }()
	} else if notification.Topic == "house.invitation.created" && w.deliveryCipher != nil {
		if err := w.repo.InvitationDeliveryAvailable(ctx, notification.InvitationID); err != nil {
			if errors.Is(err, repository.ErrInvitationUnavailable) {
				return w.repo.CompleteOutboxJob(ctx, job, message.ID)
			}
			return w.repo.FailOutboxJob(ctx, job, message.ID, err)
		}
		return w.repo.FailOutboxJob(ctx, job, message.ID, errors.New("invitation delivery payload is unavailable"))
	}
	if err := w.provider.DispatchInvitation(ctx, notification); err != nil {
		w.logger.Warn("worker_dispatch_failed",
			slog.String("job_id", job.ID),
			slog.String("outbox_message_id", message.ID),
			slog.Int("attempt", job.Attempts),
			slog.String("error", err.Error()),
		)
		return w.repo.FailOutboxJob(ctx, job, message.ID, err)
	}
	if _, fake := w.provider.(*providers.FakeInvitationProvider); fake {
		w.logger.Info("worker_dispatch_recorded_fake_only",
			slog.String("job_id", job.ID),
			slog.String("outbox_message_id", message.ID),
			slog.String("topic", message.Topic),
		)
	} else {
		w.logger.Info("worker_dispatch_succeeded",
			slog.String("job_id", job.ID),
			slog.String("outbox_message_id", message.ID),
			slog.String("topic", message.Topic),
		)
	}
	return w.repo.CompleteOutboxJob(ctx, job, message.ID)
}

func (w *Worker) dispatchNotification(ctx context.Context, job *repository.DurableJob, message *repository.OutboxMessage) error {
	if w.notificationRepo == nil || w.notificationProvider == nil {
		return w.repo.FailOutboxJob(ctx, job, message.ID, errors.New("notification provider is not configured"))
	}
	var payload repository.NotificationPayload
	if err := json.Unmarshal([]byte(message.Payload), &payload); err != nil {
		return w.repo.FailOutboxJob(ctx, job, message.ID, err)
	}
	// One deadline bounds both authorization-lock waits and all external sends
	// performed while those locks are held.
	dispatchCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	summary, err := w.notificationRepo.WithAuthorizedDispatch(dispatchCtx, payload, func(target repository.NotificationTarget, body []byte) (bool, error) {
		result, dispatchErr := w.notificationProvider.Dispatch(dispatchCtx, providers.PushTarget{Platform: target.Platform, Endpoint: target.Endpoint, P256DH: target.P256DH, Auth: target.Auth, Token: target.Token}, body)
		if result.Fake {
			w.logger.Warn("notification_channel_fake_only", slog.String("platform", target.Platform), slog.String("notification_id", payload.NotificationID))
		}
		return result.Revoke, dispatchErr
	})
	for _, targetID := range summary.InvalidTargetIDs {
		w.logger.Error("notification_subscription_quarantined", slog.String("subscription_id", targetID))
	}
	if err != nil {
		var transient *providers.TransientDeliveryError
		if errors.As(err, &transient) && !transient.RetryAt.IsZero() {
			return w.repo.FailOutboxJobAt(ctx, job, message.ID, err, transient.RetryAt)
		}
		return w.repo.FailOutboxJob(ctx, job, message.ID, err)
	}
	if summary.Dispatched && w.notificationProvider.Fake() {
		w.logger.Warn("notification_dispatch_recorded_fake_only", slog.String("notification_id", payload.NotificationID))
	}
	return w.repo.CompleteOutboxJob(ctx, job, message.ID)
}
