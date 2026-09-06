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
	logger        *slog.Logger
	repo          *repository.ReliabilityRepository
	provider      providers.InvitationProvider
	owner         string
	pollInterval  time.Duration
	leaseDuration time.Duration
	ready         atomic.Bool
}

func New(logger *slog.Logger, repo *repository.ReliabilityRepository, provider providers.InvitationProvider, owner string, pollInterval, leaseDuration time.Duration) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	if leaseDuration <= 0 {
		leaseDuration = 30 * time.Second
	}
	return &Worker{
		logger:        logger,
		repo:          repo,
		provider:      provider,
		owner:         owner,
		pollInterval:  pollInterval,
		leaseDuration: leaseDuration,
	}
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
	var notification providers.InvitationNotification
	if err := json.Unmarshal([]byte(message.Payload), &notification); err != nil {
		return w.repo.FailOutboxJob(ctx, job, message.ID, err)
	}
	if w.provider == nil {
		return w.repo.FailOutboxJob(ctx, job, message.ID, errors.New("invitation provider is not configured"))
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
	w.logger.Info("worker_dispatch_succeeded",
		slog.String("job_id", job.ID),
		slog.String("outbox_message_id", message.ID),
		slog.String("topic", message.Topic),
	)
	return w.repo.CompleteOutboxJob(ctx, job, message.ID)
}
