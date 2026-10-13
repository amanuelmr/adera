-- A push token belongs to the sign-in it was registered under. Delivery skips
-- tokens whose sign-in has ended (logout, revocation, expiry), so a phone
-- that signed out while offline, and so couldn't unregister, stops getting
-- that account's notifications. NULL for tokens registered before this; the
-- app re-registers on launch, which fills it in.

ALTER TABLE device_tokens ADD COLUMN session_family_id uuid;
