import { useEffect, useRef } from 'react';

import { useAuth } from './context';
import { clearPendingIntent, takePendingIntent } from './pending-intent';

/**
 * Carries out what the user was doing when sign-in interrupted them, once
 * they're signed in. Mounted for the app's lifetime (in the (app) layout).
 */
export function PendingIntentRunner() {
  const { status } = useAuth();
  const previous = useRef(status);

  useEffect(() => {
    const was = previous.current;
    previous.current = status;
    if (status === 'signedIn' && was === 'signedOut') {
      const run = takePendingIntent();
      // After this render: by then the sign-in modal is being dismissed (its
      // guard just turned false), so the action navigates from the screen
      // the user started on.
      if (run) setTimeout(run, 0);
    } else if (status === 'signedOut') {
      clearPendingIntent();
    }
  }, [status]);

  return null;
}
