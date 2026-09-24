# Adera mobile (Android-first)

Native client for Adera, built with Expo + Expo Router. Companion to the
Go backend in the repo root — see [`../docs/mobile-plan.md`](../docs/mobile-plan.md)
for product scope and phasing, and [`AGENTS.md`](./AGENTS.md) for Expo-specific
conventions used in this directory.

## Get started

Requires the backend running locally (`docker compose up -d --build` from the
repo root) so the app has an API to talk to.

```bash
npm install
npx expo start
```

Routes live under `src/app/` (file-based routing via Expo Router); shared
components, hooks, and constants live in `src/components/`, `src/hooks/`,
`src/constants/`. `src/api/` holds the typed API client — `client.ts` wraps
`openapi-fetch` (`apiClient`, and `unwrap()` to peel the `{data, meta}`
envelope every endpoint returns and throw `ApiError` on failure), and
`schema.d.ts` is generated from `../api/openapi.yaml`. Copy `.env.example` to
`.env` to point at a non-default backend (see `src/api/env.ts`).

## Commands

```bash
npx expo start          # dev server
npx expo lint           # lint
npx tsc --noEmit        # typecheck
npx expo-doctor         # diagnose dependency/config issues
npm run generate:api    # regenerate src/api/schema.d.ts after api/openapi.yaml changes
```
