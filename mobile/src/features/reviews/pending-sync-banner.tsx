import { useTranslation } from 'react-i18next';
import { StyleSheet, View } from 'react-native';

import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { dismissJob } from './offline-queue';
import { useOfflineQueue } from './use-offline-queue';

/** A visible retry/pending indicator, per docs/mobile-plan.md §5 — and where rejected queued items surface. */
export function PendingSyncBanner() {
  const { t } = useTranslation();
  const { pendingCount, failedJobs } = useOfflineQueue();
  if (pendingCount === 0 && failedJobs.length === 0) return null;

  return (
    <View style={styles.container}>
      {pendingCount > 0 ? (
        <ThemedView type="backgroundSelected" style={styles.banner}>
          <ThemedText type="small">
            {t('sync.pending', { count: pendingCount })}
          </ThemedText>
        </ThemedView>
      ) : null}
      {failedJobs.map((job) => (
        <ThemedView key={job.id} type="backgroundSelected" style={styles.banner}>
          <ThemedText type="smallBold">
            {t(job.type === 'review-submission' ? 'sync.reviewFailed' : 'sync.voteFailed')}
          </ThemedText>
          <ThemedText type="small">
            {job.failure?.code === 'cooldown_active'
              ? t('errors.reviewCooldown')
              : job.failure?.code === 'rate_limited'
                ? t('errors.reviewRateLimited')
                : t('sync.rejected')}
          </ThemedText>
          <Button title={t('sync.dismiss')} variant="secondary" onPress={() => dismissJob(job.id)} />
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
