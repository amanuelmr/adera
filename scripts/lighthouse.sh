#!/usr/bin/env bash
# Lighthouse budget check for the server-rendered web pages (internal/pages),
# against docs/frontend-handoff.md §5. CI runs this exact script; run it
# locally the same way:
#
#   DATABASE_URL=postgres://adera:adera@localhost:55432/lhci_check?sslmode=disable \
#     scripts/lighthouse.sh
#
# DATABASE_URL must point at a database this script may migrate and seed —
# never a real one. Reports land in .lighthouseci/.
#
# Settings (lighthouserc.json): DevTools "Slow 3G" throttling (400 ms RTT,
# 400 kbps, 4x CPU) on Lighthouse's default mobile profile; budgets are the
# handoff's 300 KB per page, 100 KB of JS, interactive within 5 s.
# The robots-txt audit is skipped: Lighthouse downloads robots.txt from
# inside the page, which the pages' strict CSP (no connect-src) blocks.
# Crawlers fetch it directly; internal/pages tests cover it instead.
set -euo pipefail

: "${DATABASE_URL:?set DATABASE_URL to a throwaway database}"
PORT="${LHCI_PORT:-18080}"
BASE="http://localhost:${PORT}"
LHCI_VERSION="0.15.1"

cd "$(dirname "$0")/.."

export APP_ENV=development HTTP_ADDR=":${PORT}" STORAGE_ENABLED=false LOG_LEVEL=warn
go build -o "${TMPDIR:-/tmp}/adera-lhci-api" ./cmd/api
API="${TMPDIR:-/tmp}/adera-lhci-api"

"$API" seed

"$API" serve &
API_PID=$!
trap 'kill "$API_PID" 2>/dev/null || true' EXIT

for _ in $(seq 1 60); do
  curl -sf "${BASE}/health" >/dev/null && break
  sleep 1
done
curl -sf "${BASE}/health" >/dev/null || { echo "API did not start" >&2; exit 1; }

# The seeded restaurant with the most reviews: a realistic, full page.
read -r SLUG TARGET_ID < <(curl -sf "${BASE}/api/v1/targets/top-rated?limit=50" |
  node -e 'let d="";process.stdin.on("data",c=>d+=c).on("end",()=>{
    const t=JSON.parse(d).data.sort((a,b)=>(b.review_count??0)-(a.review_count??0))[0];
    if(!t||!t.review_count){process.exit(1)} console.log(t.slug, t.id)})')
REVIEW_ID=$(curl -sf "${BASE}/api/v1/targets/${TARGET_ID}/reviews?limit=1" |
  node -e 'let d="";process.stdin.on("data",c=>d+=c).on("end",()=>console.log(JSON.parse(d).data[0].id))')

echo "Measuring /, /c/restaurant_cafe, /t/${SLUG}, /r/${REVIEW_ID}, /trust"
LHCI=(npx --yes "@lhci/cli@${LHCI_VERSION}")
"${LHCI[@]}" collect --url="${BASE}/" --url="${BASE}/c/restaurant_cafe" \
  --url="${BASE}/t/${SLUG}" --url="${BASE}/r/${REVIEW_ID}" --url="${BASE}/trust"
"${LHCI[@]}" assert
"${LHCI[@]}" upload --target=filesystem --outputDir=.lighthouseci/reports
