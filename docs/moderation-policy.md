# Moderation Policy (Operational)

This is the platform-side policy the backend implements. The public-facing
Amharic/English version belongs to the Trust Center content (frontend phase).
Legal obligations feeding this policy are documented in
docs/research/security-and-legal-risks.md §7 and require professional review.

## Principles

1. **Publish first, moderate transparently.** Reviews publish immediately;
   they enter moderation via reports or moderator sampling. Target
   submissions from ordinary users start `pending` (directory integrity);
   moderator-created targets publish immediately.
2. **Reports never auto-hide content.** Only a moderator decision changes
   visibility — report-count triggers would let three angry clicks censor a
   truthful review.
3. **Nothing important is destroyed.** All moderation transitions are soft;
   review bodies, evidence records, and every decision are retained. The
   `moderation_actions` audit table is append-only, and no API operation
   deletes from it.
4. **Businesses cannot pay to hide criticism.** There is no payment surface,
   and no endpoint by which a business can delete, alter, or suppress a
   customer review. Businesses may respond publicly (one editable response
   per review, edits audited) and may report.
5. **24-hour SLA for hate speech / disinformation notifications** (Proclamation
   1185/2020 exposure): the moderation queue orders oldest-first by design.

## Review states

`pending → published → under_review → (rejected | hidden | removed) → restored (published)`

| State | Visible publicly | In aggregates | Who sets it |
|---|---|---|---|
| pending | no | no | system (reserved for future pre-moderation) |
| published | yes | yes | system on creation; moderator approve/restore |
| under_review | owner + moderators | no | moderator |
| hidden | owner + moderators | no | moderator (guideline violation, appealable) |
| rejected | owner + moderators | no | moderator (fake/spam determination) |
| removed | owner + moderators | no | review author ("remove from public view") or moderator (severe) |

Aggregates adjust in the same transaction as every state change.

## Report reasons

`spam`, `fake_experience`, `conflict_of_interest`, `harassment`,
`hate_speech`, `personal_information`, `unsupported_accusation`,
`irrelevant`, `duplicate`, `manipulated_evidence`.

One open report per user per subject (unique partial index). Reporters see
their reports' status at `GET /users/me/reports` (`open → in_review →
resolved | dismissed`).

## Anti-manipulation controls

- One review per user per target per **30 days** (update instead), max
  **5 reviews per user per 24h**, enforced in the create transaction.
- Rate limits: login/OTP 5/min per IP+identifier; review writes 10/min per
  account; reports 10/min; presigns 20/h; search 2 rps per IP.
- Idempotency keys prevent duplicate submissions from flaky connections.
- No self-votes; one helpful vote per user per review (primary key).
- Review-fraud gig economies exist locally (see market research §7); device
  fingerprinting and collusion heuristics are documented future work.

## Retention & deactivation

Account deactivation keeps published reviews (they describe real commerce and
removing them would enable reputation laundering); the user can log back in to
reactivate, and can individually remove their own reviews from public view at
any time. Full data-subject erasure workflows (Proclamation 1321/2024) need
counsel input and are tracked as an open compliance item.

## Moderator toolset (implemented)

Queue (`GET /moderation/reports`), report detail/resolve, review decisions
(approve/restore/under_review/hide/reject/remove), target decisions
(approve/hide/remove/restore), evidence accept/reject (drives verification),
claim approve/reject/revoke, free-form notes, per-subject audit trail
(`GET /moderation/audit`), account suspension/reinstatement (admin), duplicate
target merge (admin). Function-level authorization: `moderator` role (admins
inherit it); admin-only for suspension, category management, and merges.
