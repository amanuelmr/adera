-- Data-subject erasure request workflow. Approval records acceptance for a
-- separately reviewed fulfillment process; it does not delete retained data.

CREATE TABLE data_erasure_requests (
    id            uuid PRIMARY KEY,
    user_id       uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    status        text NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending', 'in_review', 'approved', 'rejected', 'cancelled')),
    reason        text NOT NULL DEFAULT '' CHECK (char_length(reason) <= 1000),
    decision_note text NOT NULL DEFAULT '' CHECK (char_length(decision_note) <= 2000),
    reviewed_by   uuid REFERENCES users (id),
    reviewed_at   timestamptz,
    decided_at    timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER data_erasure_requests_updated_at BEFORE UPDATE ON data_erasure_requests
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE UNIQUE INDEX data_erasure_requests_active_unique
    ON data_erasure_requests (user_id)
    WHERE status IN ('pending', 'in_review', 'approved');
CREATE INDEX data_erasure_requests_queue_idx
    ON data_erasure_requests (status, created_at, id);
CREATE INDEX data_erasure_requests_user_idx
    ON data_erasure_requests (user_id, created_at DESC, id DESC);

CREATE TABLE data_erasure_request_events (
    id         uuid PRIMARY KEY,
    request_id uuid NOT NULL REFERENCES data_erasure_requests (id) ON DELETE CASCADE,
    actor_id   uuid REFERENCES users (id),
    action     text NOT NULL CHECK (action IN
               ('requested', 'cancelled', 'review_started', 'approved', 'rejected')),
    note       text NOT NULL DEFAULT '' CHECK (char_length(note) <= 2000),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX data_erasure_request_events_request_idx
    ON data_erasure_request_events (request_id, created_at, id);
