import { router } from 'expo-router';
import { ActivityIndicator, StyleSheet, View } from 'react-native';

import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { DiscoverSection } from './discover-section';
import { useNearbyTargets } from './queries';
import { TargetCard } from './target-card';
import { useDeviceLocation } from './use-device-location';

const PROMPT_COPY: Record<'idle' | 'denied' | 'error', { body: string; button: string }> = {
  idle: { body: 'See what’s good nearby.', button: 'Show places near you' },
  denied: { body: 'Location access was denied — turn it on to see what’s nearby.', button: 'Try again' },
  error: { body: "Couldn't get your location.", button: 'Try again' },
};

export function NearMeSection() {
  const { state, request } = useDeviceLocation();

  if (state.status === 'idle' || state.status === 'denied' || state.status === 'error') {
    const copy = PROMPT_COPY[state.status];
    return (
      <View style={styles.prompt}>
        <ThemedText type="smallBold" style={styles.title}>
          Near me
        </ThemedText>
        <ThemedText type="small" themeColor="textSecondary" style={styles.body}>
          {copy.body}
        </ThemedText>
        <Button title={copy.button} variant="secondary" onPress={request} />
      </View>
    );
  }

  if (state.status === 'loading') {
    return (
      <View style={styles.prompt}>
        <ThemedText type="smallBold" style={styles.title}>
          Near me
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
