import { Stack } from 'expo-router';
import { useTranslation } from 'react-i18next';

import { useAuth } from '@/auth/context';
import { OfflineQueueProcessor } from '@/features/reviews/offline-queue-processor';
import { PushRegistration } from '@/features/push/push-registration';

export default function AppLayout() {
  const { t } = useTranslation();
  const { status } = useAuth();
  const signedIn = status === 'signedIn';
  return (
    <>
      {/* Only mounted while signed in — a queued job or a device
          registration both belong to whoever is signed in when they
          happen. */}
      {signedIn ? (
        <>
          <OfflineQueueProcessor />
          <PushRegistration />
        </>
      ) : null}
      <Stack>
        <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
        <Stack.Screen name="search" options={{ title: t('nav.search') }} />
        <Stack.Screen name="target/[idOrSlug]/index" options={{ title: '' }} />
        <Stack.Screen name="category/[id]" options={{ title: '' }} />
        <Stack.Screen name="t/[slug]" options={{ headerShown: false }} />
        <Stack.Screen name="r/[id]" options={{ title: '' }} />
        <Stack.Protected guard={signedIn}>
          <Stack.Screen name="target/[idOrSlug]/review" options={{ title: t('nav.writeReview') }} />
          <Stack.Screen name="review/[id]/evidence" options={{ title: t('nav.addEvidence') }} />
          <Stack.Screen name="review/[id]/report" options={{ title: t('nav.reportReview') }} />
          <Stack.Screen name="businesses/index" options={{ title: t('nav.yourBusinesses') }} />
          <Stack.Screen name="businesses/[id]/index" options={{ title: t('nav.locations') }} />
          <Stack.Screen name="businesses/[id]/targets/[targetId]" options={{ title: '' }} />
          <Stack.Screen name="verify" options={{ title: t('nav.verifyEmail') }} />
          <Stack.Screen name="sessions" options={{ title: t('nav.yourDevices') }} />
        </Stack.Protected>
      </Stack>
    </>
  );
}
