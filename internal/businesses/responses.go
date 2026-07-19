package businesses

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/adera-platform/backend/internal/platform/database"
	"github.com/adera-platform/backend/internal/platform/web"
)

// reviewBusiness resolves the business owning the target a review is about.
func (r *Repo) reviewBusiness(ctx context.Context, reviewID uuid.UUID) (uuid.UUID, error) {
	var businessID *uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT t.business_id
		FROM reviews rv JOIN review_targets t ON t.id = rv.target_id
		WHERE rv.id = $1 AND rv.moderation_status = 'published'`, reviewID).Scan(&businessID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, web.ErrNotFound("review")
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolving review business: %w", err)
	}
	if businessID == nil {
		return uuid.Nil, web.ErrForbidden("this target is not linked to a claimable business")
	}
	return *businessID, nil
}

func validateResponseBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	if l := len([]rune(body)); l < 1 || l > 2000 {
		return "", web.ErrValidation("invalid response").WithDetail("body", "must be 1-2000 characters")
	}
	return body, nil
}

// CreateResponse posts the single public business reply to a review. Only
// members of the business owning the review's target may respond; a business
// can never delete or alter the review itself.
func (r *Repo) CreateResponse(ctx context.Context, reviewID, userID uuid.UUID, body string) (Response, error) {
	body, err := validateResponseBody(body)
	if err != nil {
		return Response{}, err
	}
	businessID, err := r.reviewBusiness(ctx, reviewID)
	if err != nil {
		return Response{}, err
	}
	member, err := r.IsMember(ctx, businessID, userID)
	if err != nil {
		return Response{}, err
	}
	if !member {
		return Response{}, web.ErrForbidden("only the claimed business can respond to this review")
	}
	resp := Response{ID: uuid.New(), ReviewID: reviewID, BusinessID: businessID, AuthorUserID: userID, Body: body}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO business_responses (id, review_id, business_id, author_user_id, body)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at`,
		resp.ID, resp.ReviewID, resp.BusinessID, resp.AuthorUserID, resp.Body).
		Scan(&resp.CreatedAt, &resp.UpdatedAt)
	if err != nil {
		if isUnique(err) {
			return Response{}, web.ErrConflict("this review already has a response; edit it instead")
		}
		return Response{}, fmt.Errorf("inserting response: %w", err)
	}
	return resp, nil
}

// UpdateResponse edits the response, preserving the previous text in an
// append-only audit table, atomically.
func (r *Repo) UpdateResponse(ctx context.Context, responseID, userID uuid.UUID, body string) (Response, error) {
	body, err := validateResponseBody(body)
	if err != nil {
		return Response{}, err
	}
	var resp Response
	err = database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		var prevBody string
		err := tx.QueryRow(ctx, `
			SELECT id, review_id, business_id, author_user_id, body
			FROM business_responses WHERE id = $1 FOR UPDATE`, responseID).
			Scan(&resp.ID, &resp.ReviewID, &resp.BusinessID, &resp.AuthorUserID, &prevBody)
		if errors.Is(err, pgx.ErrNoRows) {
			return web.ErrNotFound("response")
		}
		if err != nil {
			return fmt.Errorf("loading response: %w", err)
		}
		var member bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM business_members WHERE business_id = $1 AND user_id = $2)`,
			resp.BusinessID, userID).Scan(&member); err != nil {
			return fmt.Errorf("checking membership: %w", err)
		}
		if !member {
			return web.ErrForbidden("")
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO business_response_edits (id, response_id, previous_body, edited_by)
			VALUES ($1, $2, $3, $4)`, uuid.New(), resp.ID, prevBody, userID); err != nil {
			return fmt.Errorf("recording response edit: %w", err)
		}
		if err := tx.QueryRow(ctx, `
			UPDATE business_responses SET body = $2, author_user_id = $3 WHERE id = $1
			RETURNING body, created_at, updated_at`, resp.ID, body, userID).
			Scan(&resp.Body, &resp.CreatedAt, &resp.UpdatedAt); err != nil {
			return fmt.Errorf("updating response: %w", err)
		}
		resp.AuthorUserID = userID
		return nil
	})
	if err != nil {
		return Response{}, err
	}
	return resp, nil
}

// BusinessStats is the owner dashboard summary across all the business's targets.
type BusinessStats struct {
	TargetCount   int        `json:"target_count"`
	ReviewCount   int        `json:"review_count"`
	AverageRating *float64   `json:"average_rating"`
	ResponseCount int        `json:"response_count"`
	OpenReports   int        `json:"open_reports_filed"`
	LastReviewAt  *time.Time `json:"last_review_at"`
}

// Stats aggregates published-review statistics for the business dashboard.
func (r *Repo) Stats(ctx context.Context, businessID uuid.UUID) (BusinessStats, error) {
	var s BusinessStats
	err := r.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM review_targets t WHERE t.business_id = $1 AND t.moderation_status = 'published'),
			count(rv.id),
			avg(rv.overall_rating)::float8,
			max(rv.created_at)
		FROM reviews rv
		JOIN review_targets t ON t.id = rv.target_id
		WHERE t.business_id = $1 AND rv.moderation_status = 'published'`, businessID).
		Scan(&s.TargetCount, &s.ReviewCount, &s.AverageRating, &s.LastReviewAt)
	if err != nil {
		return BusinessStats{}, fmt.Errorf("aggregating business stats: %w", err)
	}
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM business_responses WHERE business_id = $1`, businessID).Scan(&s.ResponseCount); err != nil {
		return BusinessStats{}, fmt.Errorf("counting responses: %w", err)
	}
	return s, nil
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
