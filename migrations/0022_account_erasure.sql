-- Account erasure runs by itself: a request locks the account at once and is
-- carried out after a seven-day grace period unless the person signs in
-- again. Reviews stay (they describe real commerce, and removing them would
-- let a business pressure reviewers into deleting accounts) but lose
-- everything that identifies the author; the account row remains as a
-- tombstone so they still have an author.

ALTER TABLE users DROP CONSTRAINT users_status_check;
ALTER TABLE users ADD CONSTRAINT users_status_check
    CHECK (status IN ('active', 'suspended', 'deactivated', 'erased'));
-- An erased account keeps neither email nor phone.
ALTER TABLE users DROP CONSTRAINT users_check;
ALTER TABLE users ADD CONSTRAINT users_contact_check
    CHECK (status = 'erased' OR email IS NOT NULL OR phone IS NOT NULL);
ALTER TABLE users ADD COLUMN erased_at timestamptz;

ALTER TABLE data_erasure_requests DROP CONSTRAINT data_erasure_requests_status_check;
ALTER TABLE data_erasure_requests ADD CONSTRAINT data_erasure_requests_status_check
    CHECK (status IN ('pending', 'in_review', 'approved', 'rejected', 'cancelled',
                      'scheduled', 'completed'));
ALTER TABLE data_erasure_requests ADD COLUMN scheduled_for timestamptz;
ALTER TABLE data_erasure_requests ADD COLUMN completed_at timestamptz;

DROP INDEX data_erasure_requests_active_unique;
CREATE UNIQUE INDEX data_erasure_requests_active_unique
    ON data_erasure_requests (user_id)
    WHERE status IN ('pending', 'in_review', 'approved', 'scheduled');
CREATE INDEX data_erasure_requests_due_idx
    ON data_erasure_requests (scheduled_for) WHERE status = 'scheduled';

ALTER TABLE data_erasure_request_events DROP CONSTRAINT data_erasure_request_events_action_check;
ALTER TABLE data_erasure_request_events ADD CONSTRAINT data_erasure_request_events_action_check
    CHECK (action IN ('requested', 'cancelled', 'review_started', 'approved', 'rejected',
                      'completed'));

-- Stored files to delete. Erasure queues them in its transaction, and a
-- worker deletes them afterwards, so a storage outage delays deletion rather
-- than losing track of it.
CREATE TABLE pending_object_deletions (
    bucket     text NOT NULL,
    object_key text NOT NULL,
    attempts   int NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (bucket, object_key)
);
