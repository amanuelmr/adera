-- Photos of a review that leaves public view (removed by its author, or
-- hidden, rejected or removed by a moderator) are moved out of the public
-- bucket, and moved back if the review is restored. withheld records which
-- bucket holds them: false = public bucket at object_key, true = private
-- bucket at 'withheld/' || object_key.

ALTER TABLE review_media ADD COLUMN withheld boolean NOT NULL DEFAULT false;
