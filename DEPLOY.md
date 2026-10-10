# Deploying Adera

**Live:** https://adera.amanuel.work — also reachable at
`adera-api-9cfb.onrender.com`.

Runs the API on a free tier with no credit card at any step.

| Piece | Service | Notes |
| --- | --- | --- |
| Go API | Render (free web service, Frankfurt) | Builds the existing `Dockerfile` |
| Postgres | Neon (free tier, Frankfurt) | Auto-suspends, resumes in <1s |
| Object storage | none | `STORAGE_ENABLED=false`; only evidence uploads need it |

Total cost: nothing. The one compromise is Render's idle spin-down — see
[Keeping it warm](#6-keep-it-warm).

Two things about the current setup, before you change anything:

- Render is connected to the **public repo URL**, not through the GitHub App,
  so there are no webhooks and **pushes do not auto-deploy**. Use *Manual
  Deploy* in the dashboard, or connect GitHub under the service's Settings to
  get push-to-deploy.
- Free custom domains are capped at **2 per workspace**, and
  `adera.amanuel.work` uses one of them.

---

## 1. Create the database

Sign up at [neon.tech](https://neon.tech) (GitHub login, no card) and create a
project. Copy the **pooled** connection string; it looks like:

```
postgresql://USER:PASSWORD@ep-xxx-pooler.eu-central-1.aws.neon.tech/neondb?sslmode=require
```

Use the pooled URL, not the direct one — Render's free instance opens and drops
connections as it sleeps and wakes, and the pooler absorbs that.

## 2. Apply migrations

Migrations are embedded in the binary, so nothing extra is needed locally.
Render's free plan has no pre-deploy hook, so run this once from your machine:

```sh
DATABASE_URL='<neon pooled url>' go run ./cmd/api migrate
```

Use the **direct** (non-pooled) URL here, not the pooled one. Neon's pooler is
transaction-mode PgBouncer, which `CREATE EXTENSION` in `0001` and the
migrator's advisory locks do not reliably survive. The running service uses the
pooled URL; migrations use the direct one.

**Do not run `seed` against this database.** It refuses to run when
`APP_ENV=production` by design, and its admin password falls back to the
documented development value — running it would put a known-password admin
account on a public instance. The live API is not empty regardless: categories
and cities load through `0010_reference_data`, which is a migration, not seed
data.

Re-run `migrate` after any future migration is added.

## 3. Deploy the API

1. Push this branch to GitHub.
2. In Render, **New → Blueprint**, pick the `adera` repo. It reads `render.yaml`.
3. Approve the plan. The build runs the multi-stage `Dockerfile`; the runtime
   image is distroless, so expect a small image and a fast boot.

## 4. Set the secrets

In the service's **Environment** tab:

- `DATABASE_URL` — the Neon pooled string from step 1.
- `JWT_SECRET` — Render generates one. Config **rejects anything under 32
  bytes in production**, so check its length. If it's short, replace it:

  ```sh
  openssl rand -base64 48
  ```

- `SIGNAL_HASH_KEY` — also generated, also at least 32 bytes (same check, same
  fix). It keys the hashes of review fraud signals; set it once and leave it,
  since a new key stops new reviews matching older signals. Clearing it turns
  signal recording off.

Changing any of these triggers a redeploy.

### Optional: email, push, and the version gate

These are declared in `render.yaml` but left empty; each feature stays off
until its variables are set.

- **Email** (verification codes, password reset): `SMTP_HOST`,
  `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM_ADDRESS`. Until these are set,
  verification returns `503 verification_unavailable` and users who forget
  their password cannot reset it.
- **Push**: in Firebase, create a service-account key for the project the
  Android app is registered in. Upload it under **Environment → Secret
  Files** (e.g. as `fcm-service-account.json`), then set
  `FCM_CREDENTIALS_FILE=/etc/secrets/fcm-service-account.json`. Set the
  variable only after the file exists — the API won't start if it points at
  a missing file.
- **Version gate**: `APP_ANDROID_MIN_VERSION`, `APP_ANDROID_LATEST_VERSION`,
  `APP_ANDROID_STORE_URL`. Versions are plain semver (`1.2.0`, not `v1.2.0`).
  `APP_ANDROID_STORE_URL` also turns on the "Get the app" banner on the web
  pages.
- **App Links**: `ANDROID_CERT_SHA256` — the signing certificate fingerprint
  from EAS (`mobile/RELEASE.md`). Until it's set,
  `/.well-known/assetlinks.json` is a 404 and shared links open in the
  browser rather than the app.

### Web pages

The same service serves the thin web layer: `/t/{slug}` (a place and its
reviews), `/r/{id}` (one review), and `/trust` (how ratings and moderation
work), in English and Amharic. `BASE_URL` must be the public origin
(`https://adera.amanuel.work`) so canonical and share-preview links are right.

## 5. Point a domain at it

Render issues `adera-api.onrender.com` and serves TLS on it. For a custom
subdomain, add `adera.amanuel.work` under **Settings → Custom Domains**, then
at Porkbun (DNS Management for `amanuel.work`) add:

| Type | Host | Answer |
| --- | --- | --- |
| CNAME | `adera` | the target Render shows |

Leave "Do not delete existing records" checked when submitting. Certificates
issue automatically once DNS resolves.

## 6. Keep it warm

Free instances sleep after 15 minutes idle and take 30–50s to wake — bad for a
link someone clicks once.

This is handled by `.github/workflows/keepalive.yml`, which pings `/health`
every 10 minutes. No third-party uptime service and no extra account: public
repos get unlimited Actions minutes.

It deliberately runs **05:00–21:00 UTC only**, not around the clock. The free
allowance is 750 instance-hours per month **across the whole workspace**, and
staying awake 24/7 burns ~730 of them — 97% of everything, spent on one
service. The 16-hour window costs ~490 hours, still covers business hours in
Europe, Africa and the US, and leaves room for another free service later.

To go always-warm, change the cron to `*/10 * * * *` — but check the
free-hours gauge under Billing → Included Usage first.

One gotcha: GitHub disables scheduled workflows after **60 days without repo
activity**. If the API starts sleeping again, re-enable it under the Actions
tab.

## Verifying

```sh
curl -s https://adera.amanuel.work/health     # liveness
curl -s https://adera.amanuel.work/ready      # checks the DB connection
curl -s https://adera.amanuel.work/metrics    # Prometheus metrics
```

`/ready` is the one that matters: it returns unhealthy when `DATABASE_URL` is
wrong or Neon is unreachable, which is how a bad deploy shows up.

## Notes

- **Argon2 memory.** Password hashing needs ≥19 MiB per hash (OWASP minimum,
  enforced in config). Fine on a 512 MB instance for demo traffic, but many
  concurrent logins would pressure it.
- **Storage.** To enable uploads later, set `STORAGE_ENABLED=true` and supply
  `STORAGE_ACCESS_KEY` / `STORAGE_SECRET_KEY` plus bucket settings — config
  refuses to start otherwise. Any S3-compatible host works; Supabase Storage is
  already on your account.
- **Free tiers move.** Both Render's and Neon's terms have changed before.
  Re-check before relying on this for anything that matters.
