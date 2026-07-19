-- Business claims (ownership requests) and public business responses.

CREATE TABLE business_claims (
    id                  uuid PRIMARY KEY,
    business_id         uuid NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    user_id             uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    method              text NOT NULL CHECK (method IN ('document', 'phone', 'email', 'other')),
    evidence_object_key text NOT NULL DEFAULT '',
    message             text NOT NULL DEFAULT '' CHECK (char_length(message) <= 2000),
    status              text NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'approved', 'rejected', 'revoked')),
    decided_by          uuid REFERENCES users (id),
    decided_at          timestamptz,
    decision_note       text NOT NULL DEFAULT '' CHECK (char_length(decision_note) <= 2000),
    created_at          timestamptz NOT NULL DEFAULT now()
);

-- One pending claim per user per business at a time.
CREATE UNIQUE INDEX business_claims_pending_unique
    ON business_claims (business_id, user_id)
    WHERE status = 'pending';
CREATE INDEX business_claims_queue_idx ON business_claims (created_at) WHERE status = 'pending';
CREATE INDEX business_claims_user_idx ON business_claims (user_id, created_at DESC);

-- One public response per review. Businesses can never delete or alter the
-- customer review itself.
CREATE TABLE business_responses (
    id             uuid PRIMARY KEY,
    review_id      uuid NOT NULL UNIQUE REFERENCES reviews (id) ON DELETE CASCADE,
    business_id    uuid NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    author_user_id uuid NOT NULL REFERENCES users (id),
    body           text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER business_responses_updated_at BEFORE UPDATE ON business_responses
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Append-only audit of response edits (previous text preserved).
CREATE TABLE business_response_edits (
    id            uuid PRIMARY KEY,
    response_id   uuid NOT NULL REFERENCES business_responses (id) ON DELETE CASCADE,
    previous_body text NOT NULL,
    edited_by     uuid NOT NULL REFERENCES users (id),
    edited_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX business_response_edits_response_idx ON business_response_edits (response_id, edited_at DESC);
