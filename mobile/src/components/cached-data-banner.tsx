import { onlineManager } from '@tanstack/react-query';
import { useSyncExternalStore } from 'react';
import { useTranslation } from 'react-i18next';
import { StyleSheet } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { shouldShowCachedBanner, type CachedQueryState } from '@/lib/cached-data';
import { formatDateTime, formatRelative } from '@/lib/format';

function useOnline(): boolean {
  return useSyncExternalStore(
    (onChange) => onlineManager.subscribe(onChange),
    () => onlineManager.isOnline()
  );
}

export function CachedDataBanner({ query }: { query: CachedQueryState }) {
  const { t } = useTranslation();
  const online = useOnline();
  if (!shouldShowCachedBanner(query, online)) return null;
  const when = new Date(query.dataUpdatedAt);
  return (
    <ThemedView type="backgroundSelected" style={styles.banner} accessibilityLabel={t('cache.savedAt', { when: formatDateTime(when) })}>
      <ThemedText type="small">
        {t(online ? 'cache.refreshFailed' : 'cache.offline', { when: formatRelative(when) })}
      </ThemedText>
    </ThemedView>
  );
}

const styles = StyleSheet.create({
  banner: {
    borderRadius: Spacing.two,
    padding: Spacing.three,
  },
});
