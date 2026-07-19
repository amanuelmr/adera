# Architecture

Adera is a **modular monolith**: one deployable Go service, PostgreSQL for
data + search + aggregation, S3-compatible object storage for media. No Redis,
no Elasticsearch, no queues — PostgreSQL demonstrably covers MVP search,
filtering, transactions, and aggregation needs (see
docs/research/tech-stack-decisions.md for the reasoning and measured
trade-offs).

## System overview

```mermaid
flowchart LR
    Client[Mobile-first web client\n(next phase)] -->|JSON /api/v1| API

    subgraph API[Go service — modular monolith]
        direction TB
        MW[middleware: recover, request-id,\nmetrics, logging, headers, CORS,\ntimeout, authenticate]
        MODS[auth · users · categories · locations\ntargets · businesses · claims · reviews\nratings · search · media · moderation · admin]
        PLAT[platform: config · web · security\ndatabase · ratelimit · storage · logging]
        MW --> MODS --> PLAT
    end

    API -->|pgx pool| PG[(PostgreSQL 18\nFTS 'simple' + pg_trgm + am_fold)]
    API -->|presign / finalize| S3[(S3-compatible storage\npublic + private buckets)]
    Client -->|presigned POST upload| S3
    Ops[Prometheus] --> |/metrics| API
```

## Module layout

```
cmd/api                 serve | migrate | seed
internal/app            wiring shared by cmd and integration tests
internal/platform/      config, web (errors/JSON/middleware/pagination/metrics),
                        security (argon2id, JWT), database (pool + migrator),
                        ratelimit, storage (minio + memory), logging, testdb
internal/<capability>/  auth, users, categories, locations, businesses,
                        targets, reviews, ratings, search, media, claims,
                        moderation, admin
migrations/             embedded, versioned, up-only SQL
```

Each capability module owns its domain validation, SQL, HTTP transport, and
authorization checks (`models.go` / `service|repo.go` / `handler.go`).
Dependencies point downward only (modules → platform); the few cross-module
edges are explicit constructor parameters (e.g., moderation takes the reviews
repo so status changes and aggregate updates share a transaction). There are
no `utils` packages and no interfaces without a second implementation (the
two interfaces — `storage.Store`, `ratelimit.Limiter` — each have real
substitutes: MinIO/memory, keyed/unlimited).

Layering rule: HTTP handlers never contain SQL or domain rules; services never
import `net/http` types beyond error mapping. Services return `*web.Error`
values with stable machine codes; the transport layer serializes them
uniformly with request IDs.

## Key design decisions

| Decision | Rationale |
|---|---|
| stdlib `net/http` ServeMux (1.22+ patterns) | No router dependency; method+wildcard routing suffices |
| Hand-written parameterized SQL (no sqlc/ORM) | Dynamic filters compose poorly in codegen; SQL stays reviewable next to its module |
| ~100-line embedded migration runner | Advisory lock + per-file transactions + `schema_migrations`; zero dependency tree; up-only (rollbacks are roll-forward) |
| Aggregates as transactional deltas | `target_rating_stats` updated in the same tx as every review mutation; CHECK constraints + recount tests guard drift |
| JWT (HS256) access + opaque rotating refresh tokens | Single verifier service; token families with reuse detection (see below) |
| Everything stages to the private bucket | No user upload is ever directly public; public media is re-encoded (EXIF stripped) before promotion |
| In-process rate limiting | Correct for single instance; `Limiter` interface is the Redis swap-point at scale-out |

## Review submission flow

```mermaid
sequenceDiagram
    participant C as Client
    participant API as reviews module
    participant DB as PostgreSQL

    C->>API: POST /reviews (Idempotency-Key)
    API->>DB: register idempotency key (PK-race-safe)
    alt replay with stored response
        DB-->>API: stored body+status
        API-->>C: replayed response (Idempotent-Replay: true)
    else fresh
        API->>DB: BEGIN
        DB->>DB: target published? cooldown (30d/target)? daily cap (5/24h)?
        DB->>DB: load category criteria, validate codes/scales/required
        DB->>DB: INSERT review + criterion scores
        DB->>DB: UPDATE target_rating_stats (delta +1)
        API->>DB: COMMIT
        API->>DB: store response on idempotency key
        API-->>C: 201 review
    end
```

## Authentication refresh flow

```mermaid
sequenceDiagram
    participant C as Client
    participant API as auth module
    participant DB as PostgreSQL

    C->>API: POST /auth/refresh {refresh_token}
    API->>DB: SELECT session BY sha256(token) FOR UPDATE
    alt token already rotated or revoked (reuse = theft signal)
        API->>DB: revoke ENTIRE family (commit)
        API-->>C: 401
    else valid
        API->>DB: INSERT new session row (same family), link replaced_by
        API->>DB: read roles, COMMIT
        API-->>C: new access JWT (15 min) + new refresh token
    end
```

## Business claim flow

```mermaid
flowchart LR
    A[Owner submits claim\nmethod + message] --> B{Moderator decision}
    B -->|approved| C[one tx: membership granted\n+ business_owner role\n+ business marked claimed\n+ audit row]
    B -->|rejected| D[status rejected + audit row]
    C --> E[Owner can respond to reviews,\nedit target/business, see stats]
    C -.-> F[revoke: membership removed,\nrole dropped if last, audit row]
```

## Moderation flow

```mermaid
flowchart TD
    R[User/business report\nreason + details] --> Q[Queue oldest-first\n24h SLA ordering]
    Q --> M{Moderator}
    M -->|review decision| S[status change + aggregate delta\nin one transaction]
    M -->|evidence decision| V[accept → verification upgrade\n+ verified aggregates]
    M -->|resolve/dismiss report| T[reporter sees status]
    S --> A[(moderation_actions\nappend-only audit)]
    V --> A
    T --> A
```

## Entity relationships (core)

```mermaid
erDiagram
    users ||--o{ user_roles : has
    users ||--o{ auth_sessions : "token families"
    users ||--o{ reviews : writes
    users ||--o{ helpful_votes : casts
    users ||--o{ reports : files
    users ||--o{ business_claims : submits
    businesses ||--o{ business_members : "authorizes (via claims)"
    businesses ||--o{ review_targets : owns
    businesses ||--o{ business_responses : posts
    categories ||--o{ category_criteria : defines
    categories ||--o{ review_targets : classifies
    cities ||--o{ areas : contains
    cities ||--o{ review_targets : locates
    review_targets ||--o{ target_aliases : "known as"
    review_targets ||--|| target_rating_stats : aggregates
    review_targets ||--o{ reviews : receives
    reviews ||--o{ review_criterion_scores : scores
    reviews ||--o{ review_media : "public photos"
    reviews ||--o{ review_evidence : "PRIVATE evidence"
    reviews ||--o| business_responses : "one response"
    reviews ||--o{ helpful_votes : receives
    business_responses ||--o{ business_response_edits : audits
    reports }o--|| reviews : about
    moderation_actions }o--|| users : "actor"
```

## Reliability & observability

Structured JSON logs (slog, redaction layer), request IDs on every response,
`/health` (liveness), `/ready` (DB ping), `/metrics` (Prometheus: request
counts/latency by route pattern, in-flight), panic recovery, config validation
at startup, HTTP server timeouts, per-request 30s context deadline inherited
by all queries, pooled connections with lifetime/idle limits, graceful
shutdown on SIGINT/SIGTERM.

## Search

`am_fold()` (SQL, migration 0001) folds Amharic homophone series
(ሐ/ኀ→ሀ, ሠ→ሰ, ዐ→አ, ፀ→ጸ) and maps Ethiopic punctuation to spaces; queries and
index expressions both pass through `am_fold(lower(...))`. Relevance = FTS
('simple' config) rank ×2 + word-similarity over names and aliases + a small
rating boost. Alias rows carry transliterations ("Bole Cafe" ⇄ "ቦሌ ካፌ") —
the practical answer to mixed-script search.
