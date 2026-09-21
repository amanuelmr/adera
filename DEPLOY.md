# Deploying Adera

Runs the API live on a free tier, with no credit card at any step.

| Piece | Service | Notes |
| --- | --- | --- |
| Go API | Render (free web service) | Builds the existing `Dockerfile` |
| Postgres | Neon (free tier) | Auto-suspends, resumes in <1s |
| Object storage | none | `STORAGE_ENABLED=false`; only evidence uploads need it |

Total cost: nothing. The one compromise is Render's idle spin-down — see
[Keeping it warm](#6-keep-it-warm).

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

Optionally load reference data (categories, cities) plus a seed admin, so the
live API has something to return:

```sh
DATABASE_URL='<neon pooled url>' \
SEED_ADMIN_EMAIL='you@example.com' \
SEED_ADMIN_PASSWORD='<a real password>' \
  go run ./cmd/api seed
```

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

Changing either triggers a redeploy.

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
link someone clicks once. A free cron pinger fixes it:

- [cron-job.org](https://cron-job.org) or UptimeRobot, no card
- Hit `https://<your-domain>/health` every **10 minutes**

Render's free allowance is 750 instance-hours/month; staying awake costs ~730,
so this fits — but it leaves little headroom. Don't ping more than one free
service from the same account.

## Verifying

```sh
curl -s https://<your-domain>/health     # liveness
curl -s https://<your-domain>/ready      # checks the DB connection
curl -s https://<your-domain>/metrics    # Prometheus metrics
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
