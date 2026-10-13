-- Which privacy notice each account accepted, and when. Accounts created
-- before the notice existed have neither; the app asks them to accept it
-- on next sign-in.

ALTER TABLE users ADD COLUMN privacy_notice_version text;
ALTER TABLE users ADD COLUMN privacy_accepted_at timestamptz;
