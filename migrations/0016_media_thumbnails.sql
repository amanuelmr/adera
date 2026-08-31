-- Thumbnail derivative for public review photos. Nullable: an upload already
-- within the thumbnail bound has no derivative, and readers fall back to the
-- full image.

ALTER TABLE review_media ADD COLUMN thumb_key text;
