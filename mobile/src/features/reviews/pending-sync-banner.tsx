import { StyleSheet } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { usePendingSyncCount } from './use-offline-queue';

/** A visible retry/pending indicator, per docs/mobile-plan.md §5. */
export function PendingSyncBanner() {
  const count = usePendingSyncCount();
  if (count === 0) return null;

  return (
    <ThemedView type="backgroundSelected" style={styles.banner}>
      <ThemedText type="small">
        {count} item{count === 1 ? '' : 's'} waiting to send — we&apos;ll retry once you&apos;re back online.
      </ThemedText>
    </ThemedView>
  );
}

const styles = StyleSheet.create({
  banner: {
    marginHorizontal: Spacing.four,
    borderRadius: Spacing.two,
    padding: Spacing.three,
  },
});
