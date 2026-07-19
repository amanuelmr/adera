package reviews

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/adera-platform/backend/internal/platform/web"
)

// Idempotency protects review creation against duplicate submissions from
// weak connections and repeated mobile taps. The client sends an
// Idempotency-Key header; a replay with the same key returns the stored
// response, and the same key with a different body is rejected.
//
// Flow (race-safe via the primary key):
//  1. INSERT the key with the request hash; a conflicting concurrent request
//     either finds a stored response (replay) or an in-flight marker.
//  2. Execute the operation.
//  3. Store the response body/status on the key row.
//
// Keys older than 24h are purged opportunistically.

// HashRequest fingerprints the request payload.
func HashRequest(v any) []byte {
	b, _ := json.Marshal(v) // DTOs marshal deterministically enough for equality
	sum := sha256.Sum256(b)
	return sum[:]
}

// BeginIdempotent registers the key. It returns (stored, true) when a
// completed response already exists for the same payload.
func (r *Repo) BeginIdempotent(ctx context.Context, userID uuid.UUID, endpoint, key string, reqHash []byte) (json.RawMessage, int, bool, error) {
	// Opportunistic purge keeps the table small without a background job.
	if _, err := r.pool.Exec(ctx, `
		DELETE FROM idempotency_keys WHERE created_at < now() - interval '24 hours'`); err != nil {
		return nil, 0, false, fmt.Errorf("purging idempotency keys: %w", err)
	}
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO idempotency_keys (user_id, endpoint, idem_key, request_hash)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING`, userID, endpoint, key, reqHash)
	if err != nil {
		return nil, 0, false, fmt.Errorf("registering idempotency key: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return nil, 0, false, nil // fresh key: proceed
	}
	var (
		storedHash []byte
		body       *json.RawMessage
		status     *int
	)
	err = r.pool.QueryRow(ctx, `
		SELECT request_hash, response_body, response_status
		FROM idempotency_keys WHERE user_id = $1 AND endpoint = $2 AND idem_key = $3`,
		userID, endpoint, key).Scan(&storedHash, &body, &status)
	if err != nil {
		return nil, 0, false, fmt.Errorf("loading idempotency key: %w", err)
	}
	if string(storedHash) != string(reqHash) {
		return nil, 0, false, web.ErrConflict("this idempotency key was used with a different request body")
	}
	if body == nil || status == nil {
		// Original request still in flight.
		return nil, 0, false, web.ErrConflict("a request with this idempotency key is already in progress")
	}
	return *body, *status, true, nil
}

// CompleteIdempotent stores the response for future replays.
func (r *Repo) CompleteIdempotent(ctx context.Context, userID uuid.UUID, endpoint, key string, status int, body any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling idempotent response: %w", err)
	}
	if _, err := r.pool.Exec(ctx, `
		UPDATE idempotency_keys SET response_status = $4, response_body = $5
		WHERE user_id = $1 AND endpoint = $2 AND idem_key = $3`,
		userID, endpoint, key, status, raw); err != nil {
		return fmt.Errorf("storing idempotent response: %w", err)
	}
	return nil
}

// AbortIdempotent releases the key after a failed attempt so the client can
// retry with the same key.
func (r *Repo) AbortIdempotent(ctx context.Context, userID uuid.UUID, endpoint, key string) error {
	if _, err := r.pool.Exec(ctx, `
		DELETE FROM idempotency_keys
		WHERE user_id = $1 AND endpoint = $2 AND idem_key = $3 AND response_status IS NULL`,
		userID, endpoint, key); err != nil {
		return fmt.Errorf("releasing idempotency key: %w", err)
	}
	return nil
}
