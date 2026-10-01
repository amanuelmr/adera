import { router } from 'expo-router';
import { Alert, StyleSheet, View } from 'react-native';
import { useTranslation } from 'react-i18next';

import type { components } from '@/api/schema';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { useDeleteReview } from './queries';

type ListedReview = components['schemas']['ListedReview'];

export function MyReviewCard({ review }: { review: ListedReview }) {
  const { t } = useTranslation();
  const deleteReview = useDeleteReview();
  const rating = review.overall_rating ?? 0;
  const isPublished = review.moderation_status === 'published';
  // The server refuses new evidence on these (internal/media/media.go).
  const acceptsEvidence = review.moderation_status !== 'rejected' && review.moderation_status !== 'removed';

  function confirmDelete() {
    Alert.alert(t('myReviews.deleteTitle'), t('myReviews.deleteBody'), [
      { text: t('common.cancel'), style: 'cancel' },
      { text: t('common.delete'), style: 'destructive', onPress: () => review.id && deleteReview.mutate(review.id) },
    ]);
  }

  return (
    <ThemedView type="backgroundElement" style={styles.card}>
      <View style={styles.header}>
        <ThemedText type="smallBold" accessibilityLabel={t('rating.outOfFive', { rating })}>
          {'★'.repeat(rating)}
          {'☆'.repeat(Math.max(0, 5 - rating))}
        </ThemedText>
        {!isPublished ? (
          <ThemedText type="small" themeColor="textSecondary">
            {t(`myReviews.status.${review.moderation_status ?? 'pending'}`)}
          </ThemedText>
        ) : null}
      </View>

      {review.title ? <ThemedText type="smallBold">{review.title}</ThemedText> : null}
      <ThemedText type="small" numberOfLines={3}>
        {review.body}
      </ThemedText>

      <View style={styles.actions}>
        <Button title={t('common.viewTarget')} variant="secondary" onPress={() => router.push(`/target/${review.target_id}`)} />
        <Button
          title={t('common.edit')}
          variant="secondary"
          onPress={() => router.push(`/target/${review.target_id}/review?reviewId=${review.id}`)}
        />
        {acceptsEvidence ? (
          <Button
            title={t('evidence.addButton')}
            variant="secondary"
            onPress={() => router.push(`/review/${review.id}/evidence`)}
          />
        ) : null}
        <Button title={t('common.delete')} variant="secondary" onPress={confirmDelete} loading={deleteReview.isPending} />
      </View>
    </ThemedView>
  );
}

const styles = StyleSheet.create({
  card: {
    borderRadius: Spacing.two,
    padding: Spacing.three,
    gap: Spacing.two,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  actions: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing.two,
  },
});
