-- Reviews, criterion scores, public media, private verification evidence,
-- and helpful votes.

CREATE TABLE reviews (
    id                 uuid PRIMARY KEY,
    target_id          uuid NOT NULL REFERENCES review_targets (id) ON DELETE CASCADE,
    user_id            uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    overall_rating     int NOT NULL CHECK (overall_rating BETWEEN 1 AND 5),
    title              text NOT NULL DEFAULT '' CHECK (char_length(title) <= 120),
    body               text NOT NULL CHECK (char_length(body) BETWEEN 20 AND 5000),
    -- BCP-47-ish language code of the body when known; '' = undeclared.
    language           text NOT NULL DEFAULT '' CHECK (language IN ('', 'am', 'en', 'om', 'ti', 'so')),
    experience_date    date CHECK (experience_date >= DATE '2000-01-01'),
    price_paid         numeric(12, 2) CHECK (price_paid > 0),
    currency           char(3) NOT NULL DEFAULT 'ETB' CHECK (currency ~ '^[A-Z]{3}$'),
    would_recommend    boolean,
    return_likelihood  int CHECK (return_likelihood BETWEEN 1 AND 5),
    discovery_source   text CHECK (discovery_source IN
                       ('tiktok', 'instagram', 'youtube', 'facebook', 'telegram',
                        'friend', 'google_maps', 'walk_in', 'other')),
    -- Reality Check answer; only meaningful (and only allowed) when the
    -- discovery source is a social platform.
    expectation_match  text CHECK (expectation_match IN
                       ('better', 'as_expected', 'worse', 'very_different')),
    social_media_url   text NOT NULL DEFAULT '' CHECK (char_length(social_media_url) <= 300),
    verification_level text NOT NULL DEFAULT 'unverified'
                       CHECK (verification_level IN
                       ('unverified', 'media_attached', 'receipt_submitted',
                        'location_verified', 'partner_verified')),
    moderation_status  text NOT NULL DEFAULT 'published'
                       CHECK (moderation_status IN
                       ('pending', 'published', 'under_review', 'rejected', 'hidden', 'removed')),
    edit_count         int NOT NULL DEFAULT 0,
    edited_at          timestamptz,
    -- Optimistic concurrency for edits.
    version            int NOT NULL DEFAULT 1,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CHECK (expectation_match IS NULL OR discovery_source IN
           ('tiktok', 'instagram', 'youtube', 'facebook', 'telegram'))
);

CREATE TRIGGER reviews_updated_at BEFORE UPDATE ON reviews
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX reviews_target_published_idx
    ON reviews (target_id, created_at DESC)
    WHERE moderation_status = 'published';
CREATE INDEX reviews_target_rating_idx
    ON reviews (target_id, overall_rating, created_at DESC)
    WHERE moderation_status = 'published';
CREATE INDEX reviews_user_idx ON reviews (user_id, created_at DESC);
CREATE INDEX reviews_moderation_queue_idx
    ON reviews (created_at)
    WHERE moderation_status IN ('pending', 'under_review');
-- Reality Check aggregation over social-media discoveries.
CREATE INDEX reviews_expectation_idx
    ON reviews (target_id, expectation_match)
    WHERE moderation_status = 'published' AND expectation_match IS NOT NULL;

-- One row per (review, criterion). Score range is validated by domain code
-- against the criterion's own scale; the CHECK is a backstop.
CREATE TABLE review_criterion_scores (
    review_id    uuid NOT NULL REFERENCES reviews (id) ON DELETE CASCADE,
    criterion_id uuid NOT NULL REFERENCES category_criteria (id),
    score        int NOT NULL CHECK (score BETWEEN 0 AND 10),
    PRIMARY KEY (review_id, criterion_id)
);

CREATE INDEX review_criterion_scores_criterion_idx ON review_criterion_scores (criterion_id);

-- Public review photos: stored in the PUBLIC bucket, re-encoded server-side
-- (EXIF/GPS stripped) before becoming 'ready'.
CREATE TABLE review_media (
    id           uuid PRIMARY KEY,
    review_id    uuid NOT NULL REFERENCES reviews (id) ON DELETE CASCADE,
    object_key   text NOT NULL UNIQUE,
    content_type text NOT NULL,
    size_bytes   bigint NOT NULL DEFAULT 0 CHECK (size_bytes >= 0),
    status       text NOT NULL DEFAULT 'staged' CHECK (status IN ('staged', 'ready', 'removed')),
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX review_media_review_idx ON review_media (review_id) WHERE status = 'ready';

-- Private verification evidence: PRIVATE bucket, never exposed publicly,
-- accessible only to the owner and moderators via short-lived presigned GETs.
CREATE TABLE review_evidence (
    id           uuid PRIMARY KEY,
    review_id    uuid NOT NULL REFERENCES reviews (id) ON DELETE CASCADE,
    kind         text NOT NULL CHECK (kind IN
                 ('receipt', 'order_screenshot', 'product_photo', 'service_result', 'location_qr')),
    object_key   text NOT NULL UNIQUE,
    content_type text NOT NULL,
    size_bytes   bigint NOT NULL DEFAULT 0 CHECK (size_bytes >= 0),
    status       text NOT NULL DEFAULT 'staged'
                 CHECK (status IN ('staged', 'submitted', 'accepted', 'rejected')),
    reviewed_by  uuid REFERENCES users (id),
    reviewed_at  timestamptz,
    note         text NOT NULL DEFAULT '' CHECK (char_length(note) <= 1000),
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX review_evidence_review_idx ON review_evidence (review_id);
CREATE INDEX review_evidence_queue_idx ON review_evidence (created_at) WHERE status = 'submitted';

CREATE TABLE helpful_votes (
    review_id  uuid NOT NULL REFERENCES reviews (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (review_id, user_id)
);
