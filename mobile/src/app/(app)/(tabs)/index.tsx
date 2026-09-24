import { router } from 'expo-router';
import { Pressable, ScrollView, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { CategoryTile } from '@/features/targets/category-tile';
import { DiscoverSection } from '@/features/targets/discover-section';
import { TargetCard } from '@/features/targets/target-card';
import { useCategories, useTopRatedTargets, useTrendingTargets } from '@/features/targets/queries';

export default function DiscoverScreen() {
  const categories = useCategories();
  const topRated = useTopRatedTargets();
  const trending = useTrendingTargets();

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['top']}>
        <ScrollView contentContainerStyle={styles.content}>
          <ThemedView style={styles.header}>
            <ThemedText type="title" style={styles.title}>
              አደራ
            </ThemedText>
            <Pressable
              onPress={() => router.push('/search')}
              accessibilityRole="search"
              accessibilityLabel="Search targets">
              <ThemedView type="backgroundElement" style={styles.searchBar}>
                <ThemedText themeColor="textSecondary">Search restaurants, salons, sellers…</ThemedText>
              </ThemedView>
            </Pressable>
          </ThemedView>

          <DiscoverSection
            title="Categories"
            query={categories}
            emptyLabel="No categories yet."
            keyExtractor={(category) => category.id ?? category.code ?? category.name ?? ''}
            renderItem={(category) => (
              <CategoryTile
                name={category.name ?? 'Unnamed'}
                onPress={() => router.push({ pathname: '/search', params: { category: category.id ?? '' } })}
              />
            )}
          />

          <DiscoverSection
            title="Top rated"
            query={topRated}
            emptyLabel="Nothing rated yet."
            keyExtractor={(target) => target.id ?? target.slug ?? ''}
            renderItem={(target) => (
              <TargetCard
                name={target.name ?? 'Unnamed'}
                averageRating={target.average_rating ?? null}
                reviewCount={target.review_count ?? 0}
                onPress={() => router.push(`/target/${target.slug ?? target.id}`)}
              />
            )}
          />

          <DiscoverSection
            title="Trending"
            query={trending}
            emptyLabel="Nothing trending yet."
            keyExtractor={(target) => target.id ?? target.slug ?? ''}
            renderItem={(target) => (
              <TargetCard
                name={target.name ?? 'Unnamed'}
                averageRating={target.average_rating ?? null}
                reviewCount={target.review_count ?? 0}
                badge={target.recent_review_count ? `${target.recent_review_count} reviews this month` : undefined}
                onPress={() => router.push(`/target/${target.slug ?? target.id}`)}
              />
            )}
          />
        </ScrollView>
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
  content: {
    gap: Spacing.five,
    paddingBottom: Spacing.six,
  },
  header: {
    paddingHorizontal: Spacing.four,
    gap: Spacing.three,
  },
  title: {
    fontSize: 32,
    lineHeight: 40,
  },
  searchBar: {
    minHeight: 44,
    borderRadius: Spacing.five,
    paddingHorizontal: Spacing.four,
    justifyContent: 'center',
  },
});
