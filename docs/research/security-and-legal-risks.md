# Security and Legal Risk Register — Adera Review Platform (Go + PostgreSQL)

**Research date:** 2026-07-17. All sources accessed on this date unless noted.
**Scope disclaimer:** The legal sections document risks for discussion with qualified Ethiopian counsel. Nothing here is legal advice or a legal conclusion.

---

## 1. Sources

| # | Source | URL | Accessed | Observation | Adopt / Avoid |
|---|--------|-----|----------|-------------|---------------|
| 1 | OWASP API Security Top 10 (2023) | https://owasp.org/API-Security/editions/2023/en/0x11-t10/ | 2026-07-17 | 2023 edition is still the current edition; 10 risks API1–API10 verified from the page. | **Adopt** as the primary threat checklist for API design review. |
| 2 | OWASP Password Storage Cheat Sheet | https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html | 2026-07-17 | Argon2id first choice; five equivalent parameter sets listed (see §4); bcrypt minimum work factor 10 for legacy only; scrypt only if Argon2id unavailable. | **Adopt** Argon2id per §4. **Avoid** bcrypt/scrypt for a greenfield Go service. |
| 3 | RFC 9106 (Argon2) | https://www.rfc-editor.org/rfc/rfc9106.html | 2026-07-17 | §4: first recommended option t=1, p=4, m=2^21 (2 GiB), 128-bit salt, 256-bit tag; second option t=3, p=4, m=2^16 (64 MiB) for memory-constrained environments. | **Adopt** as normative reference; use OWASP numbers for the actual server config (RFC's 2 GiB default is impractical per-login on shared API hosts). |
| 4 | Auth0 — Refresh Token Rotation | https://auth0.com/docs/secure/tokens/refresh-tokens/refresh-token-rotation | 2026-07-17 | Documents token families and automatic reuse detection: reuse of a rotated token invalidates the entire family. | **Adopt** the token-family model in our own DB schema. |
| 5 | OWASP session/JWT cheat sheets; RFC 9700 (OAuth 2.0 Security BCP) | https://cheatsheetseries.owasp.org/ | 2026-07-17 | Refresh tokens must be sender-constrained **or** rotated with reuse detection. RFC 9700 exact section wording was not re-fetched — confirm before citing in policy docs. | **Adopt** rotation + reuse detection (sender-constraining is impractical for this MVP). |
| 6 | AWS S3 — Uploading objects with presigned URLs | https://docs.aws.amazon.com/AmazonS3/latest/userguide/PresignedUrlUploadObject.html | 2026-07-17 | Presigned PUT does **not** reliably enforce Content-Length server-side; POST policies support `content-length-range` and exact `Content-Type` conditions. | **Adopt** presigned POST with policy conditions (works on MinIO too). **Avoid** trusting presigned PUT alone for size limits. |
| 7 | PostgreSQL docs — pg_trgm (current, PG 18) | https://www.postgresql.org/docs/current/pgtrgm.html | 2026-07-17 | Trigrams ignore non-word chars; words padded with spaces; case-insensitive by default; no language dictionaries involved. | **Adopt** for fuzzy Amharic/English matching. |
| 8 | PostgreSQL versioning policy | https://www.postgresql.org/support/versioning/ | 2026-07-17 | Supported majors: 18 (18.4), 17 (17.10), 16 (16.14), 15 (15.18), 14 (14.23, EOL Nov 2026). | **Adopt** PG 18. **Avoid** PG 14 (EOL within 4 months). |
| 9 | Ethiopia — Personal Data Protection Proclamation No. 1321/2024 | https://justice.gov.et/en/law/personal-data-protection-proclamation/ ; https://tbestlaw.com/ethiopias-personal-data-protection-proclamation-enters-in-to-force/ | 2026-07-17 | Passed 2024-04-04, published in Federal Negarit Gazette 2024-07-24; in force. Supervisor: Ethiopian Communications Authority (ECA). GDPR-like: registration, DPO where required, DPIAs, 72-hour breach notification, extraterritorial scope. | **Adopt** as the anchor compliance regime. Needs counsel review (implementing directives may have issued since). |
| 10 | Ethiopia — Computer Crime Proclamation No. 958/2016 (ARTICLE 19 analysis; EFF) | https://www.article19.org/resources/ethiopia-computer-crime-proclamation/ | 2026-07-17 | Art. 13 criminalizes online defamation; analyses note service providers can be criminally liable for third-party illegal content they fail to remove/disable. | Treat as a **platform-liability risk**; do not rely on US-style intermediary immunity. |
| 11 | Ethiopia — Hate Speech and Disinformation Prevention Proclamation No. 1185/2020 | https://justice.gov.et/en/law/hate-speech-and-disinformation-prevention-and-suppression-proclamation/ ; https://www.accessnow.org/ethiopias-hate-speech-and-disinformation-law-the-pros-the-cons-and-a-mystery/ | 2026-07-17 | Criminal penalties for hate speech (up to 3 yrs / 100,000 ETB) and disinformation (up to 1 yr / 50,000 ETB); Art. 8(2): social media service providers must remove notified hate speech/disinformation **within 24 hours** or face civil liability. | **Adopt** a 24-hour notice-and-takedown SLA in moderation ops. Needs counsel review on whether a review platform is a covered "social media" service. |
| 12 | Ethiopia — Trade Competition and Consumer Protection Proclamation No. 813/2013 | https://www.wipo.int/wipolex/en/legislation/details/21396 | 2026-07-17 | Still the governing consumer-protection law; no successor proclamation found as of 2026-07-17. Prohibits false/misleading advertising; consumer right to accurate information. | Verify with counsel that 813/2013 is still current and how it applies to hosted reviews. |
| 13 | Alex Edwards — Rate limiting HTTP requests in Go | https://www.alexedwards.net/blog/how-to-rate-limit-http-requests | 2026-07-17 | Canonical per-IP `x/time/rate` pattern: map of limiters + mutex + stale-entry eviction goroutine. | **Adopt** pattern. |
| 14 | EXIF/GPS privacy coverage | https://exifdata.org/blog/photo-gps-data-privacy-guide-to-exif-location-removal | 2026-07-17 | GPS EXIF in uploaded photos can reveal home/work addresses; major platforms strip EXIF server-side on upload. | **Adopt** server-side EXIF stripping for all public user images. |
| 15 | go.dev/dl | https://go.dev/dl/ | 2026-07-17 | Current stable go1.26.5 (previous line go1.25.12; go1.27rc2 out). | **Adopt** go1.26.x — patch releases carry security fixes. |

---

## 2. OWASP API Security Top 10 (2023) → Adera Mapping

| Risk | Mitigation in this codebase |
|------|-----------------------------|
| **API1: Broken Object Level Authorization** | Every handler that loads a review, business, evidence object, session, or user by ID checks ownership/role against the authenticated principal. Authorization tests exercise "other user's ID" cases (see `*_test.go` authz tests). |
| **API2: Broken Authentication** | Argon2id passwords (§4), 15-minute access tokens, rotated refresh tokens with family reuse detection (§3), rate-limited login with generic errors. |
| **API3: Broken Object Property Level Authorization** | Explicit request/response DTO structs with `DisallowUnknownFields`; users can never set `verification_level`, `moderation_status`, roles, or ownership fields; internal fields never serialized. |
| **API4: Unrestricted Resource Consumption** | Per-IP and per-account rate limits, `http.MaxBytesReader` (64 KB default), `content-length-range` on upload policies, pagination caps (max 50), DB query timeouts, bounded Argon2 concurrency via login rate limits. |
| **API5: Broken Function Level Authorization** | Moderator/admin endpoints behind `RequireRole` middleware verified server-side from DB-backed role claims. |
| **API6: Unrestricted Access to Sensitive Business Flows** | Review posting throttled per account/day and per target cooldown; idempotency keys; report/vote endpoints rate limited. |
| **API7: SSRF** | The API never fetches user-supplied URLs. Social links are validated against a domain allowlist and stored as strings only. |
| **API8: Security Misconfiguration** | Distroless non-root container, private-by-default MinIO buckets, CORS allowlist, security headers middleware, no debug endpoints in prod, config validation at startup. |
| **API9: Improper Inventory Management** | Single versioned API (`/api/v1`), OpenAPI spec is the endpoint inventory. |
| **API10: Unsafe Consumption of APIs** | Storage/SMS provider responses are treated as untrusted: validated, bounded timeouts, never interpolated into SQL or logs unsanitized. |

---

## 3. Auth Token Design

- **Access token:** 15-minute HS256 JWT containing only user ID, session ID, and roles; verified statelessly.
- **Refresh token:** opaque 256-bit random value; only its SHA-256 hash is stored, with `family_id`, `user_id`, `expires_at`, `replaced_by_id`, `revoked_at`, and device metadata.
- **Rotation:** every refresh issues a new token in the same family and links the old row via `replaced_by_id`.
- **Reuse detection:** presenting an already-rotated/revoked token revokes the **entire family** and forces re-authentication.
- Absolute family lifetime 30 days; logout revokes the session; logout-all revokes all the user's sessions.
- Tokens are never logged and never placed in URLs.

## 4. Password Hashing Parameters

- **OWASP Password Storage Cheat Sheet (current):** Argon2id equivalent options include `m=19456 (19 MiB), t=2, p=1` — adopted as the default here, configurable upward via `ARGON_*` env vars (startup validation rejects weaker settings).
- **RFC 9106 §4** is the normative reference (its 2 GiB first option is impractical per-login on shared API hosts).
- Implementation: `golang.org/x/crypto/argon2.IDKey`, 16-byte random salt, 32-byte key, PHC-format encoding, constant-time comparison.

## 5. Upload Security (media and evidence to S3/MinIO)

1. Backend authorizes the action first (user may only attach media to *their own* review), then issues a **presigned POST** with policy conditions: server-generated key (never client-chosen), exact `Content-Type` from an allowlist (`image/jpeg`, `image/png`, `image/webp`), and `content-length-range` (1 KB–10 MB).
2. Short expiry (5 min); an upload row in `staged` status must be finalized by the client, at which point the server validates the object.
3. Finalize step: sniff magic bytes server-side (never trust Content-Type); for **public media**, decode and re-encode the image, which drops all EXIF/GPS metadata before the object is ever served.
4. **Private verification evidence** (receipts, order screenshots) stays in the private bucket, is never re-encoded (preserves verification integrity), and is only accessible via short-lived presigned GETs issued to moderators or the owner.
5. Both buckets are private; public media is served through explicit public-read policy on the public bucket only after finalization.

## 6. Logging Redaction

- `log/slog` JSON logs with request IDs. Never logged: passwords, token values (IDs only), OTP codes, full emails/phones, auth request bodies, presigned URLs, payment references.
- Logs contain user identifiers, which are **personal data under Proclamation 1321/2024** — apply retention limits and access control to log storage; treat IP addresses as personal data pending counsel review.

## 7. Ethiopian Legal Risk Register — **ALL ITEMS NEED PROFESSIONAL LEGAL REVIEW**

### 7.1 Data protection — Proclamation No. 1321/2024 (in force)
To scope with counsel: ECA registration and whether a DPO is required; lawful basis/consent for reviewer accounts; data-subject rights endpoints (access, correction, deletion — account deactivation/deletion is implemented, retention policy documented); DPIA for review/moderation processing; **72-hour breach notification**; cross-border transfer restrictions (matters if hosting outside Ethiopia); retention schedules; processor contracts (cloud, SMS gateway).

### 7.2 Defamation — Computer Crime Proclamation No. 958/2016
Reviews are by nature negative statements about businesses. Art. 13 criminalizes digital defamation, and analyses note criminal liability for providers who fail to remove illegal third-party content after notice. Design consequence (implemented): notice-and-takedown workflow via reports, immutable audit trail of moderation decisions, "unsupported serious accusation" report reason, hide-without-delete moderation states. Do not assume any intermediary safe harbor exists.

### 7.3 Hate speech / disinformation — Proclamation No. 1185/2020
Art. 8(2): providers must act **within 24 hours** of notification. Counsel questions: does a review platform fall within the proclamation's "social media" definition; what counts as valid notification; whether fake reviews are "disinformation". Design consequence (implemented): hate-speech report reason, moderation queue ordered by age, audit trail.

### 7.4 Consumer protection — Proclamation No. 813/2013
Undisclosed paid/incentivized reviews could constitute misleading advertising. Counsel questions: platform exposure for hosted reviews; disclosure requirements for any future promoted placements. Design consequence: conflict-of-interest report reason; no paid placement in MVP.

### 7.5 Cross-cutting content-moderation obligations
Combined effect of 958/2016 + 1185/2020: active takedown duties with criminal/civil backstops. Budget for human moderation with Amharic competence, notice intake + SLA tracking, evidence preservation (moderation actions are append-only), a government-request handling policy, and terms of service drafted by Ethiopian counsel.

### 7.6 PII in evidence images
Photos of receipts, storefronts, staff, or license plates are personal data of **third parties** under 1321/2024. Mitigations: private-by-default evidence storage, EXIF stripping for public media, moderation review before any verification level is granted, takedown path for depicted individuals.
