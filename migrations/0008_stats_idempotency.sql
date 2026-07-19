-- Transactionally-maintained rating aggregates and idempotency keys.

-- One row per target, delta-updated in the SAME transaction as any review
-- create/edit/status change so public aggregates are always consistent.
-- Only reviews with moderation_status = 'published' are counted.
-- Recent/trend averages are computed on read from indexed review rows.
CREATE TABLE target_rating_stats (
    target_id       uuid PRIMARY KEY REFERENCES review_targets (id) ON DELETE CASCADE,
    review_count    int NOT NULL DEFAULT 0 CHECK (review_count >= 0),
    rating_sum      bigint NOT NULL DEFAULT 0 CHECK (rating_sum >= 0),
    count_1         int NOT NULL DEFAULT 0 CHECK (count_1 >= 0),
    count_2         int NOT NULL DEFAULT 0 CHECK (count_2 >= 0),
    count_3         int NOT NULL DEFAULT 0 CHECK (count_3 >= 0),
    count_4         int NOT NULL DEFAULT 0 CHECK (count_4 >= 0),
    count_5         int NOT NULL DEFAULT 0 CHECK (count_5 >= 0),
    verified_count  int NOT NULL DEFAULT 0 CHECK (verified_count >= 0),
    verified_sum    bigint NOT NULL DEFAULT 0 CHECK (verified_sum >= 0),
    recommend_yes   int NOT NULL DEFAULT 0 CHECK (recommend_yes >= 0),
    recommend_total int NOT NULL DEFAULT 0 CHECK (recommend_total >= 0),
    return_sum      bigint NOT NULL DEFAULT 0 CHECK (return_sum >= 0),
    return_count    int NOT NULL DEFAULT 0 CHECK (return_count >= 0),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CHECK (review_count = count_1 + count_2 + count_3 + count_4 + count_5),
    CHECK (recommend_yes <= recommend_total),
    CHECK (verified_count <= review_count)
);

-- Ranking scans (top-rated) read these columns; a partial covering index is
-- unnecessary at MVP scale, but a plain index keeps the sort cheap.
CREATE INDEX target_rating_stats_count_idx ON target_rating_stats (review_count DESC);

-- Idempotency for unreliable mobile connections: replayed requests with the
-- same key return the stored response instead of re-executing.
CREATE TABLE idempotency_keys (
    user_id         uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    endpoint        text NOT NULL,
    idem_key        text NOT NULL CHECK (char_length(idem_key) BETWEEN 1 AND 200),
    request_hash    bytea NOT NULL,
    response_status int,
    response_body   jsonb,
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, endpoint, idem_key)
);

-- Expired keys are purged opportunistically (see reviews service).
CREATE INDEX idempotency_keys_created_idx ON idempotency_keys (created_at);
