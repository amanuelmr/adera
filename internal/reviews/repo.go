package reviews

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/platform/database"
	"github.com/adera-platform/backend/internal/platform/web"
)

// Repo persists reviews and keeps target_rating_stats consistent within the
// same transaction as every mutation.
type Repo struct {
	pool *pgxpool.Pool
	// publicMediaBaseURL turns public object keys into display URLs.
	publicMediaBaseURL string
}

func NewRepo(pool *pgxpool.Pool, publicMediaBaseURL string) *Repo {
	return &Repo{pool: pool, publicMediaBaseURL: strings.TrimSuffix(publicMediaBaseURL, "/")}
}

// Pool is exposed for sibling modules (media, moderation) composing
// transactions that touch reviews.
func (r *Repo) Pool() *pgxpool.Pool { return r.pool }

const reviewColumns = `id, target_id, user_id, overall_rating, title, body, language,
	experience_date, price_paid, currency, would_recommend, return_likelihood,
	discovery_source, expectation_match, social_media_url, verification_level,
	moderation_status, edit_count, edited_at, version, created_at, updated_at`

func scanReview(row pgx.Row) (Review, error) {
	var (
		rv      Review
		expDate *time.Time
		src     *string
		match   *string
	)
	err := row.Scan(&rv.ID, &rv.TargetID, &rv.UserID, &rv.OverallRating, &rv.Title, &rv.Body, &rv.Language,
		&expDate, &rv.PricePaid, &rv.Currency, &rv.WouldRecommend, &rv.ReturnLikelihood,
		&src, &match, &rv.SocialMediaURL, &rv.VerificationLevel,
		&rv.ModerationStatus, &rv.EditCount, &rv.EditedAt, &rv.Version, &rv.CreatedAt, &rv.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Review{}, web.ErrNotFound("review")
	}
	if err != nil {
		return Review{}, fmt.Errorf("scanning review: %w", err)
	}
	rv.ExperienceDate = expDate
	if src != nil {
		rv.DiscoverySource = *src
	}
	if match != nil {
		rv.ExpectationMatch = *match
	}
	return rv, nil
}

// statsDelta applies one review's contribution to target_rating_stats with
// direction +1 (count it) or -1 (uncount it). Callers hold the transaction.
func statsDelta(ctx context.Context, tx pgx.Tx, rv Review, d int) error {
	verified := 0
	if VerifiedLevels[rv.VerificationLevel] {
		verified = 1
	}
	recTotal, recYes := 0, 0
	if rv.WouldRecommend != nil {
		recTotal = 1
		if *rv.WouldRecommend {
			recYes = 1
		}
	}
	retCount, retSum := 0, 0
	if rv.ReturnLikelihood != nil {
		retCount = 1
		retSum = *rv.ReturnLikelihood
	}
	tag, err := tx.Exec(ctx, fmt.Sprintf(`
		UPDATE target_rating_stats SET
			review_count    = review_count + $2,
			rating_sum      = rating_sum + $3,
			count_%d        = count_%d + $2,
			verified_count  = verified_count + $4,
			verified_sum    = verified_sum + $5,
			recommend_yes   = recommend_yes + $6,
			recommend_total = recommend_total + $7,
			return_sum      = return_sum + $8,
			return_count    = return_count + $9,
			updated_at      = now()
		WHERE target_id = $1`, rv.OverallRating, rv.OverallRating),
		rv.TargetID, d, d*rv.OverallRating, d*verified, d*verified*rv.OverallRating,
		d*recYes, d*recTotal, d*retSum, d*retCount)
	if err != nil {
		return fmt.Errorf("updating rating stats: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Stats rows are created with the target; absence is a bug, but
		// self-heal instead of failing the user's write.
		if _, err := tx.Exec(ctx, `
			INSERT INTO target_rating_stats (target_id) VALUES ($1) ON CONFLICT DO NOTHING`, rv.TargetID); err != nil {
			return fmt.Errorf("initializing missing stats row: %w", err)
		}
		return statsDelta(ctx, tx, rv, d)
	}
	return nil
}

type criterionDef struct {
	ID       uuid.UUID
	Code     string
	Required bool
	Min, Max int
}

// loadCriteria returns the active criteria for a category, keyed by code.
func loadCriteria(ctx context.Context, tx pgx.Tx, categoryID uuid.UUID) (map[string]criterionDef, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, code, required, scale_min, scale_max
		FROM category_criteria WHERE category_id = $1 AND active`, categoryID)
	if err != nil {
		return nil, fmt.Errorf("querying criteria: %w", err)
	}
	defer rows.Close()
	defs := make(map[string]criterionDef)
	for rows.Next() {
		var d criterionDef
		if err := rows.Scan(&d.ID, &d.Code, &d.Required, &d.Min, &d.Max); err != nil {
			return nil, fmt.Errorf("scanning criterion: %w", err)
		}
		defs[d.Code] = d
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating criteria: %w", err)
	}
	return defs, nil
}

func validateScores(scores map[string]int, defs map[string]criterionDef) error {
	e := web.ErrValidation("invalid review")
	for code, score := range scores {
		def, ok := defs[code]
		if !ok {
			return e.WithDetail("criterion_scores", "unknown criterion: "+code)
		}
		if score < def.Min || score > def.Max {
			return e.WithDetail("criterion_scores",
				fmt.Sprintf("%s must be %d-%d", code, def.Min, def.Max))
		}
	}
	for code, def := range defs {
		if def.Required {
			if _, ok := scores[code]; !ok {
				return e.WithDetail("criterion_scores", "missing required criterion: "+code)
			}
		}
	}
	return nil
}

func insertScores(ctx context.Context, tx pgx.Tx, reviewID uuid.UUID, scores map[string]int, defs map[string]criterionDef) error {
	for code, score := range scores {
		if _, err := tx.Exec(ctx, `
			INSERT INTO review_criterion_scores (review_id, criterion_id, score)
			VALUES ($1, $2, $3)`, reviewID, defs[code].ID, score); err != nil {
			return fmt.Errorf("inserting criterion score: %w", err)
		}
	}
	return nil
}

// Create submits a review: target checks, cooldown, anti-flooding cap,
// criterion validation, insert, and aggregate update — one transaction.
// Reviews publish immediately (documented policy in docs/moderation-policy.md)
// and enter the moderation flow via reports.
func (r *Repo) Create(ctx context.Context, userID uuid.UUID, in Input) (Review, error) {
	if err := in.Validate(); err != nil {
		return Review{}, err
	}
	var rv Review
	err := database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		var categoryID uuid.UUID
		var targetStatus string
		err := tx.QueryRow(ctx, `
			SELECT category_id, moderation_status FROM review_targets WHERE id = $1`, in.TargetID).
			Scan(&categoryID, &targetStatus)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && targetStatus != "published") {
			return web.ErrNotFound("target")
		}
		if err != nil {
			return fmt.Errorf("loading target: %w", err)
		}

		// Cooldown: within the window the user updates their existing review.
		var lastAt time.Time
		var lastID uuid.UUID
		err = tx.QueryRow(ctx, `
			SELECT id, created_at FROM reviews
			WHERE user_id = $1 AND target_id = $2 AND moderation_status NOT IN ('removed', 'rejected')
			ORDER BY created_at DESC LIMIT 1`, userID, in.TargetID).Scan(&lastID, &lastAt)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("checking previous review: %w", err)
		}
		if err == nil && time.Since(lastAt) < RepeatCooldown {
			return (&web.Error{
				Status: http.StatusConflict, Code: web.CodeCooldownActive,
				Message: "you reviewed this recently; update your existing review instead",
			}).WithDetail("existing_review_id", lastID.String())
		}

		// Anti-flooding cap.
		var todayCount int
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FROM reviews
			WHERE user_id = $1 AND created_at > now() - interval '24 hours'`, userID).Scan(&todayCount); err != nil {
			return fmt.Errorf("checking daily cap: %w", err)
		}
		if todayCount >= MaxReviewsPerDay {
			return web.ErrRateLimited()
		}

		defs, err := loadCriteria(ctx, tx, categoryID)
		if err != nil {
			return err
		}
		if err := validateScores(in.CriterionScores, defs); err != nil {
			return err
		}

		id := uuid.New()
		row := tx.QueryRow(ctx, `
			INSERT INTO reviews (id, target_id, user_id, overall_rating, title, body, language,
				experience_date, price_paid, currency, would_recommend, return_likelihood,
				discovery_source, expectation_match, social_media_url, moderation_status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,nullif($13,''),nullif($14,''),$15,'published')
			RETURNING `+reviewColumns,
			id, in.TargetID, userID, in.OverallRating, in.Title, in.Body, in.Language,
			in.ExperienceDate, in.PricePaid, in.Currency, in.WouldRecommend, in.ReturnLikelihood,
			in.DiscoverySource, in.ExpectationMatch, in.SocialMediaURL)
		rv, err = scanReview(row)
		if err != nil {
			return err
		}
		if err := insertScores(ctx, tx, rv.ID, in.CriterionScores, defs); err != nil {
			return err
		}
		rv.CriterionScores = in.CriterionScores
		return statsDelta(ctx, tx, rv, +1)
	})
	if err != nil {
		return Review{}, err
	}
	return rv, nil
}

// Update edits the caller's own review with optimistic concurrency (version
// must match) and rebalances aggregates in the same transaction.
func (r *Repo) Update(ctx context.Context, userID, reviewID uuid.UUID, expectedVersion int, in Input) (Review, error) {
	if err := in.Validate(); err != nil {
		return Review{}, err
	}
	var rv Review
	err := database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		old, err := scanReview(tx.QueryRow(ctx, `
			SELECT `+reviewColumns+` FROM reviews WHERE id = $1 FOR UPDATE`, reviewID))
		if err != nil {
			return err
		}
		// Object-level authorization: only the author edits their review.
		if old.UserID != userID {
			return web.ErrForbidden("you can only edit your own review")
		}
		switch old.ModerationStatus {
		case StatusRemoved, StatusRejected:
			return web.ErrConflict("this review can no longer be edited")
		}
		if old.Version != expectedVersion {
			return web.ErrPreconditionFailed("the review changed since you loaded it; reload and retry")
		}
		if in.TargetID != uuid.Nil && in.TargetID != old.TargetID {
			return web.ErrValidation("invalid review").WithDetail("target_id", "cannot move a review to another target")
		}

		var categoryID uuid.UUID
		if err := tx.QueryRow(ctx, `
			SELECT category_id FROM review_targets WHERE id = $1`, old.TargetID).Scan(&categoryID); err != nil {
			return fmt.Errorf("loading target category: %w", err)
		}
		defs, err := loadCriteria(ctx, tx, categoryID)
		if err != nil {
			return err
		}
		if err := validateScores(in.CriterionScores, defs); err != nil {
			return err
		}

		if old.ModerationStatus == StatusPublished {
			if err := statsDelta(ctx, tx, old, -1); err != nil {
				return err
			}
		}

		row := tx.QueryRow(ctx, `
			UPDATE reviews SET
				overall_rating = $2, title = $3, body = $4, language = $5,
				experience_date = $6, price_paid = $7, currency = $8,
				would_recommend = $9, return_likelihood = $10,
				discovery_source = nullif($11, ''), expectation_match = nullif($12, ''),
				social_media_url = $13,
				edit_count = edit_count + 1, edited_at = now(), version = version + 1
			WHERE id = $1
			RETURNING `+reviewColumns,
			reviewID, in.OverallRating, in.Title, in.Body, in.Language,
			in.ExperienceDate, in.PricePaid, in.Currency, in.WouldRecommend, in.ReturnLikelihood,
			in.DiscoverySource, in.ExpectationMatch, in.SocialMediaURL)
		rv, err = scanReview(row)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM review_criterion_scores WHERE review_id = $1`, reviewID); err != nil {
			return fmt.Errorf("clearing criterion scores: %w", err)
		}
		if err := insertScores(ctx, tx, reviewID, in.CriterionScores, defs); err != nil {
			return err
		}
		rv.CriterionScores = in.CriterionScores
		if rv.ModerationStatus == StatusPublished {
			return statsDelta(ctx, tx, rv, +1)
		}
		return nil
	})
	if err != nil {
		return Review{}, err
	}
	return rv, nil
}

// Remove takes the caller's own review out of public view (soft delete: the
// record is retained for audit; aggregates are adjusted).
func (r *Repo) Remove(ctx context.Context, userID, reviewID uuid.UUID) error {
	return database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		rv, err := scanReview(tx.QueryRow(ctx, `
			SELECT `+reviewColumns+` FROM reviews WHERE id = $1 FOR UPDATE`, reviewID))
		if err != nil {
			return err
		}
		if rv.UserID != userID {
			return web.ErrForbidden("you can only remove your own review")
		}
		if rv.ModerationStatus == StatusRemoved {
			return nil // idempotent
		}
		wasPublished := rv.ModerationStatus == StatusPublished
		if _, err := tx.Exec(ctx, `
			UPDATE reviews SET moderation_status = 'removed' WHERE id = $1`, reviewID); err != nil {
			return fmt.Errorf("removing review: %w", err)
		}
		if wasPublished {
			return statsDelta(ctx, tx, rv, -1)
		}
		return nil
	})
}

// Get loads one review with its criterion scores (keyed by criterion code).
func (r *Repo) Get(ctx context.Context, reviewID uuid.UUID) (Review, error) {
	rv, err := scanReview(r.pool.QueryRow(ctx, `
		SELECT `+reviewColumns+` FROM reviews WHERE id = $1`, reviewID))
	if err != nil {
		return Review{}, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT c.code, s.score
		FROM review_criterion_scores s JOIN category_criteria c ON c.id = s.criterion_id
		WHERE s.review_id = $1`, reviewID)
	if err != nil {
		return Review{}, fmt.Errorf("querying criterion scores: %w", err)
	}
	defer rows.Close()
	rv.CriterionScores = map[string]int{}
	for rows.Next() {
		var code string
		var score int
		if err := rows.Scan(&code, &score); err != nil {
			return Review{}, fmt.Errorf("scanning criterion score: %w", err)
		}
		rv.CriterionScores[code] = score
	}
	if err := rows.Err(); err != nil {
		return Review{}, fmt.Errorf("iterating criterion scores: %w", err)
	}
	return rv, nil
}

// SetModerationStatus transitions a review's moderation state (moderator
// action), adjusting aggregates when the review enters or leaves the
// published set. Returns the previous status.
func (r *Repo) SetModerationStatus(ctx context.Context, reviewID uuid.UUID, newStatus string) (string, error) {
	var oldStatus string
	err := database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		oldStatus, err = r.SetModerationStatusTx(ctx, tx, reviewID, newStatus)
		return err
	})
	if err != nil {
		return "", err
	}
	return oldStatus, nil
}

// SetModerationStatusTx is the transaction-scoped form used by moderation
// flows that must commit the decision and audit row atomically.
func (r *Repo) SetModerationStatusTx(ctx context.Context, tx pgx.Tx, reviewID uuid.UUID, newStatus string) (string, error) {
	rv, err := scanReview(tx.QueryRow(ctx, `
		SELECT `+reviewColumns+` FROM reviews WHERE id = $1 FOR UPDATE`, reviewID))
	if err != nil {
		return "", err
	}
	oldStatus := rv.ModerationStatus
	if oldStatus == newStatus {
		return oldStatus, nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE reviews SET moderation_status = $2 WHERE id = $1`, reviewID, newStatus); err != nil {
		return "", fmt.Errorf("setting moderation status: %w", err)
	}
	if oldStatus == StatusPublished && newStatus != StatusPublished {
		return oldStatus, statsDelta(ctx, tx, rv, -1)
	}
	if oldStatus != StatusPublished && newStatus == StatusPublished {
		return oldStatus, statsDelta(ctx, tx, rv, +1)
	}
	return oldStatus, nil
}

var levelRank = map[string]int{
	VerifyNone: 0, VerifyMedia: 1, VerifyReceipt: 2, VerifyLocation: 3, VerifyPartner: 4,
}

// UpgradeVerification raises a review's verification level (levels never
// downgrade automatically), adjusting verified aggregates when the review is
// published and crosses into the verified set.
func (r *Repo) UpgradeVerification(ctx context.Context, reviewID uuid.UUID, newLevel string) error {
	if _, ok := levelRank[newLevel]; !ok {
		return fmt.Errorf("unknown verification level %q", newLevel)
	}
	return database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		return r.UpgradeVerificationTx(ctx, tx, reviewID, newLevel)
	})
}

// UpgradeVerificationTx raises a review's verification level inside an existing
// transaction.
func (r *Repo) UpgradeVerificationTx(ctx context.Context, tx pgx.Tx, reviewID uuid.UUID, newLevel string) error {
	if _, ok := levelRank[newLevel]; !ok {
		return fmt.Errorf("unknown verification level %q", newLevel)
	}
	rv, err := scanReview(tx.QueryRow(ctx, `
		SELECT `+reviewColumns+` FROM reviews WHERE id = $1 FOR UPDATE`, reviewID))
	if err != nil {
		return err
	}
	if levelRank[newLevel] <= levelRank[rv.VerificationLevel] {
		return nil // idempotent; never downgrade
	}
	if _, err := tx.Exec(ctx, `
		UPDATE reviews SET verification_level = $2 WHERE id = $1`, reviewID, newLevel); err != nil {
		return fmt.Errorf("upgrading verification level: %w", err)
	}
	crossedIntoVerified := !VerifiedLevels[rv.VerificationLevel] && VerifiedLevels[newLevel]
	if crossedIntoVerified && rv.ModerationStatus == StatusPublished {
		if _, err := tx.Exec(ctx, `
			UPDATE target_rating_stats SET
				verified_count = verified_count + 1,
				verified_sum = verified_sum + $2,
				updated_at = now()
			WHERE target_id = $1`, rv.TargetID, rv.OverallRating); err != nil {
			return fmt.Errorf("updating verified aggregates: %w", err)
		}
	}
	return nil
}
