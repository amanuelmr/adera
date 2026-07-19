package app

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewAggregatesLifecycle(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("mod")
	a.grantRoles(&mod, "moderator")
	target := a.createTarget(mod, "Aggregate Test Restaurant", catRestaurantID, "restaurant")

	// Empty state.
	s := a.stats(target)
	assert.EqualValues(t, 0, s["review_count"])
	assert.Nil(t, s["average"])
	assert.Equal(t, "none", s["confidence"])

	u1 := a.register("agg1")
	u2 := a.register("agg2")
	u3 := a.register("agg3")

	r1 := a.review(u1, target, 5, map[string]any{
		"would_recommend": true, "return_likelihood": 5,
		"criterion_scores": map[string]any{"taste": 5, "hygiene": 4},
	})
	a.review(u2, target, 3, map[string]any{
		"would_recommend": false,
		"criterion_scores": map[string]any{"taste": 3},
	})
	r3 := a.review(u3, target, 1, map[string]any{"would_recommend": false})

	s = a.stats(target)
	assert.EqualValues(t, 3, s["review_count"])
	assert.InDelta(t, 3.0, s["average"].(float64), 0.001) // (5+3+1)/3
	dist := s["distribution"].([]any)
	assert.EqualValues(t, 1, dist[0]) // one 1-star
	assert.EqualValues(t, 1, dist[2])
	assert.EqualValues(t, 1, dist[4])
	// Ranking score shrinks toward the prior: (10*3.5 + 9) / (10 + 3) = 3.38.
	assert.InDelta(t, 3.38, s["ranking_score"].(float64), 0.01)
	// Criterion averages: taste (5+3)/2 = 4.
	var tasteAvg float64
	for _, ca := range s["criterion_averages"].([]any) {
		m := ca.(map[string]any)
		if m["code"] == "taste" {
			tasteAvg = m["average"].(float64)
		}
	}
	assert.InDelta(t, 4.0, tasteAvg, 0.001)

	t.Run("edit rebalances aggregates", func(t *testing.T) {
		status, res := a.do("GET", "/api/v1/reviews/"+r1, nil, u1.Access)
		require.Equal(t, http.StatusOK, status)
		version := int(data(res)["version"].(float64))
		status, res = a.do("PUT", "/api/v1/reviews/"+r1, map[string]any{
			"overall_rating": 4, "body": "Edited body that is long enough to pass validation rules.",
			"version": version, "criterion_scores": map[string]any{"taste": 4},
		}, u1.Access)
		require.Equal(t, http.StatusOK, status, "%v", res)

		s := a.stats(target)
		assert.InDelta(t, 2.67, s["average"].(float64), 0.001) // (4+3+1)/3
		dist := s["distribution"].([]any)
		assert.EqualValues(t, 0, dist[4], "5-star bucket vacated")
		assert.EqualValues(t, 1, dist[3])
	})

	t.Run("moderation hide/restore adjusts aggregates", func(t *testing.T) {
		status, _ := a.do("POST", "/api/v1/moderation/reviews/"+r3+"/decision",
			map[string]any{"action": "hide", "note": "test"}, mod.Access)
		require.Equal(t, http.StatusOK, status)
		s := a.stats(target)
		assert.EqualValues(t, 2, s["review_count"])
		assert.InDelta(t, 3.5, s["average"].(float64), 0.001) // (4+3)/2

		// Hidden review invisible to the public and other users, visible to owner.
		status, _ = a.do("GET", "/api/v1/reviews/"+r3, nil, "")
		assert.Equal(t, http.StatusNotFound, status)
		status, _ = a.do("GET", "/api/v1/reviews/"+r3, nil, u3.Access)
		assert.Equal(t, http.StatusOK, status)

		status, _ = a.do("POST", "/api/v1/moderation/reviews/"+r3+"/decision",
			map[string]any{"action": "restore", "note": "test"}, mod.Access)
		require.Equal(t, http.StatusOK, status)
		s = a.stats(target)
		assert.EqualValues(t, 3, s["review_count"])
	})

	t.Run("own removal adjusts aggregates and is idempotent", func(t *testing.T) {
		status, _ := a.do("DELETE", "/api/v1/reviews/"+r3, nil, u3.Access)
		require.Equal(t, http.StatusOK, status)
		status, _ = a.do("DELETE", "/api/v1/reviews/"+r3, nil, u3.Access)
		require.Equal(t, http.StatusOK, status, "removal is idempotent")
		s := a.stats(target)
		assert.EqualValues(t, 2, s["review_count"])
	})

	t.Run("recommend and return-intent need minimum samples", func(t *testing.T) {
		s := a.stats(target)
		// u1's PUT edit replaced the whole review without would_recommend
		// (full-replace semantics), u3's review was removed — only u2's
		// answer remains in the sample.
		assert.EqualValues(t, 1, s["recommend_sample_size"])
		assert.Nil(t, s["recommend_percent"], "below minimum sample of 5")
	})
}

func TestReviewPolicies(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("mod")
	a.grantRoles(&mod, "moderator")
	target := a.createTarget(mod, "Policy Restaurant", catRestaurantID, "restaurant")
	u := a.register("policy")

	t.Run("criterion validation", func(t *testing.T) {
		// Unknown criterion code.
		status, res := a.do("POST", "/api/v1/reviews", map[string]any{
			"target_id": target, "overall_rating": 4,
			"body":             "A body which is definitely long enough for the check.",
			"criterion_scores": map[string]any{"nonexistent": 5},
		}, u.Access)
		assert.Equal(t, http.StatusUnprocessableEntity, status, "%v", res)

		// Score out of the criterion's scale.
		status, _ = a.do("POST", "/api/v1/reviews", map[string]any{
			"target_id": target, "overall_rating": 4,
			"body":             "A body which is definitely long enough for the check.",
			"criterion_scores": map[string]any{"taste": 9},
		}, u.Access)
		assert.Equal(t, http.StatusUnprocessableEntity, status)

		// A criterion from another category is rejected.
		status, _ = a.do("POST", "/api/v1/reviews", map[string]any{
			"target_id": target, "overall_rating": 4,
			"body":             "A body which is definitely long enough for the check.",
			"criterion_scores": map[string]any{"repair_quality": 4},
		}, u.Access)
		assert.Equal(t, http.StatusUnprocessableEntity, status)
	})

	t.Run("cooldown forces update instead of duplicate", func(t *testing.T) {
		a.review(u, target, 4, nil)
		status, res := a.do("POST", "/api/v1/reviews", map[string]any{
			"target_id": target, "overall_rating": 2,
			"body": "Another body which is definitely long enough for the check.",
		}, u.Access)
		assert.Equal(t, http.StatusConflict, status)
		assert.Equal(t, "cooldown_active", res["error"].(map[string]any)["code"])
	})

	t.Run("daily cap stops flooding", func(t *testing.T) {
		flooder := a.register("flooder")
		// 5 distinct targets reviewed today is the cap.
		for i := 0; i < 5; i++ {
			tid := a.createTarget(mod, fmt.Sprintf("Cap Target %d", i), catRestaurantID, "restaurant")
			a.review(flooder, tid, 4, nil)
		}
		extra := a.createTarget(mod, "Cap Target Extra", catRestaurantID, "restaurant")
		status, _ := a.do("POST", "/api/v1/reviews", map[string]any{
			"target_id": extra, "overall_rating": 4,
			"body": "Another body which is definitely long enough for the check.",
		}, flooder.Access)
		assert.Equal(t, http.StatusTooManyRequests, status)
	})

	t.Run("cannot review unpublished target", func(t *testing.T) {
		// A plain user's submission enters moderation as pending.
		status, res := a.do("POST", "/api/v1/targets", map[string]any{
			"target_type": "restaurant", "category_id": catRestaurantID, "name": "Pending Place",
		}, u.Access)
		require.Equal(t, http.StatusCreated, status)
		pending := data(res)["id"].(string)
		assert.Equal(t, "pending", data(res)["moderation_status"])

		status, _ = a.do("POST", "/api/v1/reviews", map[string]any{
			"target_id": pending, "overall_rating": 4,
			"body": "Another body which is definitely long enough for the check.",
		}, u.Access)
		assert.Equal(t, http.StatusNotFound, status)
	})
}

func TestConcurrentReviewSubmissions(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("mod")
	a.grantRoles(&mod, "moderator")
	target := a.createTarget(mod, "Concurrent Restaurant", catRestaurantID, "restaurant")

	const n = 8
	usersCh := make([]testUser, n)
	for i := range usersCh {
		usersCh[i] = a.register(fmt.Sprintf("conc%d", i))
	}

	var wg sync.WaitGroup
	ratings := make([]int, n)
	for i := 0; i < n; i++ {
		ratings[i] = (i % 5) + 1
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			a.review(usersCh[i], target, ratings[i], map[string]any{
				"criterion_scores": map[string]any{"taste": ratings[i]},
			})
		}(i)
	}
	wg.Wait()

	// The transactional aggregate must exactly match a ground-truth recount.
	s := a.stats(target)
	assert.EqualValues(t, n, s["review_count"])
	var sum int
	for _, r := range ratings {
		sum += r
	}
	assert.InDelta(t, float64(sum)/float64(n), s["average"].(float64), 0.01)

	var dbCount int
	var dbSum int
	err := a.pool.QueryRow(context.Background(), `
		SELECT count(*), coalesce(sum(overall_rating), 0) FROM reviews
		WHERE target_id = $1 AND moderation_status = 'published'`, target).Scan(&dbCount, &dbSum)
	require.NoError(t, err)
	var stCount, stSum int
	err = a.pool.QueryRow(context.Background(), `
		SELECT review_count, rating_sum FROM target_rating_stats WHERE target_id = $1`, target).Scan(&stCount, &stSum)
	require.NoError(t, err)
	assert.Equal(t, dbCount, stCount, "stats row must equal recount after concurrent writes")
	assert.Equal(t, dbSum, stSum)
}

func TestRealityCheckCalculation(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("mod")
	a.grantRoles(&mod, "moderator")
	target := a.createTarget(mod, "Hyped Cafe", catRestaurantID, "cafe")

	// 6 reviews, 5 with social discovery: better, as_expected ×2, worse,
	// very_different; one walk-in (excluded from expectation stats).
	matches := []struct {
		rating int
		source string
		match  string
	}{
		{5, "tiktok", "better"},
		{4, "tiktok", "as_expected"},
		{4, "instagram", "as_expected"},
		{2, "tiktok", "worse"},
		{1, "instagram", "very_different"},
		{4, "walk_in", ""},
	}
	for i, m := range matches {
		u := a.register(fmt.Sprintf("rc%d", i))
		extra := map[string]any{"discovery_source": m.source}
		if m.match != "" {
			extra["expectation_match"] = m.match
		}
		a.review(u, target, m.rating, extra)
	}

	status, res := a.do("GET", "/api/v1/targets/"+target+"/reality-check", nil, "")
	require.Equal(t, http.StatusOK, status)
	rc := data(res)

	assert.EqualValues(t, 6, rc["review_count"])
	assert.EqualValues(t, 5, rc["social_review_count"])
	assert.InDelta(t, 3.33, rc["raw_average"].(float64), 0.001) // 20/6

	ed := rc["expectation_distribution"].(map[string]any)
	assert.EqualValues(t, 1, ed["better"])
	assert.EqualValues(t, 2, ed["as_expected"])
	assert.EqualValues(t, 1, ed["worse"])
	assert.EqualValues(t, 1, ed["very_different"])

	pct := ed["percentages"].(map[string]any)
	assert.InDelta(t, 20.0, pct["better"].(float64), 0.001)
	assert.InDelta(t, 40.0, pct["as_expected"].(float64), 0.001)
	assert.InDelta(t, 60.0, pct["matched_or_better"].(float64), 0.001)

	// All reviews are recent, so trend is available and ~0.
	assert.Equal(t, true, rc["trend_available"])
	assert.InDelta(t, 0.0, rc["recent_trend"].(float64), 0.001)
	assert.Equal(t, "low", rc["confidence"])

	t.Run("small sample shows counts, not percentages", func(t *testing.T) {
		small := a.createTarget(mod, "Small Sample Cafe", catRestaurantID, "cafe")
		u := a.register("rc-small")
		a.review(u, small, 5, map[string]any{"discovery_source": "tiktok", "expectation_match": "worse"})

		status, res := a.do("GET", "/api/v1/targets/"+small+"/reality-check", nil, "")
		require.Equal(t, http.StatusOK, status)
		rc := data(res)
		ed := rc["expectation_distribution"].(map[string]any)
		assert.EqualValues(t, 1, ed["worse"], "counts are never hidden")
		assert.Nil(t, ed["percentages"], "percentages need n >= 5")
		assert.NotEmpty(t, rc["note"])
		assert.Equal(t, false, rc["trend_available"])
	})
}

func TestHelpfulVotes(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("mod")
	a.grantRoles(&mod, "moderator")
	target := a.createTarget(mod, "Votes Restaurant", catRestaurantID, "restaurant")
	author := a.register("author")
	voter := a.register("voter")
	review := a.review(author, target, 4, nil)

	// Own vote rejected.
	status, _ := a.do("PUT", "/api/v1/reviews/"+review+"/helpful", nil, author.Access)
	assert.Equal(t, http.StatusUnprocessableEntity, status)

	// Vote, idempotent revote, unvote, idempotent unvote.
	status, res := a.do("PUT", "/api/v1/reviews/"+review+"/helpful", nil, voter.Access)
	require.Equal(t, http.StatusOK, status)
	assert.EqualValues(t, 1, data(res)["helpful_count"])
	status, res = a.do("PUT", "/api/v1/reviews/"+review+"/helpful", nil, voter.Access)
	require.Equal(t, http.StatusOK, status)
	assert.EqualValues(t, 1, data(res)["helpful_count"])
	status, res = a.do("DELETE", "/api/v1/reviews/"+review+"/helpful", nil, voter.Access)
	require.Equal(t, http.StatusOK, status)
	assert.EqualValues(t, 0, data(res)["helpful_count"])
	status, res = a.do("DELETE", "/api/v1/reviews/"+review+"/helpful", nil, voter.Access)
	require.Equal(t, http.StatusOK, status)
	assert.EqualValues(t, 0, data(res)["helpful_count"])

	// Unauthenticated vote rejected.
	status, _ = a.do("PUT", "/api/v1/reviews/"+review+"/helpful", nil, "")
	assert.Equal(t, http.StatusUnauthorized, status)
}
