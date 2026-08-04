# Frontend Handoff — UX Research & Recommendations

**Adera · Ethiopian trusted review platform · prepared 2026-07-17.**
Backend-only phase is complete; this document gives the next (frontend) phase its direction. No frontend code exists or should be inferred to exist.

The "API mapping" column/section references implemented endpoints; see `api/openapi.yaml` for schemas.

---

## 1. Screens the frontend will need (MVP set)

Ordered roughly by user-journey priority.

1. **Home / Discover** — search bar, category tiles (the four launch categories), "Top rated," "Trending."
2. **Search results / Category browse** — list-first (not map-first — maps are data-heavy); filter bottom sheet: area, rating ≥ x, verified-only, target type; applied-filter chips.
3. **Target profile (business/restaurant/seller/product)** — header (name, category, area), aggregate block: average to one decimal + count + tappable 5-bar histogram + per-criterion mini-bars + recent-trend indicator, review list with sort, owner-response threads, "Write a review" CTA persistently visible.
4. **Restaurant Reality Check panel** — expectation-match distribution for social-media visitors, recent vs historical average, confidence messaging for small samples.
5. **Review submission flow** (multi-step, one question per screen — see §2 Flow B).
6. **Evidence submission** — private receipt/order upload with an explicit privacy explanation ("moderators only — never shown publicly").
7. **Review detail / permalink** — shareable card view (screenshot-friendly for Telegram), helpful button, report link, owner response.
8. **Reviewer profile** — display name, join date, verification tier, review count, list of own reviews.
9. **Onboarding & auth** — register/login (email or phone + password), OTP verification, password reset, session management.
10. **Report-review flow** — reason picker (enum from backend), free-text detail, confirmation, "my reports" status view.
11. **Business claim flow** — find business → claim with method + evidence → track claim status.
12. **Business-owner dashboard (web-first is fine)** — respond to reviews (one response per review, editable with history), see review stats.
13. **Moderator dashboard** — report queue, review/evidence detail, approve/hide/reject/restore actions, claim approvals, audit history.
14. **Activity inbox** — unread badge and localized claim, report, moderation, evidence, and owner-response updates; each item deep-links to its subject.
15. **Trust Center (static but essential)** — how ratings are computed, what "verified" means, moderation policy. Amharic and English from day one.
16. **Empty states** — designed, not defaulted: target with 0 reviews ("Be the first to review"), search with 0 results (suggest wider area / different spelling — important given Amharic/Latin transliteration variance), user with 0 reviews.
17. **Offline / error states** — cached-content banner, retry queue indicator for pending review submissions.

Later-phase screens the design system should anticipate: dish/item-level ratings, tag-chip analytics, AI summaries with evidence links, monthly award pages per neighborhood.

## 2. Expected user flows

**Flow A — Reader (majority of traffic):** open link from Telegram/search → target profile → scan aggregate + histogram → read top positive + top critical → filter by 1–2★ to stress-test → read owner responses → decide. *Implication: the profile page must be fully useful without login and load fast; never gate reading behind auth (all read endpoints are public).*

**Flow B — Write a review (target: under 90 seconds on a phone):**
1. Tap "Write a review" → if not logged in, auth inline (persist draft client-side).
2. Step 1: overall stars with adjective labels ("በጣም ጥሩ / Excellent").
3. Step 2: category criteria stars (fetched from `GET /categories/{id}/criteria`, each skippable unless required).
4. Step 3: text box with a prompt question, photo add (client-side compression before presigned upload).
5. Step 4: context — experience date, discovery source; if discovery source is social media, the expectation-match question appears; optional price paid.
6. Submit with an `Idempotency-Key` header (backend deduplicates weak-connection retries).
7. Confirmation: what happens next, share card.

**Flow C — Business owner:** search own business → claim (method + evidence) → moderator approves → get access to respond → response published with "Owner" label and timestamps.

**Flow D — Report:** three taps from any review card → reason enum → submit → confirmation → status in "my reports."

**Flow E — Moderation lifecycle (user-comprehensible):** submitted → published (default) → possibly `under_review` after reports → hidden/rejected/restored by a moderator → reviewer can see own review status; aggregates always exclude non-published reviews.

## 3. API-to-screen mapping

| Screen | Endpoints |
|---|---|
| Home / Discover | `GET /categories` · `GET /targets/top-rated` · `GET /targets/trending` |
| Search results | `GET /search/targets?q=&category=&city=&area=&type=&min_rating=&verified=&sort=&cursor=` |
| Category browse | `GET /categories/{idOrCode}` · `GET /search/targets?category=` |
| Target profile | `GET /targets/{idOrSlug}` · `GET /targets/{id}/stats` · `GET /targets/{id}/reviews?sort=&filter…&cursor=` |
| Reality Check panel | `GET /targets/{id}/reality-check` |
| Review submission | `GET /categories/{id}/criteria` · `POST /reviews` (Idempotency-Key) · `POST /reviews/{id}/media` + `POST /media/uploads` (presign) + `POST /media/uploads/{id}/finalize` |
| Evidence submission | `POST /reviews/{id}/evidence` (presign, private bucket) + finalize |
| Review permalink | `GET /reviews/{id}` · `PUT /reviews/{id}/helpful` · `DELETE /reviews/{id}/helpful` · `POST /reviews/{id}/reports` |
| Reviewer profile (self) | `GET /users/me` · `PATCH /users/me` · `GET /users/me/reviews` · `GET /users/me/reports` |
| Activity inbox | `GET /users/me/notifications?unread=` · `GET /users/me/notifications/unread-count` · `PUT /users/me/notifications/{id}/read` · `PUT /users/me/notifications/read-all` |
| Onboarding & auth | `POST /auth/register` · `POST /auth/login` · `POST /auth/refresh` · `POST /auth/logout` · `POST /auth/logout-all` · `POST /auth/verify/request` · `POST /auth/verify/confirm` · `POST /auth/password-reset/request` · `POST /auth/password-reset/confirm` · `GET /auth/sessions` · `DELETE /auth/sessions/{id}` |
| Business claim | `POST /businesses/{id}/claims` · `GET /claims/mine` |
| Owner dashboard | `GET /businesses/{id}/stats` · `POST /reviews/{id}/response` · `PUT /responses/{id}` · `POST /reviews/{id}/reports` |
| Moderator dashboard | `GET /moderation/reports?status=` · `GET /moderation/reports/{id}` · `GET /moderation/reviews/{id}/evidence` · `POST /moderation/reviews/{id}/decision` · `POST /moderation/claims/{id}/decision` · `GET /moderation/audit?subject=` |
| Admin | `POST/PATCH /admin/categories…` · `POST /admin/users/{id}/suspend` · `POST /admin/targets/{id}/merge` |
| Trust Center | static content + `GET /targets/{id}/stats` formulas documented in docs/rating-and-ranking.md |

**Intentionally deferred frontend needs** (backend hooks exist or are documented): review-language filter UI beyond code filter, topic chips/keyword extraction, dish-level ratings, external push/SMS delivery (the durable notification outbox is ready for a provider adapter), map view (coordinates are stored; no tile/geo-radius endpoint yet), machine translation of reviews, moderation-stats public page (counters derivable from audit table; no public endpoint yet).

## 4. Mobile-first recommendations

- Design at 360×640 first (dominant Android class in Ethiopia); scale up, never down.
- Single-column everything; bottom sheets for filters and reporting; sticky bottom CTA for "Write a review."
- Touch targets ≥44×44px (stars especially — generous hit areas, visual stars smaller than tap zones).
- One question per screen in the submission flow with progress dots; inline validation; never clear state on back.
- Review cards: collapse text at ~5 lines with "Read more"; paged lists (backend default page size 20, max 50).
- Assume Android WebView/Chrome + low-RAM devices: avoid heavy JS frameworks; server-rendered or lightweight-hydration (SSR + islands) strongly preferred over a large SPA bundle.
- Share-to-Telegram/WhatsApp as a first-class action on target pages and review cards (primary organic channel).

## 5. Low-bandwidth recommendations (Ethiopia-specific)

Context: a large share of connections are still 2G/3G; data cost is a real household expense; coverage outside Addis is inconsistent.

**Payload budgets (enforce in CI via Lighthouse "Slow 3G"):**
- First load of a target page: ≤ 300 KB total (HTML+CSS+JS+fonts), ≤ 500 KB with images.
- JS bundle ≤ 100 KB gzipped for the core experience.
- API responses: reviews paginated (20/page); target-detail JSON ≤ 30 KB; list DTOs are slim (no full review bodies in list views).
- Interactive < 5s on Slow 3G, < 2s on 4G.

**Image strategy:**
- Client-side resize/compress before upload (max ~1600px, quality ~0.7) — saves the *uploader's* data too. Backend enforces 10 MB hard cap via upload policy.
- Serve size variants (thumb/card/full) — derivative generation is deferred backend work; MVP stores one re-encoded original.
- Lazy-load below the fold; LQIP placeholders; explicit width/height.
- Optional "data saver" toggle: text-only review lists, photos on demand.

**Offline tolerance:**
- PWA service worker: cache-first shell, stale-while-revalidate for target pages/review lists, network-first for aggregates.
- Queue review submissions and helpful votes offline — the backend accepts an `Idempotency-Key` on review creation and helpful votes are idempotent PUT/DELETE, so retries are safe.
- Show cached-timestamp banners honestly ("Saved 2 hours ago").

## 6. Accessibility (WCAG 2.1/2.2 AA essentials)

- **Star rating input:** real `<fieldset>` of radio buttons with visible-on-focus labels ("3 stars — Good"); arrow-key operable; explicit submit; a no-JS-functional pattern exists and is preferable.
- **Star display (read-only):** text alternative ("Rated 4.3 out of 5 from 128 reviews"); 3:1 contrast for glyphs; filled-vs-empty must differ by more than color.
- **Histogram bars:** each row a button with an accessible name ("Filter: 2-star reviews, 14 reviews"); announce applied filters via live region.
- **Forms:** visible labels (not placeholder-only), errors linked with `aria-describedby`, 24×24 minimum target (WCAG 2.2) but hold the 44px internal standard.
- **General:** full keyboard operability including bottom sheets; `lang` attributes switching correctly between `am`, `en`, `om` per text node (the backend stores a language code on reviews and returns translated labels keyed by language); relative dates need absolute-date equivalents in accessible names.
- **Honest caveat:** Android TalkBack Amharic voice quality is inconsistent and the 2026 state was not verified — test on-device early.

## 7. Localization — Amharic / English / Afaan Oromo

**Scripts & direction:**
- Amharic uses the Ethiopic (Geʽez) script — **left-to-right; RTL support is NOT needed** for any launch language. Keep strings externalized anyway.
- Afaan Oromo uses Latin script (Qubee) — no special font, but doubled vowels/consonants make words long (see expansion).

**Fonts:**
- **Noto Sans Ethiopic** (Google Fonts, variable) as the Ethiopic face; ship it **subsetted** (~80–150 KB even subsetted — load only when locale = am); `font-display: swap`.
- Ethiopic glyphs are denser/taller than Latin: line-height ≥1.6, body text ≥14px, test bold weights at small sizes.

**Text expansion & layout:**
- Design buttons/labels for **+35% width tolerance**; allow two-line wrapping on chips/buttons; never hard-truncate translated labels.
- Ethiopic wordspace (፡) and punctuation (። ፣ ፤) must render and line-break correctly. (The backend search layer normalizes these — see internal/search.)

**Dates, numbers, calendar, clock:**
- Backend stores UTC and returns RFC 3339 Gregorian — non-negotiable. Render **relative dates by default** ("ከ2 ሳምንት በፊት"), which sidesteps the Ethiopian-calendar question in most UI; where absolute dates appear, offer a user setting (Ethiopian ⇄ Gregorian) via a maintained conversion library (e.g., andegna/calender approach).
- **Ethiopian 12-hour day clock** (starts at sunrise) is a real ambiguity hazard for opening hours — no major platform handles it well; treat as an open design question needing local user research. **[unverified best practice]**
- Use Western digits everywhere; Geʽez numerals (፩፪፫) are ceremonial. Currency: "1,200 ብር" / "ETB 1,200" per locale.

**Language behavior:**
- Locale switcher one tap from anywhere; persists via `PATCH /users/me` (`preferred_language`).
- User content is language-mixed (Amharic, English, transliterated Amharic). The backend stores a `language` code per review and supports filtering; search tolerates transliteration variance via alias tables and trigram matching.
- Translate the *trust-critical* surfaces first and best: moderation explanations, "what verified means," report flows. Backend returns stable machine-readable `code` values on all errors so every system message can be localized client-side.

**Open questions for the frontend phase:** real-device rendering of Noto Sans Ethiopic on the cheap-Android fleet; Amharic keyword-search quality against real review text; Ethiopian-time display conventions; a fresh hands-on teardown of 3–4 global platforms on-device before visual design (patterns only, copy nothing).
