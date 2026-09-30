import * as Notifications from 'expo-notifications';
import { useEffect } from 'react';

import { setupPushNotifications } from './register';

Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldPlaySound: false,
    shouldSetBadge: false,
    shouldShowBanner: true,
    shouldShowList: true,
  }),
});

/**
 * Registers this device for push and keeps the token current. Mounted only
 * while signed in (see (app)/_layout.tsx), same reasoning as the offline
 * queue processor: a device registration belongs to whoever is signed in
 * when it happens.
 */
export function PushRegistration() {
  useEffect(() => {
    let unsubscribeRotation: (() => void) | undefined;
    let unmounted = false;
    setupPushNotifications()
      .then((unsubscribe) => {
        // Unmounted (signed out) before setup finished — don't leak the listener.
        if (unmounted) unsubscribe?.();
        else unsubscribeRotation = unsubscribe;
      })
      // Without google-services.json (or with no Play Services) the native
      // token call rejects; push is simply unavailable, not an app error.
      .catch((err) => console.warn('Push notifications unavailable', err));

    // Tap handling beyond opening the app is Phase 2 (activity inbox,
    // docs/mobile-plan.md §9) — there's no in-app destination to route to yet.
    const responseSubscription = Notifications.addNotificationResponseReceivedListener(() => {});

    return () => {
      unmounted = true;
      unsubscribeRotation?.();
      responseSubscription.remove();
    };
  }, []);

  return null;
}
