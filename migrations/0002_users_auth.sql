-- Users, roles, authentication sessions, and verification tokens.

CREATE TABLE users (
    id                 uuid PRIMARY KEY,
    display_name       text NOT NULL CHECK (char_length(display_name) BETWEEN 2 AND 80),
    email              citext UNIQUE,
    -- E.164; Ethiopian numbers normalize to +2519xxxxxxxx / +2517xxxxxxxx.
    phone              text UNIQUE CHECK (phone ~ '^\+[1-9][0-9]{6,14}$'),
    password_hash      text,
    status             text NOT NULL DEFAULT 'active'
                       CHECK (status IN ('active', 'suspended', 'deactivated')),
    email_verified_at  timestamptz,
    phone_verified_at  timestamptz,
    preferred_language text NOT NULL DEFAULT 'en'
                       CHECK (preferred_language IN ('am', 'en', 'om', 'ti', 'so')),
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CHECK (email IS NOT NULL OR phone IS NOT NULL)
);

CREATE TRIGGER users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE user_roles (
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role       text NOT NULL CHECK (role IN ('customer', 'business_owner', 'moderator', 'admin')),
    granted_at timestamptz NOT NULL DEFAULT now(),
    granted_by uuid REFERENCES users (id),
    PRIMARY KEY (user_id, role)
);

-- One row per refresh token. Rotation inserts a new row in the same family
-- and links the old row via replaced_by_id; presenting a rotated or revoked
-- token is reuse and revokes the whole family.
CREATE TABLE auth_sessions (
    id                 uuid PRIMARY KEY,
    user_id            uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    family_id          uuid NOT NULL,
    refresh_token_hash bytea NOT NULL UNIQUE,
    device_info        text NOT NULL DEFAULT '' CHECK (char_length(device_info) <= 300),
    created_at         timestamptz NOT NULL DEFAULT now(),
    expires_at         timestamptz NOT NULL,
    last_used_at       timestamptz NOT NULL DEFAULT now(),
    revoked_at         timestamptz,
    revoked_reason     text,
    replaced_by_id     uuid REFERENCES auth_sessions (id)
);

CREATE INDEX auth_sessions_user_idx ON auth_sessions (user_id, created_at DESC);
CREATE INDEX auth_sessions_family_idx ON auth_sessions (family_id);

-- Hashed one-time codes for email/phone verification and password reset.
CREATE TABLE verification_tokens (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose     text NOT NULL CHECK (purpose IN ('email_verify', 'phone_verify', 'password_reset')),
    code_hash   bytea NOT NULL,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz,
    attempts    int NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX verification_tokens_active_idx
    ON verification_tokens (user_id, purpose)
    WHERE consumed_at IS NULL;
