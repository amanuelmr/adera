package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
)

// OutboxEvent is a durable outbox row claimed for external delivery.
// Delivery is at-least-once: a Provider may see the same event more than
// once if the process crashes between a successful Send and it being marked
// delivered.
type OutboxEvent struct {
	ID       uuid.UUID
	Topic    string
	Payload  json.RawMessage
	Attempts int
}

// Provider delivers outbox events to an external channel (email, SMS, push).
type Provider interface {
	Send(ctx context.Context, event OutboxEvent) error
}

// ConsoleProvider writes outbox events to a writer (stdout) for local
// development. It must never be constructed in production.
type ConsoleProvider struct {
	W io.Writer
}

func (p ConsoleProvider) Send(_ context.Context, event OutboxEvent) error {
	_, err := fmt.Fprintf(p.W, "[DEV NOTIFICATION] topic=%s payload=%s\n", event.Topic, event.Payload)
	if err != nil {
		return fmt.Errorf("writing dev notification: %w", err)
	}
	return nil
}

// NoopProvider is used when no real delivery integration is configured.
// Sends fail loudly so events accumulate in the outbox for later replay
// instead of being silently dropped.
type NoopProvider struct{}

// ErrProviderUnavailable signals that no delivery integration is configured.
var ErrProviderUnavailable = errors.New("notification delivery provider not configured")

func (NoopProvider) Send(context.Context, OutboxEvent) error {
	return ErrProviderUnavailable
}
