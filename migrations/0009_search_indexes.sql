-- Search indexes: full-text ('simple' config — no Amharic stemmer exists, and
-- Ethiopic has no case) plus trigram fuzzy matching over am_fold-normalized
-- names and aliases. Expressions here must match internal/search queries
-- exactly for the planner to use the indexes.

CREATE INDEX targets_fts_idx ON review_targets
    USING gin (to_tsvector('simple', am_fold(lower(name)) || ' ' || am_fold(lower(description))));

CREATE INDEX targets_name_trgm_idx ON review_targets
    USING gin (am_fold(lower(name)) gin_trgm_ops);

CREATE INDEX target_aliases_trgm_idx ON target_aliases
    USING gin (am_fold(lower(alias)) gin_trgm_ops);
