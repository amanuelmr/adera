-- Businesses (claimable owning entities) and review targets (the reviewable
-- things). A business may own several targets (a restaurant location, its
-- delivery service, a product line); a target may exist without a business
-- until one is created/claimed.

CREATE TABLE businesses (
    id                  uuid PRIMARY KEY,
    name                text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 160),
    slug                text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,120}$'),
    description         text NOT NULL DEFAULT '' CHECK (char_length(description) <= 2000),
    verification_status text NOT NULL DEFAULT 'unverified'
                        CHECK (verification_status IN ('unverified', 'claimed', 'verified')),
    status              text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended')),
    created_by          uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER businesses_updated_at BEFORE UPDATE ON businesses
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Membership is granted only by an approved claim (or an admin); it is the
-- object-level authorization source for business actions.
CREATE TABLE business_members (
    business_id uuid NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role        text NOT NULL DEFAULT 'owner' CHECK (role IN ('owner', 'manager')),
    granted_at  timestamptz NOT NULL DEFAULT now(),
    granted_by  uuid REFERENCES users (id),
    PRIMARY KEY (business_id, user_id)
);

CREATE INDEX business_members_user_idx ON business_members (user_id);

CREATE TABLE review_targets (
    id                  uuid PRIMARY KEY,
    target_type         text NOT NULL CHECK (target_type IN
                        ('business', 'location', 'online_seller', 'product',
                         'service', 'repair_provider', 'restaurant', 'cafe')),
    category_id         uuid NOT NULL REFERENCES categories (id),
    business_id         uuid REFERENCES businesses (id) ON DELETE SET NULL,
    name                text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 160),
    slug                text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,120}$'),
    description         text NOT NULL DEFAULT '' CHECK (char_length(description) <= 2000),
    city_id             uuid REFERENCES cities (id),
    area_id             uuid REFERENCES areas (id),
    address_text        text NOT NULL DEFAULT '' CHECK (char_length(address_text) <= 300),
    latitude            double precision CHECK (latitude BETWEEN -90 AND 90),
    longitude           double precision CHECK (longitude BETWEEN -180 AND 180),
    online_only         boolean NOT NULL DEFAULT false,
    phone               text NOT NULL DEFAULT '' CHECK (phone = '' OR phone ~ '^\+[1-9][0-9]{6,14}$'),
    website             text NOT NULL DEFAULT '' CHECK (char_length(website) <= 300),
    -- Validated allowlisted URLs only (tiktok/instagram/facebook/telegram/youtube).
    social_links        jsonb NOT NULL DEFAULT '{}'::jsonb,
    -- Type-specific fields validated by domain code before write.
    metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
    verification_status text NOT NULL DEFAULT 'unverified'
                        CHECK (verification_status IN ('unverified', 'verified')),
    moderation_status   text NOT NULL DEFAULT 'pending'
                        CHECK (moderation_status IN ('pending', 'published', 'hidden', 'removed', 'merged')),
    merged_into_id      uuid REFERENCES review_targets (id),
    created_by          uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CHECK ((latitude IS NULL) = (longitude IS NULL)),
    CHECK (moderation_status <> 'merged' OR merged_into_id IS NOT NULL)
);

CREATE TRIGGER review_targets_updated_at BEFORE UPDATE ON review_targets
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX targets_business_idx ON review_targets (business_id) WHERE business_id IS NOT NULL;
CREATE INDEX targets_browse_idx
    ON review_targets (category_id, city_id, area_id)
    WHERE moderation_status = 'published';
CREATE INDEX targets_moderation_queue_idx
    ON review_targets (created_at)
    WHERE moderation_status = 'pending';
CREATE INDEX targets_geo_idx
    ON review_targets (latitude, longitude)
    WHERE latitude IS NOT NULL AND moderation_status = 'published';

-- Alternate names: Amharic spellings, transliterations, old and informal
-- names. Central to mixed-script search.
CREATE TABLE target_aliases (
    id         uuid PRIMARY KEY,
    target_id  uuid NOT NULL REFERENCES review_targets (id) ON DELETE CASCADE,
    alias      text NOT NULL CHECK (char_length(alias) BETWEEN 2 AND 160),
    created_by uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (target_id, alias)
);

-- Community-suggested corrections, applied by moderators.
CREATE TABLE target_edit_suggestions (
    id          uuid PRIMARY KEY,
    target_id   uuid NOT NULL REFERENCES review_targets (id) ON DELETE CASCADE,
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    changes     jsonb NOT NULL,
    note        text NOT NULL DEFAULT '' CHECK (char_length(note) <= 1000),
    status      text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'applied', 'dismissed')),
    reviewed_by uuid REFERENCES users (id),
    reviewed_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX target_edit_suggestions_queue_idx
    ON target_edit_suggestions (created_at)
    WHERE status = 'open';
