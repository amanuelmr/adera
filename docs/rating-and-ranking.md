# Rating Aggregation & Ranking

This document is the authoritative reference for every number Adera shows
about a target. The displayed **raw average is never altered**; ranking uses a
separate, documented score.

## 1. Which reviews count

Only reviews with `moderation_status = 'published'` affect any public
aggregate. `pending`, `under_review`, `hidden`, `rejected`, and `removed`
reviews are excluded the moment their status changes — the status change and
the aggregate adjustment happen in **the same database transaction**
(`internal/reviews`, `statsDelta`). Edited reviews are handled as
"remove old contribution, add new contribution" inside the edit transaction.

Aggregates live in `target_rating_stats`, one row per target, delta-maintained
and CHECK-constrained (`review_count = count_1+…+count_5`). Merging duplicate
targets recomputes both rows from ground truth inside the merge transaction.
Integration tests assert exact aggregate values after create, edit, hide,
restore, removal, verification changes, merges, and concurrent submissions.

## 2. Displayed values

| Value | Formula | Rounding |
|---|---|---|
| Raw average | `rating_sum / review_count` | half-away-from-zero, 2 decimals |
| Distribution | `count_1 … count_5` | exact counts, never hidden |
| Verified average | same, over reviews with `verification_level ∈ {receipt_submitted, location_verified, partner_verified}` | 2 decimals |
| Recent average | mean of published reviews in the last **90 days** (computed on read) | 2 decimals |
| Historical average | the all-time raw average | 2 decimals |
| Recommendation % | `100 × recommend_yes / recommend_total` | 2 decimals |
| Return intent | mean of `return_likelihood` (1–5) | 2 decimals |
| Per-criterion averages | mean of criterion scores over published reviews | 2 decimals |

`media_attached` deliberately does **not** count as "verified": a photo adds
context but does not evidence a transaction (see docs/verification-model.md).

## 3. Minimum-sample rules

Counts are always returned. Derived percentages are withheld below minimum
samples so small numbers can't masquerade as strong conclusions:

- Recommendation % and return intent: need **n ≥ 5** answers.
- Expectation-match percentages (Reality Check): need **n ≥ 5** social-discovery reviews; below that, counts plus an explanatory `note`.
- Recent trend: needs **≥ 3** recent reviews **and ≥ 5** total.
- Confidence label on every stats payload: `none` (<3), `low` (<10), `medium` (<30), `high` (≥30) published reviews.

## 4. Ranking score (never shown as "the rating")

Sorting "top rated" by raw average lets one 5-star review beat a hundred
4.8-star reviews. Ranking therefore uses Bayesian shrinkage toward a global
prior:

```
ranking_score = (C·m + Σ ratings) / (C + n)
    m = 3.5   (prior mean)
    C = 10    (prior weight, "ten phantom average reviews")
    n = published review count
```

Example: one 5★ review → (35+5)/11 = **3.64**; a hundred reviews averaging
4.6 → (35+460)/110 = **4.50**. The ranking score appears in API responses as
`ranking_score`, clearly distinct from `average`, and this formula is public.
The constants live in `internal/targets` (`RankPriorMean`, `RankPriorWeight`)
and are asserted against `internal/ratings` by tests.

## 5. Trending

`GET /targets/trending` = published-review **volume in the last 30 days**,
ties broken by recent average. Simple and legible by design — no opaque
algorithmic feed.

## 6. Review-list "most relevant" sort

`relevant = helpful_votes + 2·(is verified) − age_in_days/30`, descending.
Single page (no cursor); the default listing sort is `newest`, and the default
is labeled so users know what they're seeing.

## 7. Reality Check (restaurants & cafés)

`GET /targets/{id}/reality-check` answers "did this socially-hyped place match
expectations?" without ever accusing a specific creator:

- **Inputs:** reviews whose `discovery_source ∈ {tiktok, instagram, youtube,
  facebook, telegram}` and that answered `expectation_match`
  (`better | as_expected | worse | very_different`). The question is only
  accepted on social-discovery reviews (enforced at validation and by a DB
  CHECK).
- **Output:** raw counts (always) per answer; percentages plus
  `matched_or_better` when n ≥ 5; raw average vs verified average; recent
  (90-day) vs historical average and their difference (`recent_trend`) when
  sample rules pass; the full rating distribution; a confidence label.
- Language stays aggregate and neutral: "68% said the experience matched
  expectations", never "influencer X lied".

## 8. Treatment summary

| Event | Aggregate effect |
|---|---|
| Review created (published) | +1 contribution, same transaction |
| Review edited | −old +new, same transaction |
| Hidden / rejected / removed / under review | −1 contribution, same transaction |
| Restored / approved | +1 contribution, same transaction |
| Verification upgrade into verified set | verified_count/sum adjusted, same transaction |
| Owner-deleted (soft "removed") | −1; the row is retained for audit |
| Target merged | both stats rows recomputed from ground truth in the merge transaction |
