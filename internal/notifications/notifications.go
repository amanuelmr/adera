// Package notifications implements the user activity inbox and its durable
// transactional outbox.
package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/platform/web"
)

// Notification contains localization-friendly event data. Clients translate
// event_type and interpolate data instead of receiving server-rendered text.
type Notification struct {
	ID          uuid.UUID         `json:"id"`
	EventType   string            `json:"event_type"`
	SubjectType string            `json:"subject_type"`
	SubjectID   uuid.UUID         `json:"subject_id"`
	Data        map[string]string `json:"data"`
	ReadAt      *time.Time        `json:"read_at,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// Service persists inbox and outbox records.
type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// EnqueueTx creates the inbox item and outbox event in the caller's domain
// transaction. The caller's dedupe key prevents repeat event insertion.
func (s *Service) EnqueueTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID,
	eventType, subjectType string, subjectID uuid.UUID, data map[string]string, dedupeKey string) error {
	if s == nil {
		return nil
	}
	if data == nil {
		data = map[string]string{}
	}
	notificationID := uuid.New()
	tag, err := tx.Exec(ctx, `
		INSERT INTO notifications
			(id, user_id, event_type, subject_type, subject_id, data, dedupe_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, dedupe_key) DO NOTHING`,
		notificationID, userID, eventType, subjectType, subjectID, data, dedupeKey)
	if err != nil {
		return fmt.Errorf("creating notification: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	payload := map[string]any{
		"notification_id": notificationID,
		"user_id":         userID,
		"event_type":      eventType,
		"subject_type":    subjectType,
		"subject_id":      subjectID,
		"data":            data,
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO notification_outbox (id, notification_id, payload)
		VALUES ($1, $2, $3)`, uuid.New(), notificationID, payload); err != nil {
		return fmt.Errorf("creating notification outbox event: %w", err)
	}
	return nil
}

func scanNotification(row pgx.Row) (Notification, error) {
	var item Notification
	var raw []byte
	if err := row.Scan(&item.ID, &item.EventType, &item.SubjectType, &item.SubjectID,
		&raw, &item.ReadAt, &item.CreatedAt); err != nil {
		return Notification{}, err
	}
	if err := json.Unmarshal(raw, &item.Data); err != nil {
		return Notification{}, fmt.Errorf("decoding notification data: %w", err)
	}
	item.CreatedAt = item.CreatedAt.UTC()
	if item.ReadAt != nil {
		readAt := item.ReadAt.UTC()
		item.ReadAt = &readAt
	}
	return item, nil
}

// List returns the caller's inbox with keyset pagination.
func (s *Service) List(ctx context.Context, userID uuid.UUID, unreadOnly bool,
	cursor *web.Cursor, limit int) ([]Notification, *web.Cursor, error) {
	args := []any{userID}
	conds := []string{"user_id = $1"}
	if unreadOnly {
		conds = append(conds, "read_at IS NULL")
	}
	if cursor != nil {
		ts, ok := cursor.Time()
		if !ok {
			return nil, nil, web.ErrValidation("invalid cursor")
		}
		args = append(args, ts, cursor.ID)
		conds = append(conds, fmt.Sprintf("(created_at, id) < ($%d, $%d)", len(args)-1, len(args)))
	}
	args = append(args, limit+1)
	rows, err := s.pool.Query(ctx, `
		SELECT id, event_type, subject_type, subject_id, data, read_at, created_at
		FROM notifications WHERE `+strings.Join(conds, " AND ")+`
		ORDER BY created_at DESC, id DESC LIMIT $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, nil, fmt.Errorf("listing notifications: %w", err)
	}
	defer rows.Close()
	var items []Notification
	for rows.Next() {
		item, err := scanNotification(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scanning notification: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterating notifications: %w", err)
	}
	var next *web.Cursor
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		value := web.TimeCursor(last.CreatedAt, last.ID)
		next = &value
	}
	return items, next, nil
}

// MarkRead idempotently marks one notification owned by the caller.
func (s *Service) MarkRead(ctx context.Context, userID, notificationID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE notifications SET read_at = coalesce(read_at, now())
		WHERE id = $1 AND user_id = $2`, notificationID, userID)
	if err != nil {
		return fmt.Errorf("marking notification read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return web.ErrNotFound("notification")
	}
	return nil
}

// MarkAllRead marks every unread notification for the caller.
func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE notifications SET read_at = now()
		WHERE user_id = $1 AND read_at IS NULL`, userID)
	if err != nil {
		return 0, fmt.Errorf("marking notifications read: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (s *Service) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting unread notifications: %w", err)
	}
	return count, nil
}

// PendingOutboxCount is used by readiness/operations tests and the
// dispatcher's callers to observe undelivered external events.
func (s *Service) PendingOutboxCount(ctx context.Context) (int, error) {
	var count int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM notification_outbox WHERE processed_at IS NULL`).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting pending notification events: %w", err)
	}
	return count, nil
}

// outboxVisibilityTimeout is how long a claimed event is hidden from other
// claimants before it is eligible for redelivery, in case the dispatcher
// crashes after Send succeeds but before the event is marked delivered.
const outboxVisibilityTimeout = 2 * time.Minute

// ClaimOutboxBatch atomically claims up to limit due, unprocessed outbox
// events and hides them from other claimants for outboxVisibilityTimeout.
// The SELECT and UPDATE run as one statement, so FOR UPDATE SKIP LOCKED is
// safe even with multiple dispatcher instances.
func (s *Service) ClaimOutboxBatch(ctx context.Context, limit int) ([]OutboxEvent, error) {
	rows, err := s.pool.Query(ctx, `
		WITH claimed AS (
			SELECT id FROM notification_outbox
			WHERE processed_at IS NULL AND available_at <= now()
			ORDER BY created_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE notification_outbox o
		SET available_at = now() + $2 * interval '1 second'
		FROM claimed
		WHERE o.id = claimed.id
		RETURNING o.id, o.topic, o.payload, o.attempts`, limit, outboxVisibilityTimeout.Seconds())
	if err != nil {
		return nil, fmt.Errorf("claiming notification outbox batch: %w", err)
	}
	defer rows.Close()
	var events []OutboxEvent
	for rows.Next() {
		var event OutboxEvent
		var raw []byte
		if err := rows.Scan(&event.ID, &event.Topic, &raw, &event.Attempts); err != nil {
			return nil, fmt.Errorf("scanning notification outbox event: %w", err)
		}
		event.Payload = raw
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating notification outbox batch: %w", err)
	}
	return events, nil
}

// MarkOutboxDelivered marks a claimed event as successfully delivered.
func (s *Service) MarkOutboxDelivered(ctx context.Context, id uuid.UUID) error {
	if _, err := s.pool.Exec(ctx, `
		UPDATE notification_outbox SET processed_at = now() WHERE id = $1`, id); err != nil {
		return fmt.Errorf("marking notification outbox event delivered: %w", err)
	}
	return nil
}

// MarkOutboxRetry records a failed delivery attempt and schedules a retry
// after backoff.
func (s *Service) MarkOutboxRetry(ctx context.Context, id uuid.UUID, sendErr error, backoff time.Duration) error {
	if _, err := s.pool.Exec(ctx, `
		UPDATE notification_outbox
		SET attempts = attempts + 1, available_at = now() + $2 * interval '1 second', last_error = $3
		WHERE id = $1`, id, backoff.Seconds(), truncateError(sendErr)); err != nil {
		return fmt.Errorf("recording notification outbox retry: %w", err)
	}
	return nil
}

// MarkOutboxAbandoned stops further delivery attempts after the retry limit
// is exhausted, while preserving attempts and last_error for operators to
// inspect and replay manually.
func (s *Service) MarkOutboxAbandoned(ctx context.Context, id uuid.UUID, sendErr error) error {
	if _, err := s.pool.Exec(ctx, `
		UPDATE notification_outbox
		SET attempts = attempts + 1, processed_at = now(), last_error = $2
		WHERE id = $1`, id, truncateError(sendErr)); err != nil {
		return fmt.Errorf("abandoning notification outbox event: %w", err)
	}
	return nil
}

// truncateError caps stored error text; last_error is diagnostic, not a log.
func truncateError(err error) string {
	const maxLen = 500
	msg := err.Error()
	if len(msg) > maxLen {
		return msg[:maxLen]
	}
	return msg
}
