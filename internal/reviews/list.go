package reviews

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/adera-platform/backend/internal/platform/web"
)

// ListFilter narrows a target's published reviews.
type ListFilter struct {
	Language         string
	Rating           int // exact star filter (histogram tap)
	MinRating        int
	VerifiedOnly     bool
	DiscoverySource  string
	ExpectationMatch string
	Since            *time.Time
}

// Sort orders. "relevant" is a single-page sort (no cursor); its formula is
// documented in docs/rating-and-ranking.md.
const (
	SortNewest      = "newest"
	SortOldest      = "oldest"
	SortHighest     = "highest"
	SortLowest      = "lowest"
	SortMostHelpful = "most_helpful"
	SortRelevant    = "relevant"
)

var validSorts = map[string]bool{
	SortNewest: true, SortOldest: true, SortHighest: true,
	SortLowest: true, SortMostHelpful: true, SortRelevant: true,
}

// compositeCursor encodes multi-key sort positions as "part1|part2".
func compositeCursor(parts ...string) string { return strings.Join(parts, "|") }

// ListForTarget returns a page of published reviews for a target.
func (r *Repo) ListForTarget(ctx context.Context, targetID uuid.UUID, viewer uuid.UUID,
	f ListFilter, sort string, cursor *web.Cursor, limit int) ([]ListedReview, *web.Cursor, error) {

	if sort == "" {
		sort = SortNewest
	}
	if !validSorts[sort] {
		return nil, nil, web.ErrValidation("invalid sort").WithDetail("sort", "unsupported sort")
	}

	args := []any{targetID, viewer}
	conds := []string{"rv.target_id = $1", "rv.moderation_status = 'published'"}
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if f.Language != "" {
		add("rv.language = $%d", f.Language)
	}
	if f.Rating > 0 {
		add("rv.overall_rating = $%d", f.Rating)
	}
	if f.MinRating > 0 {
		add("rv.overall_rating >= $%d", f.MinRating)
	}
	if f.VerifiedOnly {
		conds = append(conds, "rv.verification_level IN ('receipt_submitted', 'location_verified', 'partner_verified')")
	}
	if f.DiscoverySource != "" {
		add("rv.discovery_source = $%d", f.DiscoverySource)
	}
	if f.ExpectationMatch != "" {
		add("rv.expectation_match = $%d", f.ExpectationMatch)
	}
	if f.Since != nil {
		add("rv.created_at >= $%d", *f.Since)
	}

	// Cursor condition + order clause per sort.
	var orderBy string
	switch sort {
	case SortNewest:
		orderBy = "rv.created_at DESC, rv.id DESC"
		if cursor != nil {
			ts, ok := cursor.Time()
			if !ok {
				return nil, nil, web.ErrValidation("invalid cursor")
			}
			args = append(args, ts, cursor.ID)
			conds = append(conds, fmt.Sprintf("(rv.created_at, rv.id) < ($%d, $%d)", len(args)-1, len(args)))
		}
	case SortOldest:
		orderBy = "rv.created_at ASC, rv.id ASC"
		if cursor != nil {
			ts, ok := cursor.Time()
			if !ok {
				return nil, nil, web.ErrValidation("invalid cursor")
			}
			args = append(args, ts, cursor.ID)
			conds = append(conds, fmt.Sprintf("(rv.created_at, rv.id) > ($%d, $%d)", len(args)-1, len(args)))
		}
	case SortHighest, SortLowest:
		dir, cmp := "DESC", "<"
		if sort == SortLowest {
			dir, cmp = "ASC", ">"
		}
		orderBy = fmt.Sprintf("rv.overall_rating %s, rv.created_at %s, rv.id %s", dir, dir, dir)
		if cursor != nil {
			rating, ts, ok := splitRatingCursor(cursor.K)
			if !ok {
				return nil, nil, web.ErrValidation("invalid cursor")
			}
			args = append(args, rating, ts, cursor.ID)
			conds = append(conds, fmt.Sprintf("(rv.overall_rating, rv.created_at, rv.id) %s ($%d, $%d, $%d)",
				cmp, len(args)-2, len(args)-1, len(args)))
		}
	case SortMostHelpful:
		orderBy = "hc.n DESC, rv.created_at DESC, rv.id DESC"
		if cursor != nil {
			votes, ts, ok := splitRatingCursor(cursor.K)
			if !ok {
				return nil, nil, web.ErrValidation("invalid cursor")
			}
			args = append(args, votes, ts, cursor.ID)
			conds = append(conds, fmt.Sprintf("(hc.n, rv.created_at, rv.id) < ($%d, $%d, $%d)",
				len(args)-2, len(args)-1, len(args)))
		}
	case SortRelevant:
		// helpful votes + verified boost − age decay; single page.
		orderBy = `(hc.n
			+ CASE WHEN rv.verification_level IN ('receipt_submitted','location_verified','partner_verified') THEN 2 ELSE 0 END
			- extract(epoch FROM now() - rv.created_at) / 86400.0 / 30.0) DESC, rv.id DESC`
	}

	args = append(args, limit+1)
	query := `
		SELECT ` + prefixed(reviewColumns, "rv.") + `,
			u.display_name,
			(SELECT count(*) FROM reviews r2 WHERE r2.user_id = rv.user_id AND r2.moderation_status = 'published'),
			hc.n,
			($2::uuid IS NOT NULL AND EXISTS (
				SELECT 1 FROM helpful_votes hv WHERE hv.review_id = rv.id AND hv.user_id = $2)),
			br.id, br.body, br.created_at, br.updated_at
		FROM reviews rv
		JOIN users u ON u.id = rv.user_id
		CROSS JOIN LATERAL (SELECT count(*)::int AS n FROM helpful_votes hv WHERE hv.review_id = rv.id) hc
		LEFT JOIN business_responses br ON br.review_id = rv.id
		WHERE ` + strings.Join(conds, " AND ") + `
		ORDER BY ` + orderBy + `
		LIMIT $` + strconv.Itoa(len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("listing reviews: %w", err)
	}
	defer rows.Close()

	var out []ListedReview
	for rows.Next() {
		var (
			lr       ListedReview
			expDate  *time.Time
			src      *string
			match    *string
			respID   *uuid.UUID
			respBody *string
			respCAt  *time.Time
			respUAt  *time.Time
		)
		err := rows.Scan(&lr.ID, &lr.TargetID, &lr.UserID, &lr.OverallRating, &lr.Title, &lr.Body, &lr.Language,
			&expDate, &lr.PricePaid, &lr.Currency, &lr.WouldRecommend, &lr.ReturnLikelihood,
			&src, &match, &lr.SocialMediaURL, &lr.IncentiveType, &lr.MaterialConnection,
			&lr.DisclosureDetails, &lr.VerificationLevel,
			&lr.ModerationStatus, &lr.EditCount, &lr.EditedAt, &lr.Version, &lr.CreatedAt, &lr.UpdatedAt,
			&lr.ReviewerName, &lr.ReviewerReviewCount, &lr.HelpfulCount, &lr.ViewerVoted,
			&respID, &respBody, &respCAt, &respUAt)
		if err != nil {
			return nil, nil, fmt.Errorf("scanning listed review: %w", err)
		}
		lr.ExperienceDate = expDate
		if src != nil {
			lr.DiscoverySource = *src
		}
		if match != nil {
			lr.ExpectationMatch = *match
		}
		if respID != nil {
			lr.Response = &struct {
				ID        uuid.UUID `json:"id"`
				Body      string    `json:"body"`
				CreatedAt time.Time `json:"created_at"`
				UpdatedAt time.Time `json:"updated_at"`
			}{ID: *respID, Body: *respBody, CreatedAt: *respCAt, UpdatedAt: *respUAt}
		}
		out = append(out, lr)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterating listed reviews: %w", err)
	}

	var next *web.Cursor
	if len(out) > limit {
		out = out[:limit]
		last := out[limit-1]
		switch sort {
		case SortNewest, SortOldest:
			c := web.TimeCursor(last.CreatedAt, last.ID)
			next = &c
		case SortHighest, SortLowest:
			c := web.Cursor{K: compositeCursor(strconv.Itoa(last.OverallRating),
				last.CreatedAt.UTC().Format(time.RFC3339Nano)), ID: last.ID}
			next = &c
		case SortMostHelpful:
			c := web.Cursor{K: compositeCursor(strconv.Itoa(last.HelpfulCount),
				last.CreatedAt.UTC().Format(time.RFC3339Nano)), ID: last.ID}
			next = &c
		}
	}

	if err := r.attachMedia(ctx, out); err != nil {
		return nil, nil, err
	}
	return out, next, nil
}

// ListForUser returns the caller's own reviews across all statuses.
func (r *Repo) ListForUser(ctx context.Context, userID uuid.UUID, cursor *web.Cursor, limit int) ([]Review, *web.Cursor, error) {
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
	rows, err := r.pool.Query(ctx, `
		SELECT `+reviewColumns+` FROM reviews
		WHERE user_id = $1`+cond+`
		ORDER BY created_at DESC, id DESC
		LIMIT $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, nil, fmt.Errorf("listing own reviews: %w", err)
	}
	defer rows.Close()
	var out []Review
	for rows.Next() {
		rv, err := scanReview(rows)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, rv)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterating own reviews: %w", err)
	}
	var next *web.Cursor
	if len(out) > limit {
		out = out[:limit]
		c := web.TimeCursor(out[limit-1].CreatedAt, out[limit-1].ID)
		next = &c
	}
	return out, next, nil
}

// attachMedia loads ready public media for the listed page. Only object keys
// in the PUBLIC bucket ever appear here; private evidence is never listed.
func (r *Repo) attachMedia(ctx context.Context, page []ListedReview) error {
	if len(page) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(page))
	index := make(map[uuid.UUID]int, len(page))
	for i, lr := range page {
		ids[i] = lr.ID
		index[lr.ID] = i
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, review_id, object_key, thumb_key FROM review_media
		WHERE review_id = ANY($1) AND status = 'ready'
		ORDER BY created_at`, ids)
	if err != nil {
		return fmt.Errorf("querying review media: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var mediaID, reviewID uuid.UUID
		var key string
		var thumbKey *string
		if err := rows.Scan(&mediaID, &reviewID, &key, &thumbKey); err != nil {
			return fmt.Errorf("scanning review media: %w", err)
		}
		ref := MediaRef{ID: mediaID, URL: r.publicMediaURL(key)}
		if thumbKey != nil {
			ref.ThumbURL = r.publicMediaURL(*thumbKey)
		}
		i := index[reviewID]
		page[i].Media = append(page[i].Media, ref)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating review media: %w", err)
	}
	return nil
}

func (r *Repo) publicMediaURL(key string) string {
	if r.publicMediaBaseURL == "" {
		return key
	}
	return r.publicMediaBaseURL + "/" + key
}

func splitRatingCursor(k string) (int, time.Time, bool) {
	parts := strings.SplitN(k, "|", 2)
	if len(parts) != 2 {
		return 0, time.Time{}, false
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, time.Time{}, false
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[1])
	if err != nil {
		return 0, time.Time{}, false
	}
	return n, ts, true
}

// prefixed prepends p to each comma-separated column name.
func prefixed(cols, p string) string {
	parts := strings.Split(cols, ",")
	for i, c := range parts {
		parts[i] = p + strings.TrimSpace(c)
	}
	return strings.Join(parts, ", ")
}
