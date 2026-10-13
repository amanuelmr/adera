# Data breach response

What to do when personal data may have been exposed: leaked database or
storage credentials, a dump of the database or the private bucket, a bug that
showed one person's data to another, or a lost device with admin access.

Ethiopia's Personal Data Protection Proclamation (No. 1321/2024) expects the
authority to be told within **72 hours** of becoming aware of a breach that
risks people's rights, and the affected people without undue delay. Counsel
should confirm the exact duties; act on this plan in the meantime. The clock
starts when we find out, not when we finish investigating.

## 1. Contain (first hours)

- **Rotate what may have leaked**, on Render (Environment tab), then redeploy:
  - `DATABASE_URL`: reset the role's password in Neon first.
  - `JWT_SECRET`: signs everyone out at once. It also keys one-time codes, so
    any code in flight stops working.
  - `SIGNAL_HASH_KEY`: only if the database leaked too. A new key stops new
    reviews matching older fraud signals.
  - `METRICS_TOKEN`, `SMTP_PASSWORD`, storage keys, and the Firebase
    service-account key (revoke it in Google Cloud).
- **Sign suspect accounts out**: `POST /api/v1/admin/users/{id}/suspend`
  revokes their sessions. For a leaked admin, suspend the admin.
- **Close the hole**: revert or hotfix the deploy. To stop the app while you
  do, set `APP_ANDROID_MIN_VERSION` above every released build (the app then
  shows only "update required"); suspend the Render service to stop the web
  pages and API too.
- **Keep evidence**: copy the Render and Neon logs for the period before they
  rotate out, and note who did what and when, starting now.

## 2. Assess (within 24 hours)

Write down:

- what was exposed — use the data inventory in the privacy notice
  (`/privacy`): profile data, reviews, receipts, push tokens, fraud-signal
  hashes, sessions;
- whose — query the affected user IDs and keep the list;
- for how long, and whether anyone actually accessed it (request logs record
  method, path and status; storage access logs if enabled);
- the risk to people: email and phone are the sensitive fields for most users;
  receipts can show names, purchases and locations; password hashes are
  Argon2id and refresh tokens are stored hashed, so a database dump doesn't
  give away logins directly.

## 3. Notify (within 72 hours of finding out)

- **The authority**: what happened, what data and how many people, likely
  consequences, what we did, and a contact. If something is still unknown,
  notify anyway and follow up.
- **Affected people**, in Amharic and English: what happened to their data,
  what we did, and what they should do (e.g. watch for phishing to the leaked
  email address). Send by email, plus an in-app notification.
- **Processors**: tell Render, Neon or Google if the breach involves them.

## 4. Afterwards

- Write a short review: cause, timeline, what worked, and what changes
  prevent a repeat. Add a regression test for any code bug.
- Keep the record (this plan's notes, the notifications sent, the affected
  list) with the other compliance records for at least as long as counsel
  advises.

## Contacts

Keep this current, outside the repository if it holds personal details:

- Who decides and who notifies (owner of the service).
- Counsel.
- The data protection authority's breach reporting channel.
