-- Password-reset requests are capped per account per day, which counts codes
-- whether or not they were used; the existing index covers only unused ones.

CREATE INDEX verification_tokens_history_idx
    ON verification_tokens (user_id, purpose, created_at);
