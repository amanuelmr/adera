package targets

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/businesses"
	"github.com/adera-platform/backend/internal/platform/database"
	"github.com/adera-platform/backend/internal/platform/web"
)

// Prior parameters for the confidence-adjusted (Bayesian) ranking score.
// Documented in docs/rating-and-ranking.md; the raw average is always shown
// unaltered alongside it.
const (
	RankPriorMean   = 3.5
	RankPriorWeight = 10.0
)

// Repo provides target persistence.
type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const targetColumns = `t.id, t.target_type, t.category_id, t.business_id, t.name, t.slug, t.description,
	t.city_id, t.area_id, t.address_text, t.latitude, t.longitude, t.online_only, t.phone, t.website,
	t.social_links, t.verification_status, t.moderation_status, t.created_at, t.updated_at,
	coalesce(s.review_count, 0),
	CASE WHEN coalesce(s.review_count, 0) > 0 THEN round(s.rating_sum::numeric / s.review_count, 2)::float8 END`

const targetFrom = ` FROM review_targets t LEFT JOIN target_rating_stats s ON s.target_id = t.id `

func scanTarget(row pgx.Row) (Target, error) {
	var t Target
	err := row.Scan(&t.ID, &t.TargetType, &t.CategoryID, &t.BusinessID, &t.Name, &t.Slug, &t.Description,
		&t.CityID, &t.AreaID, &t.AddressText, &t.Latitude, &t.Longitude, &t.OnlineOnly, &t.Phone, &t.Website,
		&t.SocialLinks, &t.VerificationStatus, &t.ModerationStatus, &t.CreatedAt, &t.UpdatedAt,
		&t.ReviewCount, &t.AverageRating)
	if errors.Is(err, pgx.ErrNoRows) {
		return Target{}, web.ErrNotFound("target")
	}
	if err != nil {
		return Target{}, fmt.Errorf("scanning target: %w", err)
	}
	return t, nil
}

func validateCreateRefs(ctx context.Context, tx pgx.Tx, in CreateInput) error {
	var categoryActive bool
	if err := tx.QueryRow(ctx, `SELECT active FROM categories WHERE id = $1`, in.CategoryID).Scan(&categoryActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return web.ErrNotFound("category")
		}
		return fmt.Errorf("checking category: %w", err)
	}
	if !categoryActive {
		return web.ErrNotFound("category")
	}
	if in.BusinessID != nil {
		var active bool
		if err := tx.QueryRow(ctx, `SELECT status = 'active' FROM businesses WHERE id = $1`, *in.BusinessID).Scan(&active); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return web.ErrNotFound("business")
			}
			return fmt.Errorf("checking business: %w", err)
		}
		if !active {
			return web.ErrConflict("business is not active")
		}
	}
	return validateLocationRefs(ctx, tx, in.CityID, in.AreaID)
}

func validateLocationRefs(ctx context.Context, tx pgx.Tx, cityID, areaID *uuid.UUID) error {
	if areaID != nil && cityID == nil {
		return web.ErrValidation("invalid target").WithDetail("city_id", "required when area_id is provided")
	}
	if cityID != nil {
		var active bool
		if err := tx.QueryRow(ctx, `SELECT active FROM cities WHERE id = $1`, *cityID).Scan(&active); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return web.ErrNotFound("city")
			}
			return fmt.Errorf("checking city: %w", err)
		}
		if !active {
			return web.ErrNotFound("city")
		}
	}
	if areaID != nil {
		var belongs bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM areas
				WHERE id = $1 AND city_id = $2 AND active
			)`, *areaID, *cityID).Scan(&belongs); err != nil {
			return fmt.Errorf("checking area: %w", err)
		}
		if !belongs {
			return web.ErrValidation("invalid target").WithDetail("area_id", "must be an active area in the selected city")
		}
	}
	return nil
}

// CreateInput is the validated payload for a new target.
type CreateInput struct {
	TargetType  string
	CategoryID  uuid.UUID
	BusinessID  *uuid.UUID
	Name        string
	Description string
	CityID      *uuid.UUID
	AreaID      *uuid.UUID
	AddressText string
	Latitude    *float64
	Longitude   *float64
	OnlineOnly  bool
	Phone       string
	Website     string
	SocialLinks map[string]string
	Aliases     []string
	CreatedBy   uuid.UUID
	// Moderators publish immediately; public submissions enter moderation.
	AutoPublish bool
}

// Create inserts the target and its aliases in one transaction.
func (r *Repo) Create(ctx context.Context, in CreateInput) (Target, error) {
	id := uuid.New()
	slug := businesses.Slugify(in.Name, id)
	status := ModerationPending
	if in.AutoPublish {
		status = ModerationPublished
	}
	if in.SocialLinks == nil {
		in.SocialLinks = map[string]string{}
	}
	var t Target
	err := database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		if err := validateCreateRefs(ctx, tx, in); err != nil {
			return err
		}
		row := tx.QueryRow(ctx, `
			WITH ins AS (
				INSERT INTO review_targets
					(id, target_type, category_id, business_id, name, slug, description, city_id, area_id,
					 address_text, latitude, longitude, online_only, phone, website, social_links,
					 moderation_status, created_by)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
				RETURNING *
			)
			SELECT `+targetColumns+` FROM ins t LEFT JOIN target_rating_stats s ON s.target_id = t.id`,
			id, in.TargetType, in.CategoryID, in.BusinessID, in.Name, slug, in.Description,
			in.CityID, in.AreaID, in.AddressText, in.Latitude, in.Longitude, in.OnlineOnly,
			in.Phone, in.Website, in.SocialLinks, status, in.CreatedBy)
		var err error
		t, err = scanTarget(row)
		if err != nil {
			return err
		}
		for _, alias := range in.Aliases {
			if _, err := tx.Exec(ctx, `
				INSERT INTO target_aliases (id, target_id, alias, created_by)
				VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
				uuid.New(), id, alias, in.CreatedBy); err != nil {
				return fmt.Errorf("inserting alias: %w", err)
			}
		}
		// Stats row exists from day one so aggregate updates are pure UPDATEs.
		if _, err := tx.Exec(ctx, `INSERT INTO target_rating_stats (target_id) VALUES ($1)`, id); err != nil {
			return fmt.Errorf("initializing rating stats: %w", err)
		}
		return nil
	})
	if err != nil {
		return Target{}, err
	}
	t.Aliases = in.Aliases
	return t, nil
}

// GetByIDOrSlug loads one target with its aliases.
func (r *Repo) GetByIDOrSlug(ctx context.Context, idOrSlug string) (Target, error) {
	var (
		t   Target
		err error
	)
	if id, parseErr := uuid.Parse(idOrSlug); parseErr == nil {
		t, err = scanTarget(r.pool.QueryRow(ctx, `SELECT `+targetColumns+targetFrom+`WHERE t.id = $1`, id))
	} else {
		t, err = scanTarget(r.pool.QueryRow(ctx, `SELECT `+targetColumns+targetFrom+`WHERE t.slug = $1`, idOrSlug))
	}
	if err != nil {
		return Target{}, err
	}
	rows, err := r.pool.Query(ctx, `SELECT alias FROM target_aliases WHERE target_id = $1 ORDER BY alias`, t.ID)
	if err != nil {
		return Target{}, fmt.Errorf("querying aliases: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return Target{}, fmt.Errorf("scanning alias: %w", err)
		}
		t.Aliases = append(t.Aliases, a)
	}
	if err := rows.Err(); err != nil {
		return Target{}, fmt.Errorf("iterating aliases: %w", err)
	}
	return t, nil
}

// UpdateInput carries the member/moderator-editable fields (explicit
// allowlist; nil pointer = unchanged).
type UpdateInput struct {
	Description *string
	AddressText *string
	Phone       *string
	Website     *string
	CityID      *uuid.UUID
	AreaID      *uuid.UUID
	OnlineOnly  *bool
	SocialLinks map[string]string
}

// Update applies the allowlisted fields.
func (r *Repo) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (Target, error) {
	sets := []string{}
	args := []any{id}
	add := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if in.Description != nil {
		add("description", *in.Description)
	}
	if in.AddressText != nil {
		add("address_text", *in.AddressText)
	}
	if in.Phone != nil {
		add("phone", *in.Phone)
	}
	if in.Website != nil {
		add("website", *in.Website)
	}
	if in.CityID != nil {
		add("city_id", *in.CityID)
	}
	if in.AreaID != nil {
		add("area_id", *in.AreaID)
	}
	if in.OnlineOnly != nil {
		add("online_only", *in.OnlineOnly)
	}
	if in.SocialLinks != nil {
		add("social_links", in.SocialLinks)
	}
	if len(sets) == 0 {
		return r.GetByIDOrSlug(ctx, id.String())
	}
	err := database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		if in.CityID != nil || in.AreaID != nil {
			var currentCity, currentArea *uuid.UUID
			if err := tx.QueryRow(ctx, `SELECT city_id, area_id FROM review_targets WHERE id = $1 FOR UPDATE`, id).
				Scan(&currentCity, &currentArea); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return web.ErrNotFound("target")
				}
				return fmt.Errorf("loading target location: %w", err)
			}
			cityID, areaID := currentCity, currentArea
			if in.CityID != nil {
				cityID = in.CityID
			}
			if in.AreaID != nil {
				areaID = in.AreaID
			}
			if err := validateLocationRefs(ctx, tx, cityID, areaID); err != nil {
				return err
			}
		}
		tag, err := tx.Exec(ctx,
			`UPDATE review_targets SET `+strings.Join(sets, ", ")+` WHERE id = $1`, args...)
		if err != nil {
			return fmt.Errorf("updating target: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return web.ErrNotFound("target")
		}
		return nil
	})
	if err != nil {
		return Target{}, err
	}
	return r.GetByIDOrSlug(ctx, id.String())
}

// BrowseFilter selects published targets.
type BrowseFilter struct {
	CategoryID *uuid.UUID
	CityID     *uuid.UUID
	AreaID     *uuid.UUID
	BusinessID *uuid.UUID
	TargetType string
	MinRating  float64
	Verified   *bool
	OnlineOnly *bool
}

func (f BrowseFilter) where(args *[]any) []string {
	conds := []string{"t.moderation_status = 'published'"}
	add := func(cond string, v any) {
		*args = append(*args, v)
		conds = append(conds, fmt.Sprintf(cond, len(*args)))
	}
	if f.CategoryID != nil {
		add("t.category_id = $%d", *f.CategoryID)
	}
	if f.CityID != nil {
		add("t.city_id = $%d", *f.CityID)
	}
	if f.AreaID != nil {
		add("t.area_id = $%d", *f.AreaID)
	}
	if f.BusinessID != nil {
		add("t.business_id = $%d", *f.BusinessID)
	}
	if f.TargetType != "" {
		add("t.target_type = $%d", f.TargetType)
	}
	if f.MinRating > 0 {
		add("coalesce(s.rating_sum::float8 / nullif(s.review_count, 0), 0) >= $%d", f.MinRating)
	}
	if f.Verified != nil && *f.Verified {
		conds = append(conds, "t.verification_status = 'verified'")
	}
	if f.OnlineOnly != nil {
		add("t.online_only = $%d", *f.OnlineOnly)
	}
	return conds
}

// Browse lists published targets, newest first, with keyset pagination.
func (r *Repo) Browse(ctx context.Context, f BrowseFilter, cursor *web.Cursor, limit int) ([]Target, *web.Cursor, error) {
	var args []any
	conds := f.where(&args)
	if cursor != nil {
		ts, ok := cursor.Time()
		if !ok {
			return nil, nil, web.ErrValidation("invalid cursor")
		}
		args = append(args, ts)
		tsIdx := len(args)
		args = append(args, cursor.ID)
		conds = append(conds, fmt.Sprintf("(t.created_at, t.id) < ($%d, $%d)", tsIdx, len(args)))
	}
	args = append(args, limit+1)
	query := `SELECT ` + targetColumns + targetFrom + `
		WHERE ` + strings.Join(conds, " AND ") + `
		ORDER BY t.created_at DESC, t.id DESC
		LIMIT $` + fmt.Sprint(len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("browsing targets: %w", err)
	}
	defer rows.Close()
	var out []Target
	for rows.Next() {
		t, err := scanTarget(rows)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterating targets: %w", err)
	}
	var next *web.Cursor
	if len(out) > limit {
		out = out[:limit]
		last := out[limit-1]
		c := web.TimeCursor(last.CreatedAt, last.ID)
		next = &c
	}
	return out, next, nil
}

// RankedTarget is a target plus its ranking score for top-rated listings.
type RankedTarget struct {
	Target
	RankingScore float64 `json:"ranking_score"`
}

// TopRated returns published targets ordered by the documented
// confidence-adjusted score: (C*m + sum) / (C + n).
func (r *Repo) TopRated(ctx context.Context, f BrowseFilter, limit int) ([]RankedTarget, error) {
	var args []any
	conds := f.where(&args)
	conds = append(conds, "coalesce(s.review_count, 0) > 0")
	args = append(args, RankPriorWeight, RankPriorMean, limit)
	query := fmt.Sprintf(`SELECT `+targetColumns+`,
			round((($%d::float8 * $%d::float8 + coalesce(s.rating_sum, 0)) / ($%d::float8 + coalesce(s.review_count, 0)))::numeric, 4)::float8 AS rank_score
		`+targetFrom+`
		WHERE `+strings.Join(conds, " AND ")+`
		ORDER BY rank_score DESC, s.review_count DESC, t.id
		LIMIT $%d`, len(args)-2, len(args)-1, len(args)-2, len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying top-rated: %w", err)
	}
	defer rows.Close()
	var out []RankedTarget
	for rows.Next() {
		var t RankedTarget
		err := rows.Scan(&t.ID, &t.TargetType, &t.CategoryID, &t.BusinessID, &t.Name, &t.Slug, &t.Description,
			&t.CityID, &t.AreaID, &t.AddressText, &t.Latitude, &t.Longitude, &t.OnlineOnly, &t.Phone, &t.Website,
			&t.SocialLinks, &t.VerificationStatus, &t.ModerationStatus, &t.CreatedAt, &t.UpdatedAt,
			&t.ReviewCount, &t.AverageRating, &t.RankingScore)
		if err != nil {
			return nil, fmt.Errorf("scanning ranked target: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating ranked targets: %w", err)
	}
	return out, nil
}

// TrendingTarget adds recent activity to the target summary.
type TrendingTarget struct {
	Target
	RecentReviewCount int      `json:"recent_review_count"`
	RecentAverage     *float64 `json:"recent_average,omitempty"`
}

// Trending orders by published-review volume in the last 30 days ("popular
// recently" — simple, legible, documented).
func (r *Repo) Trending(ctx context.Context, f BrowseFilter, limit int) ([]TrendingTarget, error) {
	var args []any
	conds := f.where(&args)
	args = append(args, limit)
	query := `SELECT ` + targetColumns + `, rc.n, round(rc.avg_rating, 2)::float8
		FROM (
			SELECT rv.target_id, count(*) AS n, avg(rv.overall_rating) AS avg_rating
			FROM reviews rv
			WHERE rv.moderation_status = 'published' AND rv.created_at > now() - interval '30 days'
			GROUP BY rv.target_id
		) rc
		JOIN review_targets t ON t.id = rc.target_id
		LEFT JOIN target_rating_stats s ON s.target_id = t.id
		WHERE ` + strings.Join(conds, " AND ") + `
		ORDER BY rc.n DESC, rc.avg_rating DESC, t.id
		LIMIT $` + fmt.Sprint(len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying trending: %w", err)
	}
	defer rows.Close()
	var out []TrendingTarget
	for rows.Next() {
		var t TrendingTarget
		err := rows.Scan(&t.ID, &t.TargetType, &t.CategoryID, &t.BusinessID, &t.Name, &t.Slug, &t.Description,
			&t.CityID, &t.AreaID, &t.AddressText, &t.Latitude, &t.Longitude, &t.OnlineOnly, &t.Phone, &t.Website,
			&t.SocialLinks, &t.VerificationStatus, &t.ModerationStatus, &t.CreatedAt, &t.UpdatedAt,
			&t.ReviewCount, &t.AverageRating, &t.RecentReviewCount, &t.RecentAverage)
		if err != nil {
			return nil, fmt.Errorf("scanning trending target: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating trending targets: %w", err)
	}
	return out, nil
}

// CreateEditSuggestion records a community correction for moderator review.
func (r *Repo) CreateEditSuggestion(ctx context.Context, targetID, userID uuid.UUID, changes map[string]any, note string) (uuid.UUID, error) {
	if len(changes) == 0 {
		return uuid.Nil, web.ErrValidation("invalid suggestion").WithDetail("changes", "at least one change required")
	}
	for key := range changes {
		if !editSuggestionFields[key] {
			return uuid.Nil, web.ErrValidation("invalid suggestion").WithDetail("changes", "unsupported field: "+key)
		}
	}
	id := uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO target_edit_suggestions (id, target_id, user_id, changes, note)
		VALUES ($1, $2, $3, $4, $5)`, id, targetID, userID, changes, note)
	if err != nil {
		return uuid.Nil, fmt.Errorf("inserting edit suggestion: %w", err)
	}
	return id, nil
}

// touchUpdated is used by moderation flows that adjust status.
func (r *Repo) SetModerationStatus(ctx context.Context, id uuid.UUID, status string, mergedInto *uuid.UUID) error {
	return database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		return r.SetModerationStatusTx(ctx, tx, id, status, mergedInto)
	})
}

// SetModerationStatusTx is the transaction-scoped form used when the status
// decision must commit atomically with an audit row.
func (r *Repo) SetModerationStatusTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string, mergedInto *uuid.UUID) error {
	tag, err := tx.Exec(ctx, `
		UPDATE review_targets SET moderation_status = $2, merged_into_id = $3 WHERE id = $1`,
		id, status, mergedInto)
	if err != nil {
		return fmt.Errorf("setting target moderation status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return web.ErrNotFound("target")
	}
	return nil
}
