package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const (
	dispatcherPollInterval = 5 * time.Second
	dispatcherBatchSize    = 20
	dispatcherMaxAttempts  = 8
)

// Dispatcher drains the notification outbox to an external Provider
// (email/SMS/push), retrying failed sends with exponential backoff.
type Dispatcher struct {
	service  *Service
	provider Provider
}

func NewDispatcher(service *Service, provider Provider) *Dispatcher {
	return &Dispatcher{service: service, provider: provider}
}

// Run polls the outbox until ctx is cancelled. It never returns an error;
// per-poll failures are logged and retried on the next tick.
func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(dispatcherPollInterval)
	defer ticker.Stop()
	for {
		if _, err := d.RunOnce(ctx); err != nil {
			slog.Error("notification dispatcher poll failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RunOnce claims one batch of due outbox events and attempts delivery,
// returning how many were delivered successfully.
func (d *Dispatcher) RunOnce(ctx context.Context) (int, error) {
	events, err := d.service.ClaimOutboxBatch(ctx, dispatcherBatchSize)
	if err != nil {
		return 0, fmt.Errorf("claiming notification outbox batch: %w", err)
	}
	delivered := 0
	for _, event := range events {
		if err := d.provider.Send(ctx, event); err != nil {
			d.retry(ctx, event, err)
			continue
		}
		if err := d.service.MarkOutboxDelivered(ctx, event.ID); err != nil {
			slog.Error("marking notification outbox event delivered", "event_id", event.ID, "error", err)
			continue
		}
		delivered++
	}
	return delivered, nil
}

func (d *Dispatcher) retry(ctx context.Context, event OutboxEvent, sendErr error) {
	attempts := event.Attempts + 1
	if attempts >= dispatcherMaxAttempts {
		slog.Error("abandoning notification outbox event after max attempts",
			"event_id", event.ID, "attempts", attempts, "error", sendErr)
		if err := d.service.MarkOutboxAbandoned(ctx, event.ID, sendErr); err != nil {
			slog.Error("marking notification outbox event abandoned", "event_id", event.ID, "error", err)
		}
		return
	}
	slog.Warn("notification delivery failed; will retry",
		"event_id", event.ID, "attempts", attempts, "error", sendErr)
	if err := d.service.MarkOutboxRetry(ctx, event.ID, sendErr, backoff(attempts)); err != nil {
		slog.Error("recording notification outbox retry", "event_id", event.ID, "error", err)
	}
}

// backoff grows 2s, 4s, 8s, ... capped at 5 minutes.
func backoff(attempts int) time.Duration {
	const maxBackoff = 5 * time.Minute
	shift := attempts
	if shift > 12 {
		shift = 12
	}
	d := time.Duration(1<<shift) * time.Second
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}
