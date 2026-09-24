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
`src/constants/`.

## Commands

```bash
npx expo start        # dev server
npx expo lint         # lint
npx tsc --noEmit       # typecheck
npx expo-doctor        # diagnose dependency/config issues
```
