import { StyleSheet, View } from 'react-native';

import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { dismissJob } from './offline-queue';
import { useOfflineQueue } from './use-offline-queue';

/** A visible retry/pending indicator, per docs/mobile-plan.md §5 — and where rejected queued items surface. */
export function PendingSyncBanner() {
  const { pendingCount, failedJobs } = useOfflineQueue();
  if (pendingCount === 0 && failedJobs.length === 0) return null;

  return (
    <View style={styles.container}>
      {pendingCount > 0 ? (
        <ThemedView type="backgroundSelected" style={styles.banner}>
          <ThemedText type="small">
            {pendingCount} item{pendingCount === 1 ? '' : 's'} waiting to send — we&apos;ll keep retrying.
          </ThemedText>
        </ThemedView>
      ) : null}
      {failedJobs.map((job) => (
        <ThemedView key={job.id} type="backgroundSelected" style={styles.banner}>
          <ThemedText type="smallBold">
            {job.type === 'review-submission' ? "A review couldn't be posted" : "A helpful vote couldn't be saved"}
          </ThemedText>
          <ThemedText type="small">{job.failure?.message}</ThemedText>
          <Button title="Dismiss" variant="secondary" onPress={() => dismissJob(job.id)} />
        </ThemedView>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    marginHorizontal: Spacing.four,
    gap: Spacing.two,
  },
  banner: {
    borderRadius: Spacing.two,
    padding: Spacing.three,
    gap: Spacing.two,
  },
});
