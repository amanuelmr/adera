// Package privacy implements personal-data export and account erasure.
//
// Erasure is self-service: a request locks the account at once (signed out
// everywhere, no notifications) and is carried out ErasureGracePeriod later
// unless the person signs in again, which cancels it. Carrying it out
// removes everything that identifies them; their reviews stay, by a former
// member (docs/moderation-policy.md, "Retention & erasure").
package privacy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/notifications"
	"github.com/adera-platform/backend/internal/platform/database"
	"github.com/adera-platform/backend/internal/platform/storage"
	"github.com/adera-platform/backend/internal/platform/web"
)

// Request statuses. New requests are scheduled and end completed or
// cancelled; pending, in_review, approved and rejected belong to the earlier
// admin-reviewed workflow and appear only on old requests.
const (
	StatusPending   = "pending"
	StatusInReview  = "in_review"
	StatusApproved  = "approved"
	StatusRejected  = "rejected"
	StatusCancelled = "cancelled"
	StatusScheduled = "scheduled"
	StatusCompleted = "completed"
)

// ErasureGracePeriod is how long a request waits before it is carried out.
// Signing in during it cancels the request.
const ErasureGracePeriod = 7 * 24 * time.Hour

// ErasureRequest is a data subject's request to erase their account.
type ErasureRequest struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	Status       string     `json:"status"`
	Reason       string     `json:"reason,omitempty"`
	DecisionNote string     `json:"decision_note,omitempty"`
	ReviewedBy   *uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	DecidedAt    *time.Time `json:"decided_at,omitempty"`
	ScheduledFor *time.Time `json:"scheduled_for,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// DataExport is a versioned, machine-readable snapshot. RawMessage keeps the
// database's JSON types intact without exposing unselected internal columns.
type DataExport struct {
	SchemaVersion         string          `json:"schema_version"`
	GeneratedAt           time.Time       `json:"generated_at"`
	Profile               json.RawMessage `json:"profile"`
	Roles                 json.RawMessage `json:"roles"`
	Sessions              json.RawMessage `json:"sessions"`
	Businesses            json.RawMessage `json:"businesses_created"`
	BusinessMemberships   json.RawMessage `json:"business_memberships"`
	Targets               json.RawMessage `json:"targets_created"`
	TargetEditSuggestions json.RawMessage `json:"target_edit_suggestions"`
	Reviews               json.RawMessage `json:"reviews"`
	CriterionScores       json.RawMessage `json:"review_criterion_scores"`
	ReviewMedia           json.RawMessage `json:"review_media_metadata"`
	ReviewEvidence        json.RawMessage `json:"review_evidence_metadata"`
	HelpfulVotes          json.RawMessage `json:"helpful_votes"`
	Reports               json.RawMessage `json:"reports"`
	BusinessClaims        json.RawMessage `json:"business_claims"`
	BusinessResponses     json.RawMessage `json:"business_responses"`
	BusinessResponseEdits json.RawMessage `json:"business_response_edits"`
	Notifications         json.RawMessage `json:"notifications"`
	ErasureRequests       json.RawMessage `json:"erasure_requests"`
}

type Service struct {
	pool          *pgxpool.Pool
	notifications *notifications.Service
	store         storage.Store
	privateBucket string
	publicBucket  string
}

// NewService builds the privacy service. store may be nil when object
// storage is off; queued file deletions then wait until it is on.
func NewService(pool *pgxpool.Pool, notificationService *notifications.Service,
	store storage.Store, privateBucket, publicBucket string) *Service {
	return &Service{pool: pool, notifications: notificationService,
		store: store, privateBucket: privateBucket, publicBucket: publicBucket}
}

const requestColumns = `id, user_id, status, reason, decision_note, reviewed_by,
	reviewed_at, decided_at, scheduled_for, completed_at, created_at, updated_at`

func scanRequest(row pgx.Row) (ErasureRequest, error) {
	var item ErasureRequest
	if err := row.Scan(&item.ID, &item.UserID, &item.Status, &item.Reason, &item.DecisionNote,
		&item.ReviewedBy, &item.ReviewedAt, &item.DecidedAt, &item.ScheduledFor, &item.CompletedAt,
		&item.CreatedAt, &item.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErasureRequest{}, web.ErrNotFound("erasure request")
		}
		return ErasureRequest{}, fmt.Errorf("scanning erasure request: %w", err)
	}
	return item, nil
}

// insertEvent records a step in a request's history. A nil actorID records
// the system (the scheduled erasure job) as the actor.
func insertEvent(ctx context.Context, tx pgx.Tx, requestID, actorID uuid.UUID, action, note string) (uuid.UUID, error) {
	id := uuid.New()
	var actor *uuid.UUID
	if actorID != uuid.Nil {
		actor = &actorID
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO data_erasure_request_events (id, request_id, actor_id, action, note)
		VALUES ($1, $2, $3, $4, $5)`, id, requestID, actor, action, strings.TrimSpace(note)); err != nil {
		return uuid.Nil, fmt.Errorf("recording erasure request event: %w", err)
	}
	return id, nil
}

func validateText(field, value string, max int) (string, error) {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > max {
		return "", web.ErrValidation("invalid erasure request").WithDetail(field,
			fmt.Sprintf("must be at most %d characters", max))
	}
	return value, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// CreateRequest schedules the caller's account for erasure and locks it
// straight away: the account is deactivated, every session is revoked and
// push tokens are dropped. Signing in again within ErasureGracePeriod
// cancels the request (auth.Service.Login).
func (s *Service) CreateRequest(ctx context.Context, userID uuid.UUID, reason string) (ErasureRequest, error) {
	reason, err := validateText("reason", reason, 1000)
	if err != nil {
		return ErasureRequest{}, err
	}
	var item ErasureRequest
	err = database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		item, err = scanRequest(tx.QueryRow(ctx, `
			INSERT INTO data_erasure_requests (id, user_id, reason, status, scheduled_for)
			VALUES ($1, $2, $3, 'scheduled', now() + make_interval(secs => $4))
			RETURNING `+requestColumns, uuid.New(), userID, reason, ErasureGracePeriod.Seconds()))
		if err != nil {
			return err
		}
		if _, err := insertEvent(ctx, tx, item.ID, userID, "requested", reason); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE users SET status = 'deactivated' WHERE id = $1 AND status = 'active'`, userID); err != nil {
			return fmt.Errorf("locking account: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE auth_sessions SET revoked_at = now(), revoked_reason = 'erasure_requested'
			WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
			return fmt.Errorf("revoking sessions: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM device_tokens WHERE user_id = $1`, userID); err != nil {
			return fmt.Errorf("removing push tokens: %w", err)
		}
		return nil
	})
	if isUniqueViolation(err) {
		return ErasureRequest{}, web.ErrConflict("an active erasure request already exists")
	}
	return item, err
}

func (s *Service) ListMine(ctx context.Context, userID uuid.UUID, cursor *web.Cursor, limit int) ([]ErasureRequest, *web.Cursor, error) {
	args := []any{userID}
	cond := ""
	if cursor != nil {
		ts, ok := cursor.Time()
		if !ok {
			return nil, nil, web.ErrValidation("invalid cursor")
		}
		args = append(args, ts, cursor.ID)
		cond = fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", len(args)-1, len(args))
	}
	args = append(args, limit+1)
	rows, err := s.pool.Query(ctx, `SELECT `+requestColumns+` FROM data_erasure_requests
		WHERE user_id = $1`+cond+` ORDER BY created_at DESC, id DESC LIMIT $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, nil, fmt.Errorf("listing erasure requests: %w", err)
	}
	defer rows.Close()
	var items []ErasureRequest
	for rows.Next() {
		item, err := scanRequest(rows)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterating erasure requests: %w", err)
	}
	return requestPage(items, limit)
}

func requestPage(items []ErasureRequest, limit int) ([]ErasureRequest, *web.Cursor, error) {
	var next *web.Cursor
	if len(items) > limit {
		items = items[:limit]
		value := web.TimeCursor(items[len(items)-1].CreatedAt, items[len(items)-1].ID)
		next = &value
	}
	return items, next, nil
}

func (s *Service) CancelRequest(ctx context.Context, userID, requestID uuid.UUID) (ErasureRequest, error) {
	var item ErasureRequest
	err := database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		item, err = scanRequest(tx.QueryRow(ctx, `SELECT `+requestColumns+`
			FROM data_erasure_requests WHERE id = $1 AND user_id = $2 FOR UPDATE`, requestID, userID))
		if err != nil {
			return err
		}
		if item.Status != StatusPending && item.Status != StatusInReview && item.Status != StatusScheduled {
			return web.ErrConflict("this erasure request can no longer be cancelled")
		}
		item, err = scanRequest(tx.QueryRow(ctx, `UPDATE data_erasure_requests
			SET status = 'cancelled' WHERE id = $1 RETURNING `+requestColumns, requestID))
		if err != nil {
			return err
		}
		_, err = insertEvent(ctx, tx, item.ID, userID, "cancelled", "")
		return err
	})
	return item, err
}

// CancelOnSignIn cancels the user's scheduled erasure, if any, because they
// signed in during the grace period. It runs in the sign-in's transaction.
func CancelOnSignIn(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	var requestID uuid.UUID
	err := tx.QueryRow(ctx, `
		UPDATE data_erasure_requests SET status = 'cancelled'
		WHERE user_id = $1 AND status = 'scheduled' RETURNING id`, userID).Scan(&requestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cancelling scheduled erasure: %w", err)
	}
	_, err = insertEvent(ctx, tx, requestID, userID, "cancelled", "signed in during the grace period")
	return err
}

func (s *Service) ListForAdmin(ctx context.Context, status string, cursor *web.Cursor, limit int) ([]ErasureRequest, *web.Cursor, error) {
	args := []any{}
	conds := []string{"true"}
	if status != "" {
		switch status {
		case StatusPending, StatusInReview, StatusApproved, StatusRejected, StatusCancelled,
			StatusScheduled, StatusCompleted:
		default:
			return nil, nil, web.ErrValidation("invalid filter").WithDetail("status", "unsupported status")
		}
		args = append(args, status)
		conds = append(conds, fmt.Sprintf("status = $%d", len(args)))
	}
	if cursor != nil {
		ts, ok := cursor.Time()
		if !ok {
			return nil, nil, web.ErrValidation("invalid cursor")
		}
		args = append(args, ts, cursor.ID)
		conds = append(conds, fmt.Sprintf("(created_at, id) > ($%d, $%d)", len(args)-1, len(args)))
	}
	args = append(args, limit+1)
	rows, err := s.pool.Query(ctx, `SELECT `+requestColumns+` FROM data_erasure_requests WHERE `+
		strings.Join(conds, " AND ")+` ORDER BY created_at, id LIMIT $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, nil, fmt.Errorf("listing erasure request queue: %w", err)
	}
	defer rows.Close()
	var items []ErasureRequest
	for rows.Next() {
		item, err := scanRequest(rows)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterating erasure request queue: %w", err)
	}
	if len(items) > limit {
		items = items[:limit]
		value := web.TimeCursor(items[len(items)-1].CreatedAt, items[len(items)-1].ID)
		return items, &value, nil
	}
	return items, nil, nil
}

func jsonValue(ctx context.Context, tx pgx.Tx, query string, userID uuid.UUID) (json.RawMessage, error) {
	var raw []byte
	if err := tx.QueryRow(ctx, query, userID).Scan(&raw); err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

// Export returns a consistent snapshot of user-provided and account data.
// Credential material and private object-storage keys are never selected.
func (s *Service) Export(ctx context.Context, userID uuid.UUID) (DataExport, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return DataExport{}, fmt.Errorf("beginning data export: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	out := DataExport{SchemaVersion: "1", GeneratedAt: time.Now().UTC()}
	sections := []struct {
		name  string
		dest  *json.RawMessage
		query string
	}{
		{"profile", &out.Profile, `SELECT jsonb_build_object(
			'id', id, 'display_name', display_name, 'email', email, 'phone', phone,
			'status', status, 'email_verified_at', email_verified_at,
			'phone_verified_at', phone_verified_at, 'preferred_language', preferred_language,
			'created_at', created_at, 'updated_at', updated_at) FROM users WHERE id = $1`},
		{"roles", &out.Roles, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.role), '[]'::jsonb)
			FROM (SELECT role, granted_at FROM user_roles WHERE user_id = $1) x`},
		{"sessions", &out.Sessions, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.created_at), '[]'::jsonb)
			FROM (SELECT id, family_id, device_info, created_at, expires_at, last_used_at,
				revoked_at, revoked_reason FROM auth_sessions WHERE user_id = $1) x`},
		{"businesses", &out.Businesses, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.created_at), '[]'::jsonb)
			FROM (SELECT id, name, slug, description, verification_status, status, created_at, updated_at
				FROM businesses WHERE created_by = $1) x`},
		{"business memberships", &out.BusinessMemberships, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.granted_at), '[]'::jsonb)
			FROM (SELECT business_id, role, granted_at FROM business_members WHERE user_id = $1) x`},
		{"targets", &out.Targets, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.created_at), '[]'::jsonb)
			FROM (SELECT id, target_type, category_id, business_id, name, slug, description, city_id,
				area_id, address_text, latitude, longitude, online_only, phone, website, social_links,
				metadata, verification_status, moderation_status, merged_into_id, created_at, updated_at
				FROM review_targets WHERE created_by = $1) x`},
		{"target edit suggestions", &out.TargetEditSuggestions, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.created_at), '[]'::jsonb)
			FROM (SELECT id, target_id, changes, note, status, reviewed_at, created_at
				FROM target_edit_suggestions WHERE user_id = $1) x`},
		{"reviews", &out.Reviews, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.created_at), '[]'::jsonb)
			FROM (SELECT * FROM reviews WHERE user_id = $1) x`},
		{"criterion scores", &out.CriterionScores, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.review_id, x.code), '[]'::jsonb)
			FROM (SELECT s.review_id, c.code, s.score FROM review_criterion_scores s
				JOIN reviews r ON r.id = s.review_id JOIN category_criteria c ON c.id = s.criterion_id
				WHERE r.user_id = $1) x`},
		{"review media", &out.ReviewMedia, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.created_at), '[]'::jsonb)
			FROM (SELECT m.id, m.review_id, m.content_type, m.size_bytes, m.status, m.created_at
				FROM review_media m JOIN reviews r ON r.id = m.review_id WHERE r.user_id = $1) x`},
		{"review evidence", &out.ReviewEvidence, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.created_at), '[]'::jsonb)
			FROM (SELECT e.id, e.review_id, e.kind, e.content_type, e.size_bytes, e.status,
				e.reviewed_at, e.note, e.created_at FROM review_evidence e
				JOIN reviews r ON r.id = e.review_id WHERE r.user_id = $1) x`},
		{"helpful votes", &out.HelpfulVotes, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.created_at), '[]'::jsonb)
			FROM (SELECT review_id, created_at FROM helpful_votes WHERE user_id = $1) x`},
		{"reports", &out.Reports, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.created_at), '[]'::jsonb)
			FROM (SELECT id, review_id, target_id, reason, details, status, resolved_at,
				resolution_note, created_at FROM reports WHERE reporter_id = $1) x`},
		{"business claims", &out.BusinessClaims, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.created_at), '[]'::jsonb)
			FROM (SELECT id, business_id, method, (evidence_object_key <> '') AS evidence_attached,
				message, status, decided_at, decision_note, created_at FROM business_claims WHERE user_id = $1) x`},
		{"business responses", &out.BusinessResponses, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.created_at), '[]'::jsonb)
			FROM (SELECT id, review_id, business_id, body, created_at, updated_at
				FROM business_responses WHERE author_user_id = $1) x`},
		{"business response edits", &out.BusinessResponseEdits, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.edited_at), '[]'::jsonb)
			FROM (SELECT id, response_id, previous_body, edited_at
				FROM business_response_edits WHERE edited_by = $1) x`},
		{"notifications", &out.Notifications, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.created_at), '[]'::jsonb)
			FROM (SELECT id, event_type, subject_type, subject_id, data, read_at, created_at
				FROM notifications WHERE user_id = $1) x`},
		{"erasure requests", &out.ErasureRequests, `SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.created_at), '[]'::jsonb)
			FROM (SELECT id, status, reason, decision_note, reviewed_at, decided_at, created_at, updated_at
				FROM data_erasure_requests WHERE user_id = $1) x`},
	}
	for _, section := range sections {
		value, err := jsonValue(ctx, tx, section.query, userID)
		if err != nil {
			return DataExport{}, fmt.Errorf("exporting %s: %w", section.name, err)
		}
		*section.dest = value
	}
	if err := tx.Commit(ctx); err != nil {
		return DataExport{}, fmt.Errorf("committing data export snapshot: %w", err)
	}
	return out, nil
}
