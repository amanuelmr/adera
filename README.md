# Adera (አደራ) — Trusted Ethiopian Review Platform, Backend

Adera helps Ethiopian consumers make better decisions before spending money —
on electronics and phone repair, salons and barbers, online sellers and
delivery, and restaurants/cafés discovered through TikTok and Instagram. It
focuses on **trustworthy, structured, locally relevant reviews**: category-
specific criteria, evidence-graded verification, Amharic-first search, a
social-media "Reality Check", and transparent moderation.

**This repository is the backend only** (Go + PostgreSQL modular monolith).
Frontend/UX research for the next phase lives in `docs/frontend-handoff.md`.

## Quick start

Requirements: Docker + Docker Compose (Go 1.26 only if running outside Docker).

```bash
docker compose up -d --build      # PostgreSQL 18, MinIO, API on :8080
docker compose run --rm api seed  # migrations + development seed data
curl localhost:8080/health
curl localhost:8080/api/v1/categories
curl "localhost:8080/api/v1/search/targets?q=ቶሞካ"
```

Or natively against the compose services:

```bash
docker compose up -d postgres minio
cp .env.example .env && export $(grep -v '^#' .env | xargs)
go run ./cmd/api seed
go run ./cmd/api serve
```

### Development credentials (seed data — development only)

| Account | Identifier | Password |
|---|---|---|
| Admin (admin+moderator) | `admin@adera.local` | `admin12345!` (or `SEED_ADMIN_PASSWORD`) |
| Customers | `abebe@` `tigist@` `dawit@` `sara@` `yonas@example.com` | `password123` |
| Business owner (claimed "Kategna") | `owner@kategna.example.com` | `password123` |

Seeding refuses to run when `APP_ENV=production`, and startup validation
rejects `SEED_ADMIN_PASSWORD` in production.

### Try a flow

```bash
TOKEN=$(curl -s localhost:8080/api/v1/auth/login \
  -d '{"identifier":"abebe@example.com","password":"password123"}' \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["tokens"]["access_token"])')

TARGET=$(curl -s "localhost:8080/api/v1/search/targets?q=sheger" \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["data"][0]["id"])')

curl -s localhost:8080/api/v1/reviews \
  -H "Authorization: Bearer $TOKEN" -H "Idempotency-Key: demo-1" \
  -d "{\"target_id\":\"$TARGET\",\"overall_rating\":5,
       \"body\":\"Fast honest repair, showed me the original part packaging.\",
       \"discovery_source\":\"friend\",
       \"criterion_scores\":{\"repair_quality\":5,\"seller_honesty\":5}}"

curl -s "localhost:8080/api/v1/targets/$TARGET/stats"
```

## Tests and checks

```bash
go test ./...        # unit + integration (integration needs the compose postgres;
                     # skipped automatically when unreachable)
go test -race ./...
go vet ./...
govulncheck ./...
make check           # all of the above
```

Integration tests create a **disposable database per test** via
`TEST_DATABASE_ADMIN_URL` (default matches docker-compose) — real PostgreSQL,
no mocks.

## Documentation map

| Document | Contents |
|---|---|
| `api/openapi.yaml` | Full API contract (auth, schemas, errors, pagination, examples) |
| `docs/architecture.md` | System design, module map, mermaid flows, ER diagram |
| `docs/authorization-matrix.md` | Role × endpoint matrix, object-level rules |
| `docs/rating-and-ranking.md` | Every aggregate formula, rounding, sample rules, Bayesian ranking |
| `docs/verification-model.md` | Evidence levels, private-evidence guarantees, upload pipeline |
| `docs/moderation-policy.md` | Review states, report reasons, anti-manipulation, retention |
| `docs/frontend-handoff.md` | Screens, flows, API mapping, low-bandwidth/a11y/l10n guidance |
| `docs/research/` | Market research, UI inspiration, tech-stack decisions, security & legal risks |
| `CONTRIBUTING.md`, `SECURITY.md` | Workflow and security policy |

## Environment variables

See `.env.example` for the full annotated list. Required in production:
`DATABASE_URL`, `JWT_SECRET` (≥32 bytes). Argon2id parameters are validated
against the OWASP minimum at startup. Secrets come from the environment only —
nothing secret is committed.

## Operational endpoints

`GET /health` (liveness) · `GET /ready` (DB ping) · `GET /metrics`
(Prometheus; restrict at the network layer in production).

## Known limitations (honest list)

- SMS/email delivery is a console provider in development and reports
  `503 verification_unavailable` in production until a real provider is
  wired (`internal/auth/provider.go` is the integration point).
- Rate limiting is per-process; multi-replica deployments need the documented
  Redis-backed `Limiter` implementation.
- Image derivatives (thumbnails) and WebP re-encoding are deferred; public
  media is stored as one sanitized original.
- Search relevance thresholds were tuned on seed data; a native-speaker
  Amharic query test set is needed before launch.
- Legal compliance items in `docs/research/security-and-legal-risks.md` §7
  need professional review (data protection registration, takedown SLAs,
  retention/erasure workflows).
