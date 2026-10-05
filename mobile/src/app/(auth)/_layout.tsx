import { Stack } from 'expo-router';
import { useEffect } from 'react';

import { clearPendingIntent } from '@/auth/pending-intent';

export default function AuthLayout() {
  // Leaving sign-in without signing in (back, "Continue browsing") abandons
  // what the user was trying to do, so it mustn't fire at a later sign-in.
  // Deferred a tick: on a successful sign-in this screen unmounts in the same
  // commit that PendingIntentRunner takes the intent (synchronously), so by
  // the time this runs there's nothing left to clear.
  useEffect(() => () => void setTimeout(clearPendingIntent, 0), []);

  return (
    <Stack screenOptions={{ headerShown: false }}>
      <Stack.Screen name="sign-in" />
      <Stack.Screen name="register" />
      <Stack.Screen name="forgot-password" />
      <Stack.Screen name="reset-password" />
    </Stack>
  );
}
