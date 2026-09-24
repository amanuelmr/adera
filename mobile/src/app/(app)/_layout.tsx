import { Stack } from 'expo-router';

import { OfflineQueueProcessor } from '@/features/reviews/offline-queue-processor';
import { PushRegistration } from '@/features/push/push-registration';

export default function AppLayout() {
  return (
    <>
      {/* Only mounted while signed in — a queued job or a device
          registration both belong to whoever is signed in when they
          happen. */}
      <OfflineQueueProcessor />
      <PushRegistration />
      <Stack>
        <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
        <Stack.Screen name="search" options={{ title: 'Search' }} />
        <Stack.Screen name="target/[idOrSlug]/index" options={{ title: '' }} />
        <Stack.Screen name="target/[idOrSlug]/review" options={{ title: 'Write a review' }} />
        <Stack.Screen name="verify" options={{ title: 'Verify email' }} />
        <Stack.Screen name="sessions" options={{ title: 'Your devices' }} />
      </Stack>
    </>
  );
}
