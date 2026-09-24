import { ApiError } from '@/api/client';

/**
 * Maps a stable API error `code` to user-facing copy. Centralized because
 * every auth screen needs it, and because the API's codes are the
 * localization key going forward (docs/frontend-handoff.md §7) — swapping
 * these strings for i18next lookups later is a change to this one file.
 */
export function friendlyAuthError(err: unknown): string {
  if (err instanceof ApiError) {
    switch (err.code) {
      case 'invalid_credentials':
        return 'Incorrect email/phone or password.';
      case 'account_suspended':
        return 'This account has been suspended.';
      case 'rate_limited':
        return 'Too many attempts — try again in a few minutes.';
      case 'verification_unavailable':
        return "Verification isn't available right now, but you can still use the app.";
      case 'validation_failed':
        return Object.values(err.details ?? {})[0] ?? err.message;
      default:
        return err.message;
    }
  }
  return 'Something went wrong. Check your connection and try again.';
}
