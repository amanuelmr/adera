// What a signed-out user was trying to do when we sent them to sign in, so
// it can carry on after they sign in or register instead of dropping them
// back where they started. In memory only: it belongs to this session.

/** Why sign-in opened — shown on the sign-in screen. Keys under auth.reason.* */
export const SIGN_IN_REASONS = ['writeReview', 'vote', 'report', 'addPlace'] as const;
export type SignInReason = (typeof SIGN_IN_REASONS)[number];

// Long enough to type a password and verify an email address; short enough
// that a forgotten intent doesn't fire at some surprising later sign-in.
export const INTENT_TTL_MS = 10 * 60 * 1000;

let pending: { run: () => void; at: number } | undefined;

export function setPendingIntent(run: () => void, now: number = Date.now()): void {
  pending = { run, at: now };
}

/** Returns the pending action if it's still fresh, and forgets it either way. */
export function takePendingIntent(now: number = Date.now()): (() => void) | undefined {
  const intent = pending;
  pending = undefined;
  return intent && now - intent.at <= INTENT_TTL_MS ? intent.run : undefined;
}

export function clearPendingIntent(): void {
  pending = undefined;
}

export function isSignInReason(value: unknown): value is SignInReason {
  return typeof value === 'string' && (SIGN_IN_REASONS as readonly string[]).includes(value);
}
