/**
 * A random key, not a cryptographic one — Idempotency-Key only needs to be
 * unique per submission attempt (spec: <=200 chars, any format), reused
 * across retries of the *same* attempt so a retry after a dropped response
 * can't double-post.
 */
export function generateIdempotencyKey(): string {
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}-${Math.random().toString(36).slice(2)}`;
}
