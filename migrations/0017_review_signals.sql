-- Fraud signals for moderators (docs/moderation-policy.md, "Fraud signals").
-- Keyed HMAC-SHA256 hashes only: the network prefix (/24 IPv4, /48 IPv6) and
-- the app install ID a review was created from. Never raw addresses. Rows
-- are purged after 90 days, and go with the review if it is deleted.

CREATE TABLE review_signals (
    review_id    uuid PRIMARY KEY REFERENCES reviews (id) ON DELETE CASCADE,
    target_id    uuid NOT NULL REFERENCES review_targets (id) ON DELETE CASCADE,
    network_hash bytea CHECK (octet_length(network_hash) = 32),
    install_hash bytea CHECK (octet_length(install_hash) = 32),
    created_at   timestamptz NOT NULL DEFAULT now(),
    CHECK (network_hash IS NOT NULL OR install_hash IS NOT NULL)
);

CREATE INDEX review_signals_target_idx ON review_signals (target_id);
CREATE INDEX review_signals_created_idx ON review_signals (created_at);
