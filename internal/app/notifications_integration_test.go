package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

// slowProvider takes perSend for each event, or until its context ends.
type slowProvider struct {
	perSend time.Duration
	sent    atomic.Int32
}

func (p *slowProvider) Send(ctx context.Context, _ notifications.OutboxEvent) error {
	select {
	case <-time.After(p.perSend):
		p.sent.Add(1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestNotificationDispatcherStopsBeforeItsClaimLapses(t *testing.T) {
	a := newTestAPI(t)
	user := a.register("dispatch-budget")
	userID, err := uuid.Parse(user.ID)
	require.NoError(t, err)
	svc := notifications.NewService(a.pool)
	for i := 0; i < 5; i++ {
		tx, err := a.pool.Begin(context.Background())
		require.NoError(t, err)
		require.NoError(t, svc.EnqueueTx(context.Background(), tx, userID,
			"test.event", "account", userID, map[string]string{}, fmt.Sprintf("budget-%d", i)))
		require.NoError(t, tx.Commit(context.Background()))
	}

	// Each send takes 100ms; a 250ms budget fits two, cuts off the third.
	provider := &slowProvider{perSend: 100 * time.Millisecond}
	dispatcher := notifications.NewDispatcher(svc, provider, notifications.WithSendBudget(250*time.Millisecond))
	start := time.Now()
	delivered, err := dispatcher.RunOnce(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 2, delivered)
	assert.Less(t, time.Since(start), 400*time.Millisecond, "nothing keeps sending past the budget")

	pending, err := svc.PendingOutboxCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, pending, "the cut-off and unstarted events stay queued")

	var untouched int
	require.NoError(t, a.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM notification_outbox WHERE processed_at IS NULL AND attempts = 0`).Scan(&untouched))
	assert.Equal(t, 2, untouched, "events never started don't spend a retry attempt")
}
