-- Reports and the append-only moderation audit trail.

CREATE TABLE reports (
    id              uuid PRIMARY KEY,
    reporter_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    review_id       uuid REFERENCES reviews (id) ON DELETE CASCADE,
    target_id       uuid REFERENCES review_targets (id) ON DELETE CASCADE,
    reason          text NOT NULL CHECK (reason IN
                    ('spam', 'fake_experience', 'conflict_of_interest', 'harassment',
                     'hate_speech', 'personal_information', 'unsupported_accusation',
                     'irrelevant', 'duplicate', 'manipulated_evidence')),
    details         text NOT NULL DEFAULT '' CHECK (char_length(details) <= 2000),
    status          text NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open', 'in_review', 'resolved', 'dismissed')),
    resolved_by     uuid REFERENCES users (id),
    resolved_at     timestamptz,
    resolution_note text NOT NULL DEFAULT '' CHECK (char_length(resolution_note) <= 2000),
    created_at      timestamptz NOT NULL DEFAULT now(),
    -- Exactly one subject.
    CHECK (num_nonnulls(review_id, target_id) = 1)
);

-- A user may have at most one open report per subject.
CREATE UNIQUE INDEX reports_open_review_unique
    ON reports (reporter_id, review_id)
    WHERE status = 'open' AND review_id IS NOT NULL;
CREATE UNIQUE INDEX reports_open_target_unique
    ON reports (reporter_id, target_id)
    WHERE status = 'open' AND target_id IS NOT NULL;
-- Moderation queue ordered oldest-first (24h takedown SLA; see
-- docs/research/security-and-legal-risks.md §7.3).
CREATE INDEX reports_queue_idx ON reports (status, created_at);
CREATE INDEX reports_reporter_idx ON reports (reporter_id, created_at DESC);
CREATE INDEX reports_review_idx ON reports (review_id) WHERE review_id IS NOT NULL;

-- Append-only audit log of every moderation/administrative decision.
-- Normal API operations never delete from this table.
CREATE TABLE moderation_actions (
    id           uuid PRIMARY KEY,
    actor_id     uuid REFERENCES users (id),
    subject_type text NOT NULL CHECK (subject_type IN
                 ('review', 'target', 'claim', 'user', 'report', 'response', 'evidence')),
    subject_id   uuid NOT NULL,
    action       text NOT NULL CHECK (char_length(action) BETWEEN 2 AND 60),
    note         text NOT NULL DEFAULT '' CHECK (char_length(note) <= 2000),
    metadata     jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX moderation_actions_subject_idx
    ON moderation_actions (subject_type, subject_id, created_at DESC);
CREATE INDEX moderation_actions_actor_idx
    ON moderation_actions (actor_id, created_at DESC);
