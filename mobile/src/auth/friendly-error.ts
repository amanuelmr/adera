import { ApiError } from '@/api/client';
import i18n from '@/lib/i18n';

/**
 * Maps a stable API error `code` to user-facing copy — the API's codes are
 * the localization key (docs/frontend-handoff.md §7), so this is the one
 * place that needs to change if a new code needs copy. Uses the i18next
 * singleton directly (not the useTranslation hook) since this is a plain
 * function called from event handlers, not a component.
 */
export function friendlyAuthError(err: unknown): string {
  if (err instanceof ApiError) {
    switch (err.code) {
      case 'invalid_credentials':
        return i18n.t('errors.invalidCredentials');
      case 'account_suspended':
        return i18n.t('errors.accountSuspended');
      case 'rate_limited':
        return i18n.t('errors.rateLimited');
      case 'verification_unavailable':
        return i18n.t('errors.verificationUnavailable');
      case 'validation_failed':
        return Object.values(err.details ?? {})[0] ?? err.message;
      default:
        return err.message;
    }
  }
  return i18n.t('errors.generic');
}
