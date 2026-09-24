import { Stack, router, useLocalSearchParams } from 'expo-router';
import { useMemo, useState } from 'react';
import { ActivityIndicator, FlatList, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { CriterionBars } from '@/features/targets/criterion-bars';
import { RatingHistogram } from '@/features/targets/rating-histogram';
import { useCategories, useTarget, useTargetReviews, useTargetStats, type ReviewSort } from '@/features/targets/queries';
import { ReviewCard } from '@/features/targets/review-card';
import { FilterChip } from '@/features/search/filter-chip';

const SORT_OPTIONS: { value: ReviewSort; label: string }[] = [
  { value: 'newest', label: 'Newest' },
  { value: 'highest', label: 'Highest rated' },
  { value: 'lowest', label: 'Lowest rated' },
  { value: 'most_helpful', label: 'Most helpful' },
];

export default function TargetProfileScreen() {
  const { idOrSlug } = useLocalSearchParams<{ idOrSlug: string }>();
  const target = useTarget(idOrSlug);
  const categories = useCategories();
  const stats = useTargetStats(target.data?.id);

  const [sort, setSort] = useState<ReviewSort>('newest');
  const [rating, setRating] = useState<number>();
  const reviews = useTargetReviews(target.data?.id, { sort, rating });
  const reviewItems = useMemo(() => reviews.data?.pages.flatMap((page) => page.data) ?? [], [reviews.data]);

  if (target.isPending) {
    return (
      <ThemedView style={styles.centered}>
        <ActivityIndicator />
      </ThemedView>
    );
  }

  if (target.isError || !target.data) {
    return (
      <ThemedView style={styles.centered}>
        <ThemedText type="small" themeColor="textSecondary">
          Couldn&apos;t load this target. Check your connection and try again.
        </ThemedText>
      </ThemedView>
    );
  }

  const categoryName = categories.data?.find((category) => category.id === target.data.category_id)?.name;
  const showAggregates = !stats.data || stats.data.confidence !== 'none';

  return (
    <ThemedView style={styles.container}>
      <Stack.Screen options={{ title: target.data.name ?? '' }} />
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <FlatList
          data={reviewItems}
          keyExtractor={(item) => item.id ?? ''}
          contentContainerStyle={styles.list}
          onEndReachedThreshold={0.5}
          onEndReached={() => {
            if (reviews.hasNextPage && !reviews.isFetchingNextPage) reviews.fetchNextPage();
          }}
          ListHeaderComponent={
            <View style={styles.header}>
              <ThemedText type="title" style={styles.name}>
                {target.data.name}
              </ThemedText>
              <ThemedText type="small" themeColor="textSecondary">
                {[categoryName, target.data.address_text].filter(Boolean).join(' · ')}
              </ThemedText>

              {stats.isPending ? (
                <ActivityIndicator style={styles.aggregateGap} />
              ) : stats.isError ? (
                <ThemedText type="small" themeColor="textSecondary" style={styles.aggregateGap}>
                  Couldn&apos;t load ratings right now.
                </ThemedText>
              ) : !showAggregates ? (
                <ThemedText type="small" themeColor="textSecondary" style={styles.aggregateGap}>
                  Not enough reviews yet for reliable stats.
                </ThemedText>
              ) : stats.data ? (
                <View style={styles.aggregateGap}>
                  <View style={styles.averageRow}>
                    <ThemedText
                      type="subtitle"
                      accessibilityLabel={`Rated ${(stats.data.average ?? 0).toFixed(1)} out of 5 from ${stats.data.review_count ?? 0} reviews`}>
                      {(stats.data.average ?? 0).toFixed(1)}
                    </ThemedText>
                    <ThemedText type="small" themeColor="textSecondary">
                      {stats.data.review_count ?? 0} review{stats.data.review_count === 1 ? '' : 's'}
                    </ThemedText>
                  </View>

                  <RatingHistogram distribution={stats.data.distribution ?? []} selectedRating={rating} onSelectRating={setRating} />

                  {stats.data.criterion_averages && stats.data.criterion_averages.length > 0 ? (
                    <View style={styles.criteria}>
                      <CriterionBars criteria={stats.data.criterion_averages} />
                    </View>
                  ) : null}
                </View>
              ) : null}

              <View style={styles.sortRow}>
                {SORT_OPTIONS.map((option) => (
                  <FilterChip
                    key={option.value}
                    label={option.label}
                    selected={sort === option.value}
                    onPress={() => setSort(option.value)}
                  />
                ))}
              </View>
            </View>
          }
          renderItem={({ item }) => <ReviewCard review={item} targetId={target.data.id ?? ''} />}
          ItemSeparatorComponent={() => <View style={{ height: Spacing.two }} />}
          ListEmptyComponent={
            reviews.isPending ? (
              <ActivityIndicator style={styles.centered} />
            ) : (
              <ThemedText type="small" themeColor="textSecondary" style={styles.centered}>
                {rating ? 'No reviews at that rating.' : 'Be the first to review.'}
              </ThemedText>
            )
          }
          ListFooterComponent={reviews.isFetchingNextPage ? <ActivityIndicator style={styles.footer} /> : null}
        />

        <View style={styles.ctaBar}>
          <Button title="Write a review" onPress={() => router.push(`/target/${idOrSlug}/review`)} />
        </View>
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
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    padding: Spacing.four,
  },
  list: {
    padding: Spacing.four,
    gap: Spacing.three,
  },
  header: {
    gap: Spacing.two,
    marginBottom: Spacing.three,
  },
  name: {
    fontSize: 28,
    lineHeight: 34,
  },
  aggregateGap: {
    marginTop: Spacing.three,
    gap: Spacing.three,
  },
  averageRow: {
    flexDirection: 'row',
    alignItems: 'baseline',
    gap: Spacing.two,
  },
  criteria: {
    marginTop: Spacing.one,
  },
  sortRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing.two,
    marginTop: Spacing.three,
  },
  footer: {
    paddingVertical: Spacing.three,
  },
  ctaBar: {
    padding: Spacing.three,
  },
});
