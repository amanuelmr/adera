// Package ratings exposes read-side rating aggregates and the restaurant/café
// "Reality Check". Formulas, rounding, and minimum-sample rules are documented
// in docs/rating-and-ranking.md; write-side aggregate maintenance lives in
// internal/reviews (same-transaction deltas).
package ratings

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/platform/web"
)

// Windows and sample rules.
const (
	RecentWindow = 90 * 24 * time.Hour // "recent" average window
	// Minimum published reviews before percentages/trends are asserted.
	MinSampleForPercentages = 5
	MinSampleForTrend       = 3
)

// Confidence labels for small samples.
func confidence(n int) string {
	switch {
	case n < MinSampleForTrend:
		return "none"
	case n < 10:
		return "low"
	case n < 30:
		return "medium"
	default:
		return "high"
	}
}

// round2 rounds half-away-from-zero to 2 decimals (documented rounding rule).
func round2(f float64) float64 { return math.Round(f*100) / 100 }

// TargetStats is the full public aggregate for one target.
type TargetStats struct {
	TargetID     uuid.UUID `json:"target_id"`
	ReviewCount  int       `json:"review_count"`
	Average      *float64  `json:"average,omitempty"`
	Distribution [5]int    `json:"distribution"` // index 0 = 1★

	VerifiedCount   int      `json:"verified_count"`
	VerifiedAverage *float64 `json:"verified_average,omitempty"`

	RecentCount   int      `json:"recent_count"`
	RecentAverage *float64 `json:"recent_average,omitempty"`

	RecommendPercent   *float64 `json:"recommend_percent,omitempty"`
	RecommendSampleN   int      `json:"recommend_sample_size"`
	ReturnIntentAvg    *float64 `json:"return_intent_average,omitempty"`
	ReturnIntentSample int      `json:"return_intent_sample_size"`

	CriterionAverages []CriterionAverage `json:"criterion_averages"`
	RankingScore      float64            `json:"ranking_score"`
	Confidence        string             `json:"confidence"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

// CriterionAverage is the mean score for one category criterion.
type CriterionAverage struct {
	Code    string  `json:"code"`
	Name    string  `json:"name"`
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}

// Repo reads aggregates.
type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// Ranking prior (must match internal/targets; duplicated constant checked by
// a test to prevent drift).
const (
	rankPriorMean   = 3.5
	rankPriorWeight = 10.0
)

// Stats assembles the aggregate view: counters from target_rating_stats
// (transactionally maintained) plus computed-on-read recent averages and
// per-criterion means.
func (r *Repo) Stats(ctx context.Context, targetID uuid.UUID) (TargetStats, error) {
	s := TargetStats{TargetID: targetID}

	var ratingSum, verifiedSum, returnSum int64
	var recommendYes, recommendTotal, returnCount int
	err := r.pool.QueryRow(ctx, `
		SELECT st.review_count, st.rating_sum,
			st.count_1, st.count_2, st.count_3, st.count_4, st.count_5,
			st.verified_count, st.verified_sum,
			st.recommend_yes, st.recommend_total, st.return_sum, st.return_count,
			st.updated_at
		FROM target_rating_stats st
		JOIN review_targets t ON t.id = st.target_id
		WHERE st.target_id = $1 AND t.moderation_status = 'published'`, targetID).
		Scan(&s.ReviewCount, &ratingSum,
			&s.Distribution[0], &s.Distribution[1], &s.Distribution[2], &s.Distribution[3], &s.Distribution[4],
			&s.VerifiedCount, &verifiedSum,
			&recommendYes, &recommendTotal, &returnSum, &returnCount,
			&s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return TargetStats{}, web.ErrNotFound("target")
	}
	if err != nil {
		return TargetStats{}, fmt.Errorf("loading rating stats: %w", err)
	}

	if s.ReviewCount > 0 {
		avg := round2(float64(ratingSum) / float64(s.ReviewCount))
		s.Average = &avg
	}
	if s.VerifiedCount > 0 {
		v := round2(float64(verifiedSum) / float64(s.VerifiedCount))
		s.VerifiedAverage = &v
	}
	s.RecommendSampleN = recommendTotal
	if recommendTotal >= MinSampleForPercentages {
		p := round2(100 * float64(recommendYes) / float64(recommendTotal))
		s.RecommendPercent = &p
	}
	s.ReturnIntentSample = returnCount
	if returnCount >= MinSampleForPercentages {
		ri := round2(float64(returnSum) / float64(returnCount))
		s.ReturnIntentAvg = &ri
	}
	// Ranking score: (C*m + sum) / (C + n) — Bayesian shrinkage toward the
	// global prior so tiny samples cannot dominate rankings. The raw average
	// above is never altered.
	s.RankingScore = round2((rankPriorWeight*rankPriorMean + float64(ratingSum)) / (rankPriorWeight + float64(s.ReviewCount)))
	s.Confidence = confidence(s.ReviewCount)

	// Recent average, computed on read from the partial index.
	var recentAvg *float64
	err = r.pool.QueryRow(ctx, `
		SELECT count(*)::int, avg(overall_rating)::float8
		FROM reviews
		WHERE target_id = $1 AND moderation_status = 'published'
		  AND created_at > now() - interval '90 days'`, targetID).Scan(&s.RecentCount, &recentAvg)
	if err != nil {
		return TargetStats{}, fmt.Errorf("computing recent average: %w", err)
	}
	if recentAvg != nil {
		v := round2(*recentAvg)
		s.RecentAverage = &v
	}

	// Per-criterion averages across published reviews.
	rows, err := r.pool.Query(ctx, `
		SELECT c.code, c.name, round(avg(s.score), 2)::float8, count(*)::int
		FROM review_criterion_scores s
		JOIN reviews rv ON rv.id = s.review_id
		JOIN category_criteria c ON c.id = s.criterion_id
		WHERE rv.target_id = $1 AND rv.moderation_status = 'published'
		GROUP BY c.code, c.name, c.sort_order
		ORDER BY c.sort_order`, targetID)
	if err != nil {
		return TargetStats{}, fmt.Errorf("querying criterion averages: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ca CriterionAverage
		if err := rows.Scan(&ca.Code, &ca.Name, &ca.Average, &ca.Count); err != nil {
			return TargetStats{}, fmt.Errorf("scanning criterion average: %w", err)
		}
		s.CriterionAverages = append(s.CriterionAverages, ca)
	}
	if err := rows.Err(); err != nil {
		return TargetStats{}, fmt.Errorf("iterating criterion averages: %w", err)
	}
	return s, nil
}

// RealityCheck answers "did this socially-hyped place match expectations?".
// Counts are never hidden; percentages and trends appear only above the
// documented minimum samples, and Confidence flags small samples.
type RealityCheck struct {
	TargetID uuid.UUID `json:"target_id"`

	ReviewCount   int      `json:"review_count"`
	RawAverage    *float64 `json:"raw_average,omitempty"`
	VerifiedCount int      `json:"verified_count"`
	VerifiedAvg   *float64 `json:"verified_average,omitempty"`

	RecentCount    int      `json:"recent_count"`
	RecentAverage  *float64 `json:"recent_average,omitempty"`
	HistoricalAvg  *float64 `json:"historical_average,omitempty"`
	RecentTrend    *float64 `json:"recent_trend,omitempty"` // recent − historical
	TrendAvailable bool     `json:"trend_available"`

	SocialReviewCount int                     `json:"social_review_count"`
	Expectations      ExpectationDistribution `json:"expectation_distribution"`

	Distribution [5]int `json:"rating_distribution"`
	Confidence   string `json:"confidence"`
	Note         string `json:"note,omitempty"`
}

// ExpectationDistribution reports counts (always) and percentages (when the
// sample is large enough) for the expectation-match question.
type ExpectationDistribution struct {
	Better        int `json:"better"`
	AsExpected    int `json:"as_expected"`
	Worse         int `json:"worse"`
	VeryDifferent int `json:"very_different"`

	Percentages *ExpectationPercentages `json:"percentages,omitempty"`
}

type ExpectationPercentages struct {
	Better        float64 `json:"better"`
	AsExpected    float64 `json:"as_expected"`
	Worse         float64 `json:"worse"`
	VeryDifferent float64 `json:"very_different"`
	// MatchedOrBetter = better + as_expected, the headline number.
	MatchedOrBetter float64 `json:"matched_or_better"`
}

// RealityCheckFor computes the reality check for a target.
func (r *Repo) RealityCheckFor(ctx context.Context, targetID uuid.UUID) (RealityCheck, error) {
	stats, err := r.Stats(ctx, targetID)
	if err != nil {
		return RealityCheck{}, err
	}
	rc := RealityCheck{
		TargetID:      targetID,
		ReviewCount:   stats.ReviewCount,
		RawAverage:    stats.Average,
		VerifiedCount: stats.VerifiedCount,
		VerifiedAvg:   stats.VerifiedAverage,
		RecentCount:   stats.RecentCount,
		RecentAverage: stats.RecentAverage,
		HistoricalAvg: stats.Average,
		Distribution:  stats.Distribution,
		Confidence:    stats.Confidence,
	}

	// Trend: recent window vs all-time, asserted only above minimum samples.
	if stats.RecentCount >= MinSampleForTrend && stats.ReviewCount >= MinSampleForPercentages &&
		stats.RecentAverage != nil && stats.Average != nil {
		trend := round2(*stats.RecentAverage - *stats.Average)
		rc.RecentTrend = &trend
		rc.TrendAvailable = true
	}

	// Expectation-match distribution among published social-discovery reviews.
	rows, err := r.pool.Query(ctx, `
		SELECT expectation_match, count(*)::int
		FROM reviews
		WHERE target_id = $1 AND moderation_status = 'published' AND expectation_match IS NOT NULL
		GROUP BY expectation_match`, targetID)
	if err != nil {
		return RealityCheck{}, fmt.Errorf("querying expectation distribution: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var match string
		var n int
		if err := rows.Scan(&match, &n); err != nil {
			return RealityCheck{}, fmt.Errorf("scanning expectation row: %w", err)
		}
		switch match {
		case "better":
			rc.Expectations.Better = n
		case "as_expected":
			rc.Expectations.AsExpected = n
		case "worse":
			rc.Expectations.Worse = n
		case "very_different":
			rc.Expectations.VeryDifferent = n
		}
	}
	if err := rows.Err(); err != nil {
		return RealityCheck{}, fmt.Errorf("iterating expectation rows: %w", err)
	}
	e := rc.Expectations
	rc.SocialReviewCount = e.Better + e.AsExpected + e.Worse + e.VeryDifferent

	if rc.SocialReviewCount >= MinSampleForPercentages {
		total := float64(rc.SocialReviewCount)
		pct := func(n int) float64 { return round2(100 * float64(n) / total) }
		rc.Expectations.Percentages = &ExpectationPercentages{
			Better:          pct(e.Better),
			AsExpected:      pct(e.AsExpected),
			Worse:           pct(e.Worse),
			VeryDifferent:   pct(e.VeryDifferent),
			MatchedOrBetter: pct(e.Better + e.AsExpected),
		}
	} else if rc.SocialReviewCount > 0 {
		rc.Note = "not enough social-media reviews yet for reliable percentages; counts shown"
	} else {
		rc.Note = "no social-media discovery reviews yet"
	}
	return rc, nil
}

// Handler serves the aggregate endpoints.
type Handler struct {
	repo *Repo
}

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/targets/{id}/stats", h.stats)
	mux.HandleFunc("GET /api/v1/targets/{id}/reality-check", h.realityCheck)
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	s, err := h.repo.Stats(r.Context(), id)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, s)
}

func (h *Handler) realityCheck(w http.ResponseWriter, r *http.Request) {
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	rc, err := h.repo.RealityCheckFor(r.Context(), id)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, rc)
}
