package notifications

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsoleProviderSend(t *testing.T) {
	var buf bytes.Buffer
	p := ConsoleProvider{W: &buf}
	event := OutboxEvent{ID: uuid.New(), Topic: "notification.created", Payload: []byte(`{"a":1}`)}

	err := p.Send(context.Background(), event)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "notification.created")
	assert.Contains(t, buf.String(), `{"a":1}`)
}

func TestNoopProviderSend(t *testing.T) {
	err := NoopProvider{}.Send(context.Background(), OutboxEvent{ID: uuid.New()})

	assert.ErrorIs(t, err, ErrProviderUnavailable)
}

func TestBackoffGrowsAndCaps(t *testing.T) {
	cases := []struct {
		attempts int
		want     time.Duration
	}{
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{20, 5 * time.Minute}, // capped
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, backoff(tc.attempts))
	}
}

func TestTruncateError(t *testing.T) {
	short := errors.New("boom")
	assert.Equal(t, "boom", truncateError(short))

	long := errors.New(string(make([]byte, 600)))
	assert.Len(t, truncateError(long), 500)
}
