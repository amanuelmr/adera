-- Extensions and shared functions.

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS citext;

-- am_fold normalizes Amharic (Ethiopic) text for search:
--   * folds homophone letter series that are used interchangeably in modern
--     spelling (ሐ/ኀ → ሀ series, ሠ → ሰ, ዐ → አ, ፀ → ጸ), preserving vowel order;
--   * maps Ethiopic wordspace and punctuation (፡ ። ፣ ፤ ፥) to ASCII spaces so
--     tokenizers and trigram matching behave consistently.
-- Latin text passes through unchanged (callers lower() separately).
-- Heuristic folding; see docs/research/tech-stack-decisions.md §4.
CREATE OR REPLACE FUNCTION am_fold(text) RETURNS text
    IMMUTABLE PARALLEL SAFE STRICT LANGUAGE sql AS
$$
SELECT translate($1,
    'ሐሑሒሓሔሕሖኀኁኂኃኄኅኆሠሡሢሣሤሥሦዐዑዒዓዔዕዖፀፁፂፃፄፅፆ፡።፣፤፥',
    'ሀሁሂሃሄህሆሀሁሂሃሄህሆሰሱሲሳሴስሶአኡኢኣኤእኦጸጹጺጻጼጽጾ     ')
$$;

-- set_updated_at keeps updated_at current on any UPDATE.
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger
    LANGUAGE plpgsql AS
$$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;
