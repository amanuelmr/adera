-- Categories with category-specific review criteria, cities, and areas.
-- Criteria live in the database (not code) so admins can evolve them and so
-- labels are translatable; translations are jsonb maps keyed by language code.

CREATE TABLE categories (
    id                       uuid PRIMARY KEY,
    code                     text NOT NULL UNIQUE CHECK (code ~ '^[a-z0-9_]{2,40}$'),
    name                     text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 80),
    name_translations        jsonb NOT NULL DEFAULT '{}'::jsonb,
    description              text NOT NULL DEFAULT '' CHECK (char_length(description) <= 500),
    description_translations jsonb NOT NULL DEFAULT '{}'::jsonb,
    sort_order               int NOT NULL DEFAULT 0,
    active                   boolean NOT NULL DEFAULT true,
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER categories_updated_at BEFORE UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE category_criteria (
    id                 uuid PRIMARY KEY,
    category_id        uuid NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    code               text NOT NULL CHECK (code ~ '^[a-z0-9_]{2,40}$'),
    name               text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 80),
    label_translations jsonb NOT NULL DEFAULT '{}'::jsonb,
    description        text NOT NULL DEFAULT '' CHECK (char_length(description) <= 300),
    required           boolean NOT NULL DEFAULT false,
    scale_min          int NOT NULL DEFAULT 1 CHECK (scale_min >= 0),
    scale_max          int NOT NULL DEFAULT 5,
    sort_order         int NOT NULL DEFAULT 0,
    active             boolean NOT NULL DEFAULT true,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    UNIQUE (category_id, code),
    CHECK (scale_max > scale_min AND scale_max <= 10)
);

CREATE TRIGGER category_criteria_updated_at BEFORE UPDATE ON category_criteria
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX category_criteria_active_idx
    ON category_criteria (category_id, sort_order)
    WHERE active;

CREATE TABLE cities (
    id         uuid PRIMARY KEY,
    name       text NOT NULL UNIQUE CHECK (char_length(name) BETWEEN 2 AND 80),
    name_am    text NOT NULL DEFAULT '',
    country    char(2) NOT NULL DEFAULT 'ET',
    sort_order int NOT NULL DEFAULT 0,
    active     boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE areas (
    id         uuid PRIMARY KEY,
    city_id    uuid NOT NULL REFERENCES cities (id) ON DELETE CASCADE,
    name       text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 80),
    name_am    text NOT NULL DEFAULT '',
    sort_order int NOT NULL DEFAULT 0,
    active     boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (city_id, name)
);

CREATE INDEX areas_city_idx ON areas (city_id, sort_order) WHERE active;
