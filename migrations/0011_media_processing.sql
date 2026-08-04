-- Recoverable state for coordinating public media finalization across object
-- storage and PostgreSQL.

ALTER TABLE review_media DROP CONSTRAINT review_media_status_check;
ALTER TABLE review_media
    ADD CONSTRAINT review_media_status_check
    CHECK (status IN ('staged', 'processing', 'ready', 'removed'));
ALTER TABLE review_media
    ADD COLUMN processing_started_at timestamptz;
ALTER TABLE review_media
    ADD COLUMN staging_key text;
UPDATE review_media SET staging_key = object_key WHERE status = 'staged';

CREATE INDEX review_media_stale_processing_idx
    ON review_media (processing_started_at)
    WHERE status = 'processing';
CREATE INDEX review_media_staging_cleanup_idx
    ON review_media (review_id)
    WHERE status = 'ready' AND staging_key IS NOT NULL;
