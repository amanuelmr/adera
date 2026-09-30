import { Stack, useLocalSearchParams } from 'expo-router';
import { useMemo } from 'react';
import { ActivityIndicator, FlatList, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { QueryError } from '@/components/query-error';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { OwnerReviewCard } from '@/features/businesses/owner-review-card';
import { useTarget, useTargetReviews } from '@/features/targets/queries';

export default function BusinessTargetReviewsScreen() {
  const { targetId } = useLocalSearchParams<{ targetId: string }>();
  const target = useTarget(targetId!);
  const reviews = useTargetReviews(targetId, { sort: 'newest' });
  const items = useMemo(() => reviews.data?.pages.flatMap((page) => page.data ?? []) ?? [], [reviews.data]);

  return (
    <ThemedView style={styles.container}>
      <Stack.Screen options={{ title: target.data?.name ?? 'Reviews' }} />
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <FlatList
          data={items}
          keyExtractor={(item) => item.id ?? ''}
          contentContainerStyle={styles.list}
          onEndReachedThreshold={0.5}
          onEndReached={() => {
            if (reviews.hasNextPage && !reviews.isFetchingNextPage) reviews.fetchNextPage();
          }}
          renderItem={({ item }) => <OwnerReviewCard review={item} targetId={targetId!} />}
          ItemSeparatorComponent={() => <View style={{ height: Spacing.two }} />}
          ListEmptyComponent={
            reviews.isPending ? (
              <ActivityIndicator style={styles.centered} />
            ) : reviews.isError ? (
              <QueryError onRetry={() => reviews.refetch()} retrying={reviews.isRefetching} />
            ) : (
              <ThemedText type="small" themeColor="textSecondary" style={styles.centered}>
                No reviews yet.
              </ThemedText>
            )
          }
          ListFooterComponent={reviews.isFetchingNextPage ? <ActivityIndicator style={styles.centered} /> : null}
        />
      </SafeAreaView>
    </ThemedView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  safeArea: {
    flex: 1,
  },
  list: {
    padding: Spacing.four,
    gap: Spacing.three,
  },
  centered: {
    padding: Spacing.four,
  },
});
