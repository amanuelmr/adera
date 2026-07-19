package reviews

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/adera-platform/backend/internal/platform/web"
)

// Vote marks a published review as helpful. Idempotent: voting twice is a
// no-op. Users cannot vote for their own reviews.
func (r *Repo) Vote(ctx context.Context, reviewID, userID uuid.UUID) (int, error) {
	var authorID uuid.UUID
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, moderation_status FROM reviews WHERE id = $1`, reviewID).
		Scan(&authorID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, web.ErrNotFound("review")
	}
	if err != nil {
		return 0, fmt.Errorf("loading review: %w", err)
	}
	if status != StatusPublished {
		return 0, web.ErrNotFound("review")
	}
	if authorID == userID {
		return 0, web.ErrValidation("invalid vote").WithDetail("review_id", "you cannot vote for your own review")
	}
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO helpful_votes (review_id, user_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, reviewID, userID); err != nil {
		return 0, fmt.Errorf("inserting vote: %w", err)
	}
	return r.voteCount(ctx, reviewID)
}

// Unvote removes the caller's helpful vote. Idempotent.
func (r *Repo) Unvote(ctx context.Context, reviewID, userID uuid.UUID) (int, error) {
	if _, err := r.pool.Exec(ctx, `
		DELETE FROM helpful_votes WHERE review_id = $1 AND user_id = $2`, reviewID, userID); err != nil {
		return 0, fmt.Errorf("deleting vote: %w", err)
	}
	return r.voteCount(ctx, reviewID)
}

func (r *Repo) voteCount(ctx context.Context, reviewID uuid.UUID) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM helpful_votes WHERE review_id = $1`, reviewID).Scan(&n); err != nil {
		return 0, fmt.Errorf("counting votes: %w", err)
	}
	return n, nil
}
