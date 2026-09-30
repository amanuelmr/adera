import { router } from 'expo-router';
import * as Notifications from 'expo-notifications';
import { useEffect } from 'react';

import { ACTIVITY_KEY } from '@/features/activity/queries';
import { queryClient } from '@/lib/query-client';
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

    // Tapping a push opens the inbox, where the item can be read and followed.
    const responseSubscription = Notifications.addNotificationResponseReceivedListener(() => {
      router.push('/activity');
    });
    // A push arriving while the app is open means the inbox and badge are stale.
    const receivedSubscription = Notifications.addNotificationReceivedListener(() => {
      queryClient.invalidateQueries({ queryKey: ACTIVITY_KEY });
    });

    return () => {
      unmounted = true;
      unsubscribeRotation?.();
      responseSubscription.remove();
      receivedSubscription.remove();
    };
  }, []);

  return null;
}
