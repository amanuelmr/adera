import { router } from 'expo-router';
import { useTranslation } from 'react-i18next';
import { ActivityIndicator, Linking, StyleSheet, View } from 'react-native';

import { Button } from '@/components/button';
import { Card } from '@/components/card';
import { Icon } from '@/components/icon';
import { ThemedText } from '@/components/themed-text';
import { Radius, Spacing } from '@/constants/theme';
import { AddPlaceLink } from '@/features/places/add-place-link';
import { useTheme } from '@/hooks/use-theme';
import { DiscoverSection } from './discover-section';
import { useNearbyTargets } from './queries';
import { TargetCard, TargetCardSkeletons } from './target-card';
import { useDeviceLocation } from './use-device-location';

type PromptStatus = 'idle' | 'denied' | 'blocked' | 'error';

export function NearMeSection() {
  const { t } = useTranslation();
  const theme = useTheme();
  const { state, request } = useDeviceLocation();

  if (state.status === 'ready') {
    return <NearMeResults latitude={state.latitude} longitude={state.longitude} />;
  }

  const status: PromptStatus | 'loading' = state.status;
  return (
    <View style={styles.inset}>
      <Card style={styles.prompt}>
        <View style={[styles.iconWrap, { backgroundColor: theme.primaryContainer }]}>
          <Icon name="map-pin" size={22} color="onPrimaryContainer" />
        </View>
        <View style={styles.promptText}>
          <ThemedText type="headline">{t('nearMe.title')}</ThemedText>
          {status === 'loading' ? (
            <ActivityIndicator style={styles.loading} />
          ) : (
            <>
              <ThemedText type="small" themeColor="textSecondary">
                {t(`nearMe.${status}Body`)}
              </ThemedText>
              {status === 'blocked' ? (
                // The system won't show the prompt again; only Settings can grant it.
                <Button
                  title={t('nearMe.openSettings')}
                  variant="secondary"
                  onPress={() => Linking.openSettings()}
                  style={styles.button}
                />
              ) : (
                <Button
                  title={t(status === 'idle' ? 'nearMe.show' : 'common.retry')}
                  icon={status === 'idle' ? 'navigation' : 'refresh'}
                  variant="secondary"
                  onPress={request}
                  style={styles.button}
                />
              )}
            </>
          )}
        </View>
      </Card>
    </View>
  );
}

function NearMeResults({ latitude, longitude }: { latitude: number; longitude: number }) {
  const { t } = useTranslation();
  const nearby = useNearbyTargets({ latitude, longitude });

  return (
    <DiscoverSection
      title={t('nearMe.title')}
      query={nearby}
      emptyLabel={t('nearMe.empty')}
      emptyAction={<AddPlaceLink />}
      placeholder={<TargetCardSkeletons />}
      keyExtractor={(target, index) => target.id ?? target.slug ?? String(index)}
      renderItem={(target) => (
        <TargetCard
          target={target}
          badge={target.distance_km != null ? t('nearMe.distance', { km: target.distance_km.toFixed(1) }) : undefined}
          onPress={() => router.push(`/target/${target.slug ?? target.id}`)}
        />
      )}
    />
  );
}

const styles = StyleSheet.create({
  inset: {
    paddingHorizontal: Spacing.four,
  },
  prompt: {
    flexDirection: 'row',
    gap: Spacing.twoHalf,
    padding: Spacing.three,
  },
  iconWrap: {
    width: 44,
    height: 44,
    borderRadius: Radius.md,
    alignItems: 'center',
    justifyContent: 'center',
  },
  promptText: {
    flex: 1,
    gap: Spacing.one,
  },
  loading: {
    alignSelf: 'flex-start',
    marginTop: Spacing.two,
  },
  button: {
    alignSelf: 'flex-start',
    marginTop: Spacing.two,
  },
});
