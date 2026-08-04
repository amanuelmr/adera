-- User activity inbox and durable outbox for external delivery adapters.

CREATE TABLE notifications (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    event_type   text NOT NULL CHECK (char_length(event_type) BETWEEN 2 AND 80),
    subject_type text NOT NULL CHECK (subject_type IN
                 ('account', 'claim', 'report', 'review', 'target', 'evidence', 'response')),
    subject_id   uuid NOT NULL,
    data         jsonb NOT NULL DEFAULT '{}'::jsonb,
    dedupe_key   text NOT NULL CHECK (char_length(dedupe_key) BETWEEN 2 AND 200),
    read_at      timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, dedupe_key)
);

CREATE INDEX notifications_user_created_idx
    ON notifications (user_id, created_at DESC, id DESC);
CREATE INDEX notifications_user_unread_idx
    ON notifications (user_id, created_at DESC)
    WHERE read_at IS NULL;

CREATE TABLE notification_outbox (
    id              uuid PRIMARY KEY,
    notification_id uuid NOT NULL UNIQUE REFERENCES notifications (id) ON DELETE CASCADE,
    topic           text NOT NULL DEFAULT 'notification.created',
    payload         jsonb NOT NULL,
    attempts        int NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    available_at    timestamptz NOT NULL DEFAULT now(),
    processed_at    timestamptz,
    last_error      text NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX notification_outbox_pending_idx
    ON notification_outbox (available_at, created_at)
    WHERE processed_at IS NULL;
