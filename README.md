# Adera (አደራ) — Trusted Ethiopian Review Platform

Adera helps Ethiopian consumers make better decisions before spending money —
on electronics and phone repair, salons and barbers, online sellers and
delivery, and restaurants and cafés discovered through TikTok and Instagram.

This repository holds both halves of the product:
- **Backend** (Go + PostgreSQL modular monolith) — everything at the repo
  root (`cmd/`, `internal/`, `api/`, `migrations/`).
- **Mobile app** (Expo + React Native, Android-first) — [`mobile/`](mobile/),
  see [`mobile/README.md`](mobile/README.md).

Product scope and phasing for the mobile client live in
[`docs/mobile-plan.md`](docs/mobile-plan.md); the earlier web/UX research is in
[`docs/frontend-handoff.md`](docs/frontend-handoff.md).

## Motivation

In Ethiopia, deciding where to spend money is largely an act of faith. Reviews
live in scattered TikTok and Telegram posts, business directories are shallow
contact-info dumps, Google Maps coverage is thin and often wrong outside a few
landmarks, and social-media hype has no accountability loop when the viral café
disappoints. No existing platform combines verified experiences, Amharic-first
discovery, resident-oriented categories, public business accountability, and a
way to compare online hype against real experience.

Adera is that missing layer. It is built around **trustworthy, structured,
locally relevant reviews** rather than a single star rating:

- **Category-specific criteria** stored in the database (electronics
  authenticity and warranty, salon punctuality, delivery refund handling,
  restaurant taste and value) instead of one generic score.
- **Evidence-graded verification** — a review is only labeled verified after a
  moderator accepts receipt or location evidence, which stays private.
- **Amharic-first Unicode search** with homophone folding and transliteration
  aliases, so "Bole Cafe" and "ቦሌ ካፌ" find the same place.
- **A restaurant "Reality Check"** that aggregates whether a socially-hyped
  spot matched expectations — in neutral, aggregate language, never accusing a
  creator.
- **Transparent moderation**: published-first, reports never auto-hide, and
  every decision is written to an append-only audit trail.
- **Visible review disclosures** for discounts, free products/services,
  payments, and material relationships, returned on every review surface.

The name *Adera* (አደራ) is Amharic for a sacred trust — something entrusted to
you to safeguard.

## Quick Start

Requirements: Docker + Docker Compose (Go 1.26 only if running outside Docker).

```bash
docker compose up -d --build      # PostgreSQL 18, MinIO, API on :8080
docker compose run --rm api seed  # migrations + development seed data
curl localhost:8080/health
curl localhost:8080/api/v1/categories
curl "localhost:8080/api/v1/search/targets?q=ቶሞካ"
```

Or run the API natively against the Compose services:

```bash
docker compose up -d postgres minio
cp .env.example .env && export $(grep -v '^#' .env | xargs)
go run ./cmd/api seed
go run ./cmd/api serve
```

### Development credentials (seed data — development only)

| Account | Identifier | Password |
|---|---|---|
| Admin (admin + moderator) | `admin@adera.local` | `admin12345!` (or `SEED_ADMIN_PASSWORD`) |
| Customers | `abebe@example.com` … `yonas@example.com` | `password123` |
| Business owner (claimed "Kategna") | `owner@kategna.example.com` | `password123` |

Seeding refuses to run when `APP_ENV=production`, and startup validation
rejects `SEED_ADMIN_PASSWORD` in production.

## Usage

The API is versioned under `/api/v1`, uses bearer access tokens, and returns a
uniform `{"data": …}` / `{"error": {"code", "message"}}` envelope. The full
contract is in [`api/openapi.yaml`](api/openapi.yaml).

### Example: log in, search, review, read aggregates

```bash
TOKEN=$(curl -s localhost:8080/api/v1/auth/login \
  -d '{"identifier":"abebe@example.com","password":"password123"}' \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["tokens"]["access_token"])')

TARGET=$(curl -s "localhost:8080/api/v1/search/targets?q=sheger" \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["data"][0]["id"])')

curl -s localhost:8080/api/v1/reviews \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: demo-1" \
  -d "{\"target_id\":\"$TARGET\",\"overall_rating\":5,\"body\":\"Fast honest repair, showed me the original part packaging.\",\"discovery_source\":\"friend\",\"criterion_scores\":{\"repair_quality\":5,\"seller_honesty\":5}}"

curl -s "localhost:8080/api/v1/targets/$TARGET/stats"
```

### Operational endpoints

- `GET /docs` — Swagger UI for the API contract
- `GET /openapi.yaml` — raw OpenAPI contract for tooling/imports
- `GET /health` — liveness
- `GET /ready` — readiness (checks database connectivity)
- `GET /metrics` — Prometheus metrics (restrict at the network layer in production)

### Environment variables

See [`.env.example`](.env.example) for the full annotated list. Required in
production: `DATABASE_URL` and `JWT_SECRET` (≥ 32 bytes). Argon2id parameters
are validated against the OWASP minimum at startup. Secrets are read from the
environment only — nothing secret is committed.

Setting `SMTP_HOST` turns on emailed verification codes and password resets;
leaving it empty keeps codes on the console in development and returns
`503 verification_unavailable` in production. To exercise the real delivery
path locally, point it at a catcher such as MailHog:

```bash
docker run -d -p 1025:1025 -p 8025:8025 mailhog/mailhog
export SMTP_HOST=127.0.0.1 SMTP_PORT=1025 SMTP_TLS=none \
       SMTP_FROM_ADDRESS=no-reply@adera.local
```

### Documentation map

| Document | Contents |
|---|---|
| [`api/openapi.yaml`](api/openapi.yaml) | Full API contract (auth, schemas, errors, pagination, examples) |
| [`docs/architecture.md`](docs/architecture.md) | System design, module map, Mermaid flows, ER diagram |
| [`docs/authorization-matrix.md`](docs/authorization-matrix.md) | Role × endpoint matrix, object-level rules |
| [`docs/rating-and-ranking.md`](docs/rating-and-ranking.md) | Aggregate formulas, rounding, sample rules, Bayesian ranking |
| [`docs/verification-model.md`](docs/verification-model.md) | Evidence levels, private-evidence guarantees, upload pipeline |
| [`docs/moderation-policy.md`](docs/moderation-policy.md) | Review states, report reasons, anti-manipulation, retention |
| [`docs/frontend-handoff.md`](docs/frontend-handoff.md) | Screens, flows, API mapping, low-bandwidth / a11y / l10n guidance |
| [`docs/research/`](docs/research/) | Market research, UI inspiration, tech-stack decisions, security & legal risks |

## Contributing

Contributions are welcome. Before opening a pull request, run the full check
suite (unit + integration tests, race detector, and vulnerability scan):

```bash
go test ./...       # integration tests need the Compose Postgres; skipped when unreachable
go test -race ./...
go vet ./...
govulncheck ./...
make check          # all of the above
```

Integration tests create a **disposable database per test** via
`TEST_DATABASE_ADMIN_URL` (the default matches Docker Compose) — real
PostgreSQL, no mocks.

Please keep the module-by-capability layout, write parameterized SQL only, add
an authorization test for any endpoint that loads an object by ID, and update
the relevant document when behavior changes. Full guidelines are in
[`CONTRIBUTING.md`](CONTRIBUTING.md); security policy is in
[`SECURITY.md`](SECURITY.md).

## Known limitations

- SMS delivery has no provider: phone verification reports
  `503 verification_unavailable` until an SMS or Telegram integration is added
  (`internal/auth/provider.go` is the integration point). Email verification
  and password reset deliver over SMTP once `SMTP_HOST` is configured, and
  fall back to the console in development.
- Push delivery requires `FCM_CREDENTIALS_FILE`; without it outbox events
  accumulate for later replay. Messages are data-only, so a client must render
  and localize them (`internal/notifications/fcm.go`).
- Rate limiting is per-process; multi-replica deployments need the documented
  Redis-backed `Limiter` implementation.
- Public review photos are stored as a sanitized original plus one thumbnail
  (480 px longest edge, served as `thumb_url`). Further sizes and WebP
  re-encoding are still deferred.
- Search relevance thresholds were tuned on seed data; a native-speaker Amharic
  query test set is needed before launch.
- Legal compliance items in
  [`docs/research/security-and-legal-risks.md`](docs/research/security-and-legal-risks.md)
  §7 need professional review (data-protection registration, takedown SLAs,
  retention/erasure workflows).
