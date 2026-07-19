// Package search implements Unicode-aware target search over PostgreSQL:
// full-text ('simple' config; no Amharic stemmer exists and Ethiopic has no
// case), pg_trgm word-similarity for typo tolerance, and alias matching for
// mixed-script and transliterated names (e.g., "Bole Cafe" ⇄ "ቦሌ ካፌ").
// Both sides of every comparison pass through am_fold(lower(...)) — the
// homophone-folding function defined in migrations/0001.
package search

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/platform/web"
	"github.com/adera-platform/backend/internal/targets"
)

// similarityThreshold is tuned low because Ethiopic syllables carry more
// information per code point than Latin letters (see
// docs/research/tech-stack-decisions.md §4); validated by integration tests.
const similarityThreshold = 0.25

// Result is a search hit with its relevance score.
type Result struct {
	targets.Target
	Score float64 `json:"score"`
}

// Repo runs search queries.
type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// Search returns published targets matching q, best first. Relevance combines
// full-text rank (weighted highest), word similarity over name and aliases,
// and a small rating-confidence boost so well-reviewed places win ties.
func (r *Repo) Search(ctx context.Context, q string, f targets.BrowseFilter, limit int) ([]Result, error) {
	var args []any
	args = append(args, q)
	qIdx := len(args)

	conds := []string{"t.moderation_status = 'published'"}
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
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
	args = append(args, similarityThreshold)
	thIdx := len(args)
	args = append(args, limit)
	limitIdx := len(args)

	// The tsvector expression matches the GIN index in migrations/0009 exactly.
	query := fmt.Sprintf(`
		WITH qn AS (SELECT am_fold(lower($%d)) AS v)
		SELECT t.id, t.target_type, t.category_id, t.business_id, t.name, t.slug, t.description,
			t.city_id, t.area_id, t.address_text, t.latitude, t.longitude, t.online_only, t.phone, t.website,
			t.social_links, t.verification_status, t.moderation_status, t.created_at, t.updated_at,
			coalesce(s.review_count, 0),
			CASE WHEN coalesce(s.review_count, 0) > 0 THEN round(s.rating_sum::numeric / s.review_count, 2)::float8 END,
			round((
				2.0 * ts_rank(to_tsvector('simple', am_fold(lower(t.name)) || ' ' || am_fold(lower(t.description))),
				              websearch_to_tsquery('simple', qn.v))
				+ greatest(nm.sim, coalesce(al.sim, 0))
				+ coalesce(s.rating_sum::float8 / nullif(s.review_count, 0), 0) / 50.0
			)::numeric, 4)::float8 AS score
		FROM review_targets t
		CROSS JOIN qn
		LEFT JOIN target_rating_stats s ON s.target_id = t.id
		CROSS JOIN LATERAL (SELECT word_similarity(qn.v, am_fold(lower(t.name))) AS sim) nm
		LEFT JOIN LATERAL (
			SELECT max(word_similarity(qn.v, am_fold(lower(a.alias)))) AS sim
			FROM target_aliases a WHERE a.target_id = t.id
		) al ON true
		WHERE %s
		  AND (
			to_tsvector('simple', am_fold(lower(t.name)) || ' ' || am_fold(lower(t.description)))
				@@ websearch_to_tsquery('simple', qn.v)
			OR nm.sim >= $%d
			OR coalesce(al.sim, 0) >= $%d
		  )
		ORDER BY score DESC, t.id
		LIMIT $%d`, qIdx, strings.Join(conds, " AND "), thIdx, thIdx, limitIdx)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("searching targets: %w", err)
	}
	defer rows.Close()
	var out []Result
	for rows.Next() {
		var t Result
		err := rows.Scan(&t.ID, &t.TargetType, &t.CategoryID, &t.BusinessID, &t.Name, &t.Slug, &t.Description,
			&t.CityID, &t.AreaID, &t.AddressText, &t.Latitude, &t.Longitude, &t.OnlineOnly, &t.Phone, &t.Website,
			&t.SocialLinks, &t.VerificationStatus, &t.ModerationStatus, &t.CreatedAt, &t.UpdatedAt,
			&t.ReviewCount, &t.AverageRating, &t.Score)
		if err != nil {
			return nil, fmt.Errorf("scanning search result: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating search results: %w", err)
	}
	return out, nil
}

// Handler serves the search endpoint.
type Handler struct {
	repo    *Repo
	limiter interface{ Allow(string) bool }
}

func NewHandler(repo *Repo, limiter interface{ Allow(string) bool }) *Handler {
	return &Handler{repo: repo, limiter: limiter}
}

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/search/targets", h.search)
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	if !h.limiter.Allow("search:" + web.ClientIP(r, false)) {
		web.RespondError(w, r, web.ErrRateLimited())
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if l := len([]rune(q)); l < 2 || l > 120 {
		web.RespondError(w, r, web.ErrValidation("invalid search").WithDetail("q", "must be 2-120 characters"))
		return
	}
	f, err := targets.ParseBrowseFilter(r)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	results, err := h.repo.Search(r.Context(), q, f, web.ParseLimit(r))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, results)
}
