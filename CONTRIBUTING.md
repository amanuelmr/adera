# Contributing

## Setup

```bash
docker compose up -d postgres minio
cp .env.example .env
go run ./cmd/api seed
go run ./cmd/api serve
```

## Before opening a PR

```bash
make check   # go vet, go test, go test -race, govulncheck
```

Integration tests need the compose PostgreSQL (they create disposable
databases and are skipped when it's unreachable — CI always runs them).

## Code conventions

- Module-by-capability layout (`internal/<capability>`): each module owns its
  domain validation, SQL, HTTP transport, and authorization checks. No
  `utils`/`helpers`/`common` packages.
- Handlers never contain SQL or domain rules; services return `*web.Error`
  with stable machine-readable codes.
- All SQL is parameterized — string-concatenated user input is an automatic
  review rejection. Dynamic filters append to the args slice and reference
  `$n` placeholders.
- Every table change is a new migration file (`migrations/NNNN_name.sql`).
  Migrations are up-only: to undo, write a new forward migration.
- Any endpoint that loads an object by ID needs an authorization test with
  another user's credentials (see docs/authorization-matrix.md invariants).
- Aggregate-affecting changes (review status, scores, verification) must
  update `target_rating_stats` in the same transaction and extend the
  aggregate tests.
- User-facing strings on the API are error `code`s (localizable), not prose
  the frontend must parse.
- Never log tokens, passwords, OTPs, presigned URLs, or payment references
  (`internal/platform/logging` redacts known keys defensively, but don't rely
  on it).

## Commit style

Small, focused commits; imperative subject lines ("Add claim revocation
audit"); reference the doc you updated when behavior changes (rating formulas
→ docs/rating-and-ranking.md, etc.).
