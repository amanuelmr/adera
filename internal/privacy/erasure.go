package privacy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/adera-platform/backend/internal/platform/database"
)

// FormerMemberName replaces an erased account's display name. Clients show
// their own wording for it (reviewer_erased in review listings).
const FormerMemberName = "Former member"

// RunDueErasures carries out up to limit requests whose grace period is
// over, each in its own transaction, and returns how many it completed. A
// request that fails is logged and retried on the next run.
func (s *Service) RunDueErasures(ctx context.Context, limit int) (int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id FROM data_erasure_requests
		WHERE status = 'scheduled' AND scheduled_for <= now()
		ORDER BY scheduled_for LIMIT $1`, limit)
	if err != nil {
		return 0, fmt.Errorf("finding due erasures: %w", err)
	}
	type due struct{ requestID, userID uuid.UUID }
	var items []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.requestID, &d.userID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scanning due erasure: %w", err)
		}
		items = append(items, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterating due erasures: %w", err)
	}

	done := 0
	for _, d := range items {
		err := database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
			// Re-check under lock: the person may have signed in since.
			var status string
			err := tx.QueryRow(ctx, `
				SELECT status FROM data_erasure_requests WHERE id = $1 FOR UPDATE`, d.requestID).Scan(&status)
			if err != nil {
				return fmt.Errorf("locking request: %w", err)
			}
			if status != StatusScheduled {
				return errSkip
			}
			if err := eraseAccount(ctx, tx, d.userID, s.files.PrivateBucket, s.files.PublicBucket); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				UPDATE data_erasure_requests
				SET status = 'completed', completed_at = now(), reason = ''
				WHERE id = $1`, d.requestID); err != nil {
				return fmt.Errorf("completing request: %w", err)
			}
			_, err = insertEvent(ctx, tx, d.requestID, uuid.Nil, "completed", "")
			return err
		})
		if errors.Is(err, errSkip) {
			continue
		}
		if err != nil {
			slog.ErrorContext(ctx, "erasing account", "request_id", d.requestID, "error", err)
			continue
		}
		done++
	}
	return done, nil
}

var errSkip = errors.New("skip")

// eraseAccount removes everything that identifies the user, keeping their
// reviews (rating, text, criterion scores) under a tombstone account. Stored
// files are queued for deletion in the same transaction.
//
// Kept, because they are not about the person or are needed as records:
// reviews and their scores, helpful votes (counts), reports they filed and
// moderation history, business responses written on a business's behalf,
// places and edit suggestions they contributed, and the erasure request
// itself (without its reason) as proof it was carried out.
func eraseAccount(ctx context.Context, tx pgx.Tx, userID uuid.UUID, privateBucket, publicBucket string) error {
	steps := []struct {
		what  string
		query string
		args  []any
	}{
		// Photos and receipts. A photo is in the public bucket, or the
		// private one under withheld/ while its review is out of view, or
		// still staged; evidence and claim documents are private.
		{"queueing review photos", `
			INSERT INTO pending_object_deletions (bucket, object_key)
			SELECT CASE WHEN m.status = 'ready' AND NOT m.withheld THEN $2 ELSE $3 END,
			       CASE WHEN m.status = 'ready' AND m.withheld THEN 'withheld/' ELSE '' END || k.key
			FROM review_media m JOIN reviews r ON r.id = m.review_id
			CROSS JOIN LATERAL (VALUES (m.object_key), (m.thumb_key)) AS k(key)
			WHERE r.user_id = $1 AND k.key IS NOT NULL
			ON CONFLICT DO NOTHING`, []any{userID, publicBucket, privateBucket}},
		{"queueing evidence", `
			INSERT INTO pending_object_deletions (bucket, object_key)
			SELECT $2, e.object_key FROM review_evidence e JOIN reviews r ON r.id = e.review_id
			WHERE r.user_id = $1
			ON CONFLICT DO NOTHING`, []any{userID, privateBucket}},
		{"queueing claim documents", `
			INSERT INTO pending_object_deletions (bucket, object_key)
			SELECT $2, evidence_object_key FROM business_claims
			WHERE user_id = $1 AND evidence_object_key <> ''
			ON CONFLICT DO NOTHING`, []any{userID, privateBucket}},
		{"deleting review photos", `
			DELETE FROM review_media WHERE review_id IN (SELECT id FROM reviews WHERE user_id = $1)`, []any{userID}},
		{"deleting evidence", `
			DELETE FROM review_evidence WHERE review_id IN (SELECT id FROM reviews WHERE user_id = $1)`, []any{userID}},
		{"deleting fraud signals", `
			DELETE FROM review_signals WHERE review_id IN (SELECT id FROM reviews WHERE user_id = $1)`, []any{userID}},
		// A profile link identifies the author outright.
		{"clearing profile links", `
			UPDATE reviews SET social_media_url = '' WHERE user_id = $1 AND social_media_url <> ''`, []any{userID}},
		{"deleting business claims", `DELETE FROM business_claims WHERE user_id = $1`, []any{userID}},
		{"deleting business memberships", `DELETE FROM business_members WHERE user_id = $1`, []any{userID}},
		{"deleting roles", `DELETE FROM user_roles WHERE user_id = $1`, []any{userID}},
		{"deleting sessions", `DELETE FROM auth_sessions WHERE user_id = $1`, []any{userID}},
		{"deleting codes", `DELETE FROM verification_tokens WHERE user_id = $1`, []any{userID}},
		{"deleting push tokens", `DELETE FROM device_tokens WHERE user_id = $1`, []any{userID}},
		{"deleting notifications", `DELETE FROM notifications WHERE user_id = $1`, []any{userID}},
		{"deleting stored responses", `DELETE FROM idempotency_keys WHERE user_id = $1`, []any{userID}},
		{"clearing request reasons", `
			UPDATE data_erasure_requests SET reason = '' WHERE user_id = $1`, []any{userID}},
		{"erasing the account", `
			UPDATE users SET display_name = $2, email = NULL, phone = NULL, password_hash = NULL,
				email_verified_at = NULL, phone_verified_at = NULL,
				status = 'erased', erased_at = now()
			WHERE id = $1`, []any{userID, FormerMemberName}},
	}
	for _, step := range steps {
		if _, err := tx.Exec(ctx, step.query, step.args...); err != nil {
			return fmt.Errorf("%s: %w", step.what, err)
		}
	}
	return nil
}

// PurgeQueuedObjects deletes up to limit queued files from storage and
// returns how many it deleted. A failure leaves the row for the next run.
func (s *Service) PurgeQueuedObjects(ctx context.Context, limit int) (int, error) {
	if s.files.Store == nil {
		return 0, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT bucket, object_key FROM pending_object_deletions
		ORDER BY attempts, created_at LIMIT $1`, limit)
	if err != nil {
		return 0, fmt.Errorf("finding queued deletions: %w", err)
	}
	type object struct{ bucket, key string }
	var objects []object
	for rows.Next() {
		var o object
		if err := rows.Scan(&o.bucket, &o.key); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scanning queued deletion: %w", err)
		}
		objects = append(objects, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterating queued deletions: %w", err)
	}

	deleted := 0
	for _, o := range objects {
		if err := s.files.Store.Remove(ctx, o.bucket, o.key); err != nil {
			slog.ErrorContext(ctx, "deleting stored file", "bucket", o.bucket, "key", o.key, "error", err)
			if _, err := s.pool.Exec(ctx, `
				UPDATE pending_object_deletions SET attempts = attempts + 1
				WHERE bucket = $1 AND object_key = $2`, o.bucket, o.key); err != nil {
				return deleted, fmt.Errorf("recording failed deletion: %w", err)
			}
			continue
		}
		if _, err := s.pool.Exec(ctx, `
			DELETE FROM pending_object_deletions WHERE bucket = $1 AND object_key = $2`, o.bucket, o.key); err != nil {
			return deleted, fmt.Errorf("clearing deleted file: %w", err)
		}
		deleted++
	}
	return deleted, nil
}
