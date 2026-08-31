-- Push destinations, so the notification outbox can reach mobile clients.

CREATE TABLE device_tokens (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- Unique per device: when a second account signs in on the same handset
    -- the token moves to the new user, so the previous one stops receiving
    -- that device's pushes. Shared phones make this a privacy requirement,
    -- not an optimisation.
    token        text NOT NULL UNIQUE CHECK (char_length(token) BETWEEN 32 AND 4096),
    platform     text NOT NULL CHECK (platform IN ('android', 'ios', 'web')),
    app_version  text NOT NULL DEFAULT '' CHECK (char_length(app_version) <= 40),
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX device_tokens_user_idx ON device_tokens (user_id);
