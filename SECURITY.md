# Security Policy

## Reporting a vulnerability

Email the maintainers (see repository owner profile) with details and
reproduction steps. Please do not open public issues for security reports and
do not access other users' data while demonstrating an issue. We aim to
acknowledge within 72 hours.

## Scope highlights

Especially interested in: authentication/session flaws (token rotation,
family reuse detection), object-level authorization bypasses (accessing other
users' reviews/evidence/sessions/businesses), private-evidence exposure,
upload pipeline bypasses (polyglots, metadata survival), SQL injection, and
rate-limit bypasses enabling review manipulation.

## Design summary (for researchers)

- Argon2id password hashing (OWASP parameters, PHC format, constant-time
  compare); generic auth errors; login/OTP rate limits.
- 15-minute HS256 access tokens; opaque refresh tokens stored as SHA-256
  hashes with token families — reuse of a rotated token revokes the family.
- Strict JSON decoding (unknown fields rejected), explicit update allowlists,
  request body caps, per-request timeouts.
- All uploads via presigned POST (server-chosen keys, exact content type,
  size range) into a private bucket; public images re-encoded (EXIF/GPS
  stripped) before promotion; evidence never publicly readable.
- Append-only moderation audit log; secrets from environment only.

Full threat mapping: docs/research/security-and-legal-risks.md (OWASP API
Top 10 2023 table).
