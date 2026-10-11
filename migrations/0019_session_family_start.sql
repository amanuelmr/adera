-- Each refresh rotation inserts a new row with a fresh expiry, so a session
-- kept in use never ended. Every row now carries when its family (the
-- original sign-in) began, so refresh can enforce a maximum session lifetime
-- without scanning the family.

ALTER TABLE auth_sessions ADD COLUMN family_created_at timestamptz;

UPDATE auth_sessions s
SET family_created_at = f.started
FROM (SELECT family_id, min(created_at) AS started FROM auth_sessions GROUP BY family_id) f
WHERE s.family_id = f.family_id;

ALTER TABLE auth_sessions
    ALTER COLUMN family_created_at SET DEFAULT now(),
    ALTER COLUMN family_created_at SET NOT NULL;
