# Tech Stack Decisions — Adera Backend (Go + PostgreSQL)

**Research date:** 2026-07-17. Each version below is marked **[verified]** (checked against the named primary source on that date) or **[assumed]**.

---

## 1. Verified Versions

| Component | Version | Verified against | Notes |
|-----------|---------|------------------|-------|
| Go | **go1.26.5** [verified] | go.dev/dl, 2026-07-17 | Local toolchain is go1.26.1; `go.mod` targets go1.26. Only the two newest majors get security patches — stay on 1.26.x and take patch releases. |
| PostgreSQL | **18** (latest minor **18.4**) [verified] | postgresql.org/support/versioning | Supported: 18.4 (EOL 2030-11-14), 17.10, 16.14, 15.18, 14.23 (**EOL 2026-11-12 — avoid**). Chosen: **18** — GA major, longest runway. |
| pgx | **v5.10.0+** [verified] | pkg.go.dev/github.com/jackc/pgx/v5 | v5 is the active major; pin whatever `go get @latest` resolves ≥ v5.10.0. |
| sqlc | v1.31.1 [verified] | sqlc releases | Available but **not used** — see §3. |
| golang-migrate | v4.19.1 [verified] | GitHub releases | Considered but **not used** — see §3. |
| distroless base | `gcr.io/distroless/static-debian12:nonroot` [assumed — family verified as current practice, exact digest pinned at build] | Google distroless repo | See §5. |

## 2. Go stdlib features leaned on (no framework)

- **`net/http.ServeMux` (Go 1.22+):** method + wildcard routing (`mux.HandleFunc("GET /api/v1/targets/{id}", h)`, `r.PathValue("id")`). No third-party router.
- Note **Go 1.26 change [verified]:** ServeMux trailing-slash redirects now use **307** instead of 301 — tests must not assert 301.
- **`log/slog`:** structured JSON logging with request IDs and redaction discipline.
- `http.MaxBytesReader`, `context` deadlines, `crypto/rand`, `crypto/subtle` for constant-time comparison.

## 3. Dependency decisions

| Dependency | Decision | Why |
|------------|----------|-----|
| `github.com/jackc/pgx/v5` + `pgxpool` | **Adopted** | Best-maintained PG driver. One pool per process; `MaxConns` explicit; contexts everywhere; `defer rows.Close()`. |
| sqlc | **Not adopted** | The repository layer uses hand-written, parameterized SQL constants with explicit scanning — the "another type-safe SQL approach" option. Rationale: no codegen step in the build, full control over dynamic filters (search/listing endpoints need composed WHERE clauses that sqlc handles poorly), and reviewable SQL colocated with the module that owns it. Revisit if query count grows past comfort. |
| golang-migrate | **Not adopted** | Migrations are plain versioned `.sql` files under `migrations/`, applied by a ~100-line runner in `internal/platform/database` (advisory lock, `schema_migrations` table, each file in a transaction, embedded via `go:embed`). Zero extra dependency tree, works identically in tests/CI/prod via `api migrate`. Up-only by design (documented; rollbacks are roll-forward migrations). |
| `golang.org/x/crypto/argon2` | **Adopted** | Argon2id, m=19456 KiB / t=2 / p=1 defaults (OWASP), configurable upward. |
| `golang.org/x/time/rate` | **Adopted** | Keyed token-bucket limiters with janitor eviction; per-IP pre-auth + per-account for sensitive flows. In-process only — documented Redis swap-point for multi-replica scale-out. |
| `github.com/golang-jwt/jwt/v5` | **Adopted** | HS256 access tokens (single-service monolith — asymmetric signing adds no value until a second verifier exists); algorithm pinned at parse. |
| `github.com/google/uuid` | **Adopted** | UUIDv4 public identifiers (documented: non-sequential, no creation-time leak). |
| `github.com/minio/minio-go/v7` | **Adopted** | Presigned **POST** policies (`content-length-range`, exact content-type, server-generated keys) against MinIO/S3. Presigned PUT rejected: does not enforce size server-side. |
| `github.com/prometheus/client_golang` | **Adopted** | `/metrics`: request counts/durations, in-flight, DB pool stats. |
| `github.com/stretchr/testify` | **Adopted (test-only)** | Assertion ergonomics for a large test suite. |
| ORMs (GORM), routers (chi/gin), Redis, Elasticsearch, Kafka | **Rejected** | Stdlib mux suffices; PostgreSQL handles search/aggregation at MVP scale; no measured requirement. |

## 4. Search strategy: Amharic (Ethiopic script) + English

Facts established [verified against PostgreSQL docs]:
- PostgreSQL ships **no Amharic FTS dictionary/stemmer**. The `simple` config lowercases and splits on non-word characters — adequate because Ethiopic has no letter case and no built-in stemmer could help anyway.
- `pg_trgm` trigrams form over Ethiopic code points (classified as letters under ICU/glibc), but each Ethiopic syllable carries more information than a Latin letter, so similarity thresholds must be tuned lower (`word_similarity` preferred; threshold 0.25 in queries, validated in tests).

Design (implemented in `internal/search`):
1. UTF-8 everywhere; NFC normalization plus mapping of Ethiopic wordspace `፡` (U+1361) and punctuation (`።` `፣` `፤`) to ASCII space, applied in Go both at index time (aliases/names are stored normalized in the index expression) and at query time.
2. Amharic homophone folding (ሀ/ሐ/ኀ-families, ሰ/ሠ, አ/ዐ, ጸ/ፀ) applied to the query and via a folded expression index — the single highest-impact trick for Amharic search quality [domain knowledge; validated with seed-data tests, needs native-speaker test set].
3. FTS: `to_tsvector('simple', name || ' ' || description)` expression GIN index + `websearch_to_tsquery('simple', q)`.
4. `pg_trgm` GIN index on normalized name and on `target_aliases.alias` for fuzzy/substring matching.
5. **Alias table** holds Latin transliterations ("Bole Cafe" ⇄ "ቦሌ ካፌ"), old names, informal names — this is how mixed-script queries actually get solved; no automatic transliteration.
6. Ranking: exact FTS match > word_similarity > substring, combined with rating-confidence score; defer external engines until PG measurably fails.

## 5. Docker production build

Multi-stage: `golang:1.26` build stage (module + build caches) → `gcr.io/distroless/static-debian12:nonroot`. `CGO_ENABLED=0`, `-trimpath -ldflags="-s -w"`. Distroless-static over scratch because it includes CA certs, tzdata, and a non-root user. No shell in the runtime image; ~15 MB.

## 6. Rate-limit design

Two in-process layers using `x/time/rate` keyed limiter maps with mutex + janitor eviction (canonical pattern, alexedwards.net):
- **Per-IP (pre-auth):** login/register/password-reset/OTP 5/min; search 2 rps burst 10; global default 10 rps burst 20.
- **Per-account (post-auth):** review creation 5/day plus per-target 30-day cooldown; reports 20/day; upload presigns 20/hour.
- 429 + `Retry-After`; keys redacted in logs. Scale-out: swap store behind the `ratelimit.Limiter` interface (Redis) when replicas > 1.
