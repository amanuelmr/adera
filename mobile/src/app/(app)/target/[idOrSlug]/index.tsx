import { Stack, useLocalSearchParams } from 'expo-router';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ActivityIndicator, FlatList, Pressable, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { QueryError } from '@/components/query-error';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { CriterionBars } from '@/features/targets/criterion-bars';
import { RatingHistogram } from '@/features/targets/rating-histogram';
import { RealityCheckPanel } from '@/features/targets/reality-check-panel';
import {
  useCategories,
  useRealityCheck,
  useTarget,
  useTargetReviews,
  useTargetStats,
  type ReviewSort,
} from '@/features/targets/queries';
import { ReviewCard } from '@/features/targets/review-card';
import { categoryName } from '@/features/targets/category-name';
import { FilterChip } from '@/features/search/filter-chip';
import { WriteReviewCta } from '@/features/reviews/write-review-cta';
import { shareTarget } from '@/features/share/share';

const REALITY_CHECK_CATEGORY_CODE = 'restaurant_cafe';

const SORT_OPTIONS: ReviewSort[] = ['newest', 'highest', 'lowest', 'most_helpful'];

export default function TargetProfileScreen() {
  const { idOrSlug } = useLocalSearchParams<{ idOrSlug: string }>();
  const { t } = useTranslation();
  const target = useTarget(idOrSlug);
  const categories = useCategories();
  const stats = useTargetStats(target.data?.id);
  const category = categories.data?.find((c) => c.id === target.data?.category_id);
  const showRealityCheck = category?.code === REALITY_CHECK_CATEGORY_CODE;
  const realityCheck = useRealityCheck(showRealityCheck ? target.data?.id : undefined);

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
          {t('review.loadTargetFailed')}
        </ThemedText>
      </ThemedView>
    );
  }

  const categoryLabel = categoryName(category);
  const showAggregates = !stats.data || stats.data.confidence !== 'none';

  return (
    <ThemedView style={styles.container}>
      <Stack.Screen
        options={{
          title: target.data.name ?? '',
          headerRight: () => (
            <Pressable onPress={() => shareTarget(target.data!)} accessibilityRole="button">
              <ThemedText type="link">{t('common.share')}</ThemedText>
            </Pressable>
          ),
        }}
      />
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <FlatList
          data={reviewItems}
          keyExtractor={(item, index) => item.id ?? String(index)}
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
                {[categoryLabel, target.data.address_text].filter(Boolean).join(' · ')}
              </ThemedText>

              {stats.isPending ? (
                <ActivityIndicator style={styles.aggregateGap} />
              ) : stats.isError ? (
                <ThemedText type="small" themeColor="textSecondary" style={styles.aggregateGap}>
                  {t('target.statsFailed')}
                </ThemedText>
              ) : !showAggregates ? (
                <ThemedText type="small" themeColor="textSecondary" style={styles.aggregateGap}>
                  {t('target.notEnoughReviews')}
                </ThemedText>
              ) : stats.data ? (
                <View style={styles.aggregateGap}>
                  <View style={styles.averageRow}>
                    <ThemedText
                      type="subtitle"
                      accessibilityLabel={t('rating.summary', {
                        rating: (stats.data.average ?? 0).toFixed(1),
                        count: stats.data.review_count ?? 0,
                      })}>
                      {(stats.data.average ?? 0).toFixed(1)}
                    </ThemedText>
                    <ThemedText type="small" themeColor="textSecondary">
                      {t('account.reviewCount', { count: stats.data.review_count ?? 0 })}
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

              {showRealityCheck && realityCheck.data ? (
                <View style={styles.realityCheck}>
                  <RealityCheckPanel realityCheck={realityCheck.data} />
                </View>
              ) : null}

              <View style={styles.sortRow}>
                {SORT_OPTIONS.map((option) => (
                  <FilterChip
                    key={option}
                    label={t(`target.sort.${option}`)}
                    selected={sort === option}
                    onPress={() => setSort(option)}
                  />
                ))}
              </View>
            </View>
          }
          renderItem={({ item }) => (
            <ReviewCard review={item} targetId={target.data.id ?? ''} targetName={target.data.name ?? ''} />
          )}
          ItemSeparatorComponent={() => <View style={{ height: Spacing.two }} />}
          ListEmptyComponent={
            reviews.isPending ? (
              <ActivityIndicator style={styles.centered} />
            ) : reviews.isError ? (
              <QueryError onRetry={() => reviews.refetch()} retrying={reviews.isRefetching} />
            ) : (
              <ThemedText type="small" themeColor="textSecondary" style={styles.centered}>
                {t(rating ? 'target.noReviewsAtRating' : 'target.beFirst')}
              </ThemedText>
            )
          }
          ListFooterComponent={reviews.isFetchingNextPage ? <ActivityIndicator style={styles.footer} /> : null}
        />

        <WriteReviewCta targetId={target.data.id} idOrSlug={idOrSlug} />
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
  realityCheck: {
    marginTop: Spacing.three,
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
});
