import { router } from 'expo-router';
import { useCallback } from 'react';

import { useAuth } from './context';
import { setPendingIntent, type SignInReason } from './pending-intent';

/**
 * Runs a write action when signed in. Otherwise opens sign-in, saying why,
 * and remembers the action so it carries on once the user signs in or
 * registers (see PendingIntentRunner).
 */
export function useRequireSignIn() {
  const { status } = useAuth();
  return useCallback(
    (action: () => void, reason: SignInReason) => {
      if (status === 'signedIn') {
        action();
        return;
      }
      setPendingIntent(action);
      router.push({ pathname: '/sign-in', params: { reason } });
    },
    [status]
  );
}
