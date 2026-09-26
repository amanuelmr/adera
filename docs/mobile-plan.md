# Mobile App Plan — Android-first client

**Adera · prepared 2026-08-15.** Companion to [`frontend-handoff.md`](frontend-handoff.md),
which covers the web/UX research. This document covers the **native mobile client**
and the backend work it requires. No mobile or web frontend code exists yet.

Where the two documents disagree, this one states why explicitly (see §2).

---

## 1. Product decision — what the app is for

The app is the **retention surface**, not the acquisition surface.

`frontend-handoff.md` §2 identifies Flow A (a reader arriving from a Telegram or
search link) as the majority of traffic. **A native app cannot serve that traffic** —
nobody installs an APK to read one review. That audience needs a web page.

The app exists for the people who come *back*:

| Audience | Why the app, not the web |
|---|---|
| Reviewers | Reviewing happens **at the venue**, on bad or no signal. Camera access, on-device compression, and a durable offline submit queue. |
| Nearby discovery | GPS "what's good around here" — impossible to do well in a mobile browser. |
| Returning users | Push notifications drive return visits. Web push on Android Chrome is unreliable and invisible on many OEM builds. |
| Business owners | Owners check a phone, not a desktop. Responding to a review should be a notification tap. |

**Decision: build the Android app as the primary product, plus a thin
server-rendered web layer for target profiles and review permalinks only** — so
shared links land somewhere useful and convert to installs. The full web frontend
described in `frontend-handoff.md` is deferred.

### 1.1 Correcting one assumption in the handoff doc

`frontend-handoff.md` §4 recommends against "a large SPA bundle" on
low-bandwidth connections and concludes SSR + islands. That reasoning is correct
**for the web**, but it does not transfer to a native app: an app ships its UI
once through the Play Store and thereafter sends only JSON. For a returning user
a native app is *cheaper* on data than any PWA.

The real cost of an app is **install size over expensive data** (§7), not runtime
payload. Budget for it explicitly rather than assuming it away.

### 1.2 Platform

**Android only for v1.** iOS share in the target market does not justify a second
platform. Keep the codebase cross-platform-capable so iOS is a later decision, not
a rewrite.

## 2. Scope — what ships in the app, and what does not

Screen numbers refer to `frontend-handoff.md` §1.

**In the app:**

| # | Screen | Notes |
|---|---|---|
| 1 | Home / Discover | Category tiles, top-rated, trending |
| 2 | Search + filters | Bottom-sheet filters; **single page, no infinite scroll** (§4.4) |
| 3 | Target profile | Aggregate block, histogram, review list, owner responses |
| 4 | Reality Check panel | Restaurant/café targets only |
| 5 | Review submission | **The flagship flow.** Offline-queued (§5) |
| 6 | Evidence submission | Private upload, explicit privacy copy |
| 7 | Review permalink | Share-to-Telegram as a first-class action |
| 8 | Reviewer profile (self) | |
| 9 | Onboarding & auth | Email-first (§6) |
| 10 | Report-review flow | |
| 14 | Activity inbox | Backed by push |
| — | Near me | **New — not in the handoff doc.** Requires backend work (§4.3) |
| 12 | Owner: respond to reviews | Response only; not the full dashboard |

**Not in the app** (web or deferred):

| # | Screen | Why |
|---|---|---|
| 11 | Business claim flow | Rare, document-heavy. Web. |
| 12 | Full owner dashboard | Desk task. Web, as the handoff doc already says. |
| 13 | Moderator dashboard | Desk task with long queues. Web. |
| 15 | Trust Center | Static content. Web, linked from the app. |
| — | Admin | Web. |

**Thin web layer (v1):** target profile, review permalink, Trust Center, and an
app-install banner. Everything else on the web waits.

## 3. What the backend already provides

Verified against `api/openapi.yaml` and `internal/`. No architectural change is
needed to support a mobile client — these were designed for it:

| Capability | Evidence |
|---|---|
| Bearer access + single-use refresh rotation with theft detection | `api/openapi.yaml` `/auth/refresh` |
| Multi-session listing and per-device revocation | `/auth/sessions`, `DELETE /auth/sessions/{id}` |
| Cursor pagination (`meta.next_cursor`) | `components/parameters/cursor` |
| `Idempotency-Key` on review creation; replays return the stored response | `POST /reviews` |
| Idempotent helpful votes (`PUT`/`DELETE`) | `/reviews/{id}/helpful` |
| Presigned direct-to-storage upload; server strips EXIF/GPS on finalize | `/reviews/{id}/media`, `/media/uploads/{id}/finalize` |
| Durable notification outbox behind a pluggable `Provider` interface | `internal/notifications/dispatcher.go`, `provider.go` |
| Notifications carry `event_type` + `data`, not server-rendered text | `internal/notifications/notifications.go` — client interpolates and localizes |
| Stable machine-readable error `code` on every error | uniform `{"error": {"code", "message"}}` envelope |
| Slugs on targets | `GET /targets/{idOrSlug}` — clean deep links / App Links |
| Coordinates stored, with a partial geo index | `migrations/0004_businesses_targets.sql:79` |

The last two matter: deep links work today, and the data for "near me" is already
in the table — only the query is missing.

## 4. Backend gaps that block the app

Verification delivery (§4.1) is resolved. Push (§4.2) is the one remaining
launch blocker; the §4.3 gaps each block a specific feature.

### 4.1 Verification delivery — decided: email only for v1

**Status: implemented.** Verification codes and password resets now deliver
over SMTP (`internal/auth/email.go`), enabled by setting `SMTP_HOST`. Email was
already the preferred channel in `internal/auth/service.go`, and the schema has
always allowed email-only accounts, so this needed one `Provider`
implementation rather than an auth redesign.

**Phone/SMS is deliberately deferred**, on a zero-budget constraint. Phone
verification reports `503 verification_unavailable` rather than pretending to
send. The cost of that choice is bounded by two facts:

- **Reading never requires auth**, so email-only gates writers, not the
  majority-reader traffic in §1.
- **Writing a review does not require a verified contact.** The `verified`
  concept on reviews (`internal/reviews/models.go`) is evidence-based —
  `receipt_submitted`, `location_verified` — and is unrelated to account
  verification.

What email delivery actually buys is **password reset**: without it, a user who
forgets their password is locked out permanently. That, not signup, is why it
had to exist.

Phone registration stays in the schema and the API, unverified. Adding SMS or
Telegram later is a second `Provider` implementation plus config — not a
migration, and not a re-auth of existing users.

**Telegram Gateway is the researched candidate** when there is budget: ~$0.01
per delivered code, automatic refunds for undelivered ones, and a free
`checkSendAbility` check, against a market where Telegram is the dominant
channel. Its limit is that the recipient must be a Telegram user who shared
that number, so it still needs an SMS fallback for full coverage.

### 4.2 The remaining blocker

**No push adapter and no device-token registration.**
The outbox, retry/backoff, and `Provider` interface are all built; there is no
FCM implementation and no endpoint for a client to register its device token.
Without this the activity inbox is pull-only and the app loses its main
retention mechanism.

### 4.3 Feature gaps

**1. No geo/radius endpoint.** Coordinates and an index exist; there is no
`nearby` query. "Near me" is the app's strongest native differentiator and cannot
be built today. Needs `GET /targets/nearby?lat=&lng=&radius=&category=`.
`frontend-handoff.md` §3 already lists map view as intentionally deferred — for
the app it should be un-deferred.

**2. ~~No image derivatives.~~ Done.** Finalizing a photo now also stores a
thumbnail bounded to 480 px on its longest edge, surfaced as `thumb_url` on
review media. Measured on a 1600x1200 upload: 114 KB full versus 15 KB
thumbnail. Additional sizes and WebP remain deferred.

**3. No minimum-version / force-upgrade endpoint.** Needed *before* the first
store release, not after — without it a broken client can never be retired. Cheap
to add: a version check the app calls at startup.

### 4.4 Worth researching, not blocking

- **No social login.** Email/phone + password only. Telegram-based auth may fit
  this market better than Google sign-in — worth a look, not a v1 commitment.
- **Search returns a single ranked page (max 50), no cursor.** Fine for a search
  screen; do not build infinite scroll against it. Revisit only if users hit the
  ceiling.

## 5. Offline behavior

The backend already makes this safe — the client just has to use it.

**Queue and retry:**
- Review submissions are queued locally with a client-generated `Idempotency-Key`
  and retried until accepted. Replays return the stored response, so a retry after
  an ambiguous timeout cannot double-post.
- Helpful votes queue as idempotent `PUT`/`DELETE` — last write wins, no
  reconciliation needed.
- Photo uploads queue as a two-phase job (presign → upload → finalize). The
  presign expires in 5 minutes, so a queued upload must **re-request the ticket**
  rather than replaying a stale one.

**Cache:**
- Target profiles and review lists: stale-while-revalidate, with an honest
  "saved N hours ago" banner as the handoff doc specifies.
- Aggregates and Reality Check: network-first, never served stale without a label —
  these are trust-critical numbers.

**Known server-side constraints the client must surface, not discover:**
- One review per user per target per 30 days → `409 cooldown_active` with
  `existing_review_id`. Check before opening the compose flow; offer "update your
  review" instead of failing at submit time.
- Max 5 reviews per user per 24h → `429`. Same reasoning.
- `expectation_match` is only accepted when `discovery_source` is a social
  platform. Drive this from the form, not from a rejected submit.

## 6. Auth on mobile

- **Email-first for v1**, per §4.1. Phone may be collected but cannot be
  verified yet, so it must not gate anything in the client.
- Store the refresh token in Android Keystore-backed encrypted storage, never in
  plain `SharedPreferences`.
- **Refresh rotation is single-use with family revocation.** Two concurrent
  refreshes will revoke the whole session and log the user out. The client must
  serialize refresh through a single-flight mutex — this is the most likely
  source of mystery logouts if handled carelessly.
- Surface `/auth/sessions` as a "your devices" screen; it is already built.
- Never gate reading behind auth — all read endpoints are public, and the handoff
  doc is emphatic about this.

## 7. Performance and data budgets

Adapting `frontend-handoff.md` §5 to an app:

- **APK download ≤ 25 MB**, and treat this as a hard product constraint — it is a
  real cost to the user, paid before they see any value. Enable ABI splits and
  R8/resource shrinking from the first release build.
- **Per-screen API payload ≤ 30 KB** — matches the existing target-detail budget.
- Review lists paginate at the backend default of 20.
- Client-side image compression before upload (max ~1600px, ~0.7 quality) — saves
  the *uploader's* data, which matters as much as the reader's.
- Ship a **data-saver mode**: text-only review lists, photos on demand.
- Cold start to interactive < 3s on a low-RAM device.

## 8. Stack recommendation

**React Native + Expo, Android-first.**

- Not Flutter: larger APK, and §7 makes download size a product constraint.
- Not native Kotlin: too slow for a small team, and it forecloses iOS entirely.
- **Generate the API client from `api/openapi.yaml`** rather than hand-writing it.
  The contract is complete and typed; hand-writing 76 endpoints is wasted work and
  will drift.
- The thin web layer should be server-rendered per `frontend-handoff.md` §4 —
  it serves exactly the low-bandwidth drive-by reader that doc was written for.

## 9. Phasing

**Phase 0 — unblock (backend). Complete.** Verification delivery (email, §4.1),
push (FCM adapter + device-token registration), `/targets/nearby`, image
thumbnails, and the version gate are all implemented. Push needs a Firebase
project and credentials before it delivers anything.

**Phase 1 — app MVP.** Auth + OTP, discover, search, target profile, review
submission with offline queue, my reviews, push. Ship to internal testing.

**Phase 2 — depth.** Near me, activity inbox, owner response flow, Reality Check,
share cards, evidence submission.

**Phase 3 — launch readiness.** Amharic localization hardened on real devices,
data-saver mode, thin web layer, staged Play Store rollout.

## 10. Open questions

Carried from `frontend-handoff.md` §7 plus mobile-specific ones. All need local
research or on-device testing, not a decision from this document:

- **Noto Sans Ethiopic on the cheap-Android fleet** — flagged as unverified in the
  handoff doc and still unverified. Test in week one, not at the end; it affects
  every screen.
- **Android TalkBack Amharic voice quality** — same caveat, same urgency.
- **Email reach in this market** — what share of the target audience has and
  checks email. This is the real risk created by the §4.1 decision, and it is
  measurable only after launch.
- **Which SMS gateway**, if phone verification is funded later. Local providers
  with direct Ethio Telecom routes are the candidates; sender-ID registration
  with the ECA has bureaucratic lead time and should start before it is needed.
- **The bilingual email copy** in `internal/auth/email.go` is a first draft and
  needs a native Amharic review — it is trust-critical text.
- **Ethiopian 12-hour day clock** for opening hours — unresolved in the handoff
  doc, still an open design question needing local user research.
- **Play Store reach** — whether the target audience installs from Play at all, or
  whether APK sideloading via Telegram is the real distribution channel. This
  materially affects update strategy and makes §4.3 item 3 more important.
- **Telegram-based auth** as an alternative to password login in this market.
