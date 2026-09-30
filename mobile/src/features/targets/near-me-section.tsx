import { router } from 'expo-router';
import { useTranslation } from 'react-i18next';
import { ActivityIndicator, Linking, StyleSheet, View } from 'react-native';

import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { DiscoverSection } from './discover-section';
import { useNearbyTargets } from './queries';
import { TargetCard } from './target-card';
import { useDeviceLocation } from './use-device-location';

type PromptStatus = 'idle' | 'denied' | 'blocked' | 'error';

export function NearMeSection() {
  const { t } = useTranslation();
  const { state, request } = useDeviceLocation();

  if (state.status === 'idle' || state.status === 'denied' || state.status === 'blocked' || state.status === 'error') {
    const status: PromptStatus = state.status;
    return (
      <View style={styles.prompt}>
        <ThemedText type="smallBold" style={styles.title}>
          {t('nearMe.title')}
        </ThemedText>
        <ThemedText type="small" themeColor="textSecondary" style={styles.body}>
          {t(`nearMe.${status}Body`)}
        </ThemedText>
        {status === 'blocked' ? (
          // The system won't show the prompt again; only Settings can grant it.
          <Button title={t('nearMe.openSettings')} variant="secondary" onPress={() => Linking.openSettings()} />
        ) : (
          <Button title={t(status === 'idle' ? 'nearMe.show' : 'common.retry')} variant="secondary" onPress={request} />
        )}
      </View>
    );
  }

  if (state.status === 'loading') {
    return (
      <View style={styles.prompt}>
        <ThemedText type="smallBold" style={styles.title}>
          {t('nearMe.title')}
        </ThemedText>
        <ActivityIndicator />
      </View>
    );
  }

  return <NearMeResults latitude={state.latitude} longitude={state.longitude} />;
}

function NearMeResults({ latitude, longitude }: { latitude: number; longitude: number }) {
  const nearby = useNearbyTargets({ latitude, longitude });

  return (
    <DiscoverSection
      title="Near me"
      query={nearby}
      emptyLabel="Nothing nearby yet."
      keyExtractor={(target) => target.id ?? target.slug ?? ''}
      renderItem={(target) => (
        <TargetCard
          name={target.name ?? 'Unnamed'}
          averageRating={target.average_rating ?? null}
          reviewCount={target.review_count ?? 0}
          badge={target.distance_km != null ? `${target.distance_km.toFixed(1)} km away` : undefined}
          onPress={() => router.push(`/target/${target.slug ?? target.id}`)}
        />
      )}
    />
  );
}

const styles = StyleSheet.create({
  prompt: {
    paddingHorizontal: Spacing.four,
    gap: Spacing.two,
    alignItems: 'flex-start',
  },
  title: {
    marginBottom: Spacing.one,
  },
  body: {
    marginBottom: Spacing.one,
  },
});
