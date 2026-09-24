import { Stack } from 'expo-router';
import { useTranslation } from 'react-i18next';

import { OfflineQueueProcessor } from '@/features/reviews/offline-queue-processor';
import { PushRegistration } from '@/features/push/push-registration';

export default function AppLayout() {
  const { t } = useTranslation();
  return (
    <>
      {/* Only mounted while signed in — a queued job or a device
          registration both belong to whoever is signed in when they
          happen. */}
      <OfflineQueueProcessor />
      <PushRegistration />
      <Stack>
        <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
        <Stack.Screen name="search" options={{ title: t('nav.search') }} />
        <Stack.Screen name="target/[idOrSlug]/index" options={{ title: '' }} />
        <Stack.Screen name="target/[idOrSlug]/review" options={{ title: t('nav.writeReview') }} />
        <Stack.Screen name="verify" options={{ title: t('nav.verifyEmail') }} />
        <Stack.Screen name="sessions" options={{ title: t('nav.yourDevices') }} />
      </Stack>
    </>
  );
}
