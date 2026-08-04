# Authorization Matrix

Two layers everywhere:

- **Function-level** — route middleware: `RequireAuth`, `RequireRole(moderator)`,
  `RequireRole(admin)`. Roles ride in the access token (≤15-min staleness);
  the `admin` role implies `moderator`.
- **Object-level** — inside services, against the database: authorship
  (`review.user_id = caller`), business membership (`business_members`),
  session ownership. Object-level failures on resources the caller shouldn't
  know exist return 404; on resources they may know exist, 403.

Roles: `customer` (default), `business_owner` (granted only by claim
approval; removed when the last membership is revoked), `moderator`, `admin`.

Legend: ✅ allowed · Ⓞ allowed with object-level check · ❌ forbidden

| Endpoint | Anonymous | Customer | Business owner | Moderator | Admin |
|---|---|---|---|---|---|
| **Auth** |
| POST /auth/register, /auth/login, /auth/refresh, /auth/password-reset/* | ✅ | ✅ | ✅ | ✅ | ✅ |
| POST /auth/logout, /auth/logout-all · GET /auth/sessions | ❌ | ✅ | ✅ | ✅ | ✅ |
| DELETE /auth/sessions/{id} | ❌ | Ⓞ own session | Ⓞ | Ⓞ | Ⓞ |
| POST /auth/verify/* | ❌ | ✅ own contact | ✅ | ✅ | ✅ |
| **Current user** |
| GET/PATCH /users/me · POST /users/me/password · DELETE /users/me | ❌ | ✅ self only | ✅ | ✅ | ✅ |
| GET /users/me/reviews · /users/me/reports | ❌ | ✅ own only | ✅ | ✅ | ✅ |
| GET/PUT /users/me/notifications* | ❌ | Ⓞ own inbox only | Ⓞ | Ⓞ | Ⓞ |
| **Catalog (public reads)** |
| GET /categories*, /locations/*, /targets*, /targets/{id}/stats, /reality-check, /reviews/{id}, /targets/{id}/reviews, /search/targets | ✅ | ✅ | ✅ | ✅ | ✅ |
| Unpublished target/review visibility | ❌ 404 | Ⓞ author of the review | Ⓞ | ✅ | ✅ |
| **Targets** |
| POST /targets | ❌ | ✅ → pending | ✅ → pending | ✅ → published | ✅ → published |
| PATCH /targets/{id} | ❌ | ❌ | Ⓞ member of owning business | ✅ | ✅ |
| POST /targets/{id}/edit-suggestions | ❌ | ✅ | ✅ | ✅ | ✅ |
| **Reviews** |
| POST /reviews | ❌ | ✅ (cooldown, daily cap) | ✅ | ✅ | ✅ |
| PUT/DELETE /reviews/{id} | ❌ | Ⓞ author only | Ⓞ author only | Ⓞ author only¹ | Ⓞ¹ |
| PUT/DELETE /reviews/{id}/helpful | ❌ | ✅ not own review | ✅ | ✅ | ✅ |
| POST /reviews/{id}/media, /evidence · finalize | ❌ | Ⓞ author only | Ⓞ | Ⓞ | Ⓞ |
| GET /reviews/{id}/evidence | ❌ | Ⓞ author only | Ⓞ author only | via moderation route | via moderation route |
| **Businesses & responses** |
| POST /businesses | ❌ | ✅ (grants no control) | ✅ | ✅ | ✅ |
| PATCH /businesses/{id} · GET /businesses/{id}/stats | ❌ | ❌ | Ⓞ member only | ✅ | ✅ |
| POST /reviews/{id}/response · PUT /responses/{id} | ❌ | ❌ | Ⓞ member of the review's business | ❌² | ❌² |
| **Claims** |
| POST /businesses/{id}/claims · GET /claims/mine | ❌ | ✅ | ✅ | ✅ | ✅ |
| POST /moderation/claims/{id}/decision, /revoke | ❌ | ❌ | ❌ | ✅ | ✅ |
| **Reports & moderation** |
| POST /reviews/{id}/reports · /targets/{id}/reports | ❌ | ✅ | ✅ | ✅ | ✅ |
| GET /moderation/reports*, /moderation/audit | ❌ | ❌ | ❌ | ✅ | ✅ |
| POST /moderation/reviews/{id}/decision · targets · evidence · notes · reports/{id}/resolve | ❌ | ❌ | ❌ | ✅ | ✅ |
| **Admin** |
| /admin/users/{id}/suspend, /reinstate, /roles | ❌ | ❌ | ❌ | ❌ | ✅ (not self) |
| /admin/categories*, /admin/criteria/{id} | ❌ | ❌ | ❌ | ❌ | ✅ |
| /admin/targets/{id}/merge | ❌ | ❌ | ❌ | ❌ | ✅ |
| **Ops** |
| GET /health, /ready, /metrics³ | ✅ | ✅ | ✅ | ✅ | ✅ |

¹ Moderators use `/moderation/reviews/{id}/decision`, never author-edit
endpoints — moderators cannot rewrite someone's words, only change status.
² Moderators/admins respond to reviews only if they are themselves members of
the business; roles alone don't speak for a business.
³ `/metrics` must be network-restricted in production deployments.

**Invariants enforced by tests** (`internal/app/platform_integration_test.go`
and friends): every protected endpoint 401s anonymously; every
moderator/admin endpoint 403s for customers; moderators cannot reach admin
endpoints; cross-user review edits 403; foreign sessions look nonexistent;
non-members can't read business stats or edit responses; strangers can't list
evidence or mutate another user's activity inbox; suspension is immediate
(all sessions revoked).
