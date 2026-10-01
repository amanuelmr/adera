import { Redirect, useLocalSearchParams } from 'expo-router';
import { ActivityIndicator, StyleSheet } from 'react-native';

import { QueryError } from '@/components/query-error';
import { ThemedView } from '@/components/themed-view';
import { useReview } from '@/features/reviews/queries';

/**
 * https://…/r/{id} opened via App Links. There's no standalone review
 * screen in the app — reviews live on their place's page — so this resolves
 * the review's place and opens that.
 */
export default function WebReviewLink() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const review = useReview(id);

  if (review.data?.target_id) return <Redirect href={`/target/${review.data.target_id}`} />;
  return (
    <ThemedView style={styles.centered}>
      {review.isError ? <QueryError onRetry={() => review.refetch()} /> : <ActivityIndicator />}
    </ThemedView>
  );
}

const styles = StyleSheet.create({
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
});
