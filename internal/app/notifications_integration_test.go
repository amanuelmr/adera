package app

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/notifications"
)

// recordingProvider records delivered events, or fails every send when told to.
type recordingProvider struct {
	mu     sync.Mutex
	events []notifications.OutboxEvent
	fail   bool
}

func (p *recordingProvider) Send(_ context.Context, event notifications.OutboxEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.fail {
		return errors.New("boom")
	}
	p.events = append(p.events, event)
	return nil
}

func TestNotificationDispatcherDeliversAndRetries(t *testing.T) {
	a := newTestAPI(t)
	user := a.register("dispatch-user")
	userID, err := uuid.Parse(user.ID)
	require.NoError(t, err)

	svc := notifications.NewService(a.pool)
	tx, err := a.pool.Begin(context.Background())
	require.NoError(t, err)
	require.NoError(t, svc.EnqueueTx(context.Background(), tx, userID,
		"test.event", "account", userID, map[string]string{"k": "v"}, "dispatch-test-1"))
	require.NoError(t, tx.Commit(context.Background()))

	pending, err := svc.PendingOutboxCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, pending)

	failing := &recordingProvider{fail: true}
	dispatcher := notifications.NewDispatcher(svc, failing)
	delivered, err := dispatcher.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, delivered, "failed sends must not count as delivered")

	var attempts int
	var lastError string
	require.NoError(t, a.pool.QueryRow(context.Background(), `
		SELECT o.attempts, o.last_error FROM notification_outbox o
		JOIN notifications n ON n.id = o.notification_id
		WHERE n.dedupe_key = $1`, "dispatch-test-1").Scan(&attempts, &lastError))
	assert.Equal(t, 1, attempts)
	assert.Contains(t, lastError, "boom")

	working := &recordingProvider{}
	dispatcher = notifications.NewDispatcher(svc, working)
	delivered, err = dispatcher.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, delivered, "event is hidden behind retry backoff and is not yet due")

	_, err = a.pool.Exec(context.Background(), `
		UPDATE notification_outbox SET available_at = now()
		WHERE notification_id IN (SELECT id FROM notifications WHERE dedupe_key = $1)`, "dispatch-test-1")
	require.NoError(t, err)

	delivered, err = dispatcher.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, delivered)
	require.Len(t, working.events, 1)
	assert.Equal(t, "notification.created", working.events[0].Topic)

	pending, err = svc.PendingOutboxCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, pending)
}
