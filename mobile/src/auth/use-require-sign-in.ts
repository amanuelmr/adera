import { router } from 'expo-router';
import { useCallback } from 'react';

import { useAuth } from './context';

/** Runs a write action when signed in; otherwise opens sign-in over the current screen. */
export function useRequireSignIn() {
  const { status } = useAuth();
  return useCallback(
    (action: () => void) => {
      if (status === 'signedIn') action();
      else router.push('/sign-in');
    },
    [status]
  );
}
