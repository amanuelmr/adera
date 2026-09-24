import { Stack } from 'expo-router';

import { OfflineQueueProcessor } from '@/features/reviews/offline-queue-processor';

export default function AppLayout() {
  return (
    <>
      {/* Only mounted while signed in — queued jobs are tied to the
          signed-in user, and retrying them signed out would just 401. */}
      <OfflineQueueProcessor />
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
