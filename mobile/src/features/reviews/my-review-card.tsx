import { router } from 'expo-router';
import { Alert, StyleSheet, View } from 'react-native';

import type { components } from '@/api/schema';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { useDeleteReview } from './queries';

type ListedReview = components['schemas']['ListedReview'];

const MODERATION_LABELS: Record<NonNullable<ListedReview['moderation_status']>, string> = {
  pending: 'Pending review',
  published: 'Published',
  under_review: 'Under review',
  rejected: 'Rejected',
  hidden: 'Hidden',
  removed: 'Removed',
};

export function MyReviewCard({ review }: { review: ListedReview }) {
  const deleteReview = useDeleteReview();
  const rating = review.overall_rating ?? 0;
  const isPublished = review.moderation_status === 'published';

  function confirmDelete() {
    Alert.alert('Delete this review?', 'This can’t be undone.', [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Delete', style: 'destructive', onPress: () => review.id && deleteReview.mutate(review.id) },
    ]);
  }

  return (
    <ThemedView type="backgroundElement" style={styles.card}>
      <View style={styles.header}>
        <ThemedText type="smallBold" accessibilityLabel={`Rated ${rating} out of 5`}>
          {'★'.repeat(rating)}
          {'☆'.repeat(Math.max(0, 5 - rating))}
        </ThemedText>
        {!isPublished ? (
          <ThemedText type="small" themeColor="textSecondary">
            {MODERATION_LABELS[review.moderation_status ?? 'pending']}
          </ThemedText>
        ) : null}
      </View>

      {review.title ? <ThemedText type="smallBold">{review.title}</ThemedText> : null}
      <ThemedText type="small" numberOfLines={3}>
        {review.body}
      </ThemedText>

      <View style={styles.actions}>
        <Button title="View target" variant="secondary" onPress={() => router.push(`/target/${review.target_id}`)} />
        <Button
          title="Edit"
          variant="secondary"
          onPress={() => router.push(`/target/${review.target_id}/review?reviewId=${review.id}`)}
        />
        <Button title="Delete" variant="secondary" onPress={confirmDelete} loading={deleteReview.isPending} />
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
