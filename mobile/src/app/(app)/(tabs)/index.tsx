import { router } from 'expo-router';
import { useTranslation } from 'react-i18next';
import { Pressable, ScrollView, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { CategoryTile } from '@/features/targets/category-tile';
import { categoryName } from '@/features/targets/category-name';
import { DiscoverSection } from '@/features/targets/discover-section';
import { NearMeSection } from '@/features/targets/near-me-section';
import { TargetCard } from '@/features/targets/target-card';
import { useCategories, useTopRatedTargets, useTrendingTargets } from '@/features/targets/queries';
import { PendingSyncBanner } from '@/features/reviews/pending-sync-banner';

export default function DiscoverScreen() {
  const { t } = useTranslation();
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
              accessibilityLabel={t('nav.search')}>
              <ThemedView type="backgroundElement" style={styles.searchBar}>
                <ThemedText themeColor="textSecondary">{t('search.placeholder')}</ThemedText>
              </ThemedView>
            </Pressable>
          </ThemedView>

          <PendingSyncBanner />

          <NearMeSection />

          <DiscoverSection
            title={t('discover.categories')}
            query={categories}
            emptyLabel={t('discover.noCategories')}
            keyExtractor={(category, index) => category.id ?? category.code ?? category.name ?? String(index)}
            renderItem={(category) => (
              <CategoryTile
                name={categoryName(category) ?? t('common.unnamed')}
                onPress={() => category.id && router.push(`/category/${category.id}`)}
              />
            )}
          />

          <DiscoverSection
            title={t('discover.topRated')}
            query={topRated}
            emptyLabel={t('discover.noTopRated')}
            keyExtractor={(target, index) => target.id ?? target.slug ?? String(index)}
            renderItem={(target) => (
              <TargetCard
                name={target.name ?? t('common.unnamed')}
                averageRating={target.average_rating ?? null}
                reviewCount={target.review_count ?? 0}
                onPress={() => router.push(`/target/${target.slug ?? target.id}`)}
              />
            )}
          />

          <DiscoverSection
            title={t('discover.trending')}
            query={trending}
            emptyLabel={t('discover.noTrending')}
            keyExtractor={(target, index) => target.id ?? target.slug ?? String(index)}
            renderItem={(target) => (
              <TargetCard
                name={target.name ?? t('common.unnamed')}
                averageRating={target.average_rating ?? null}
                reviewCount={target.review_count ?? 0}
                badge={target.recent_review_count ? t('discover.thisMonth', { count: target.recent_review_count }) : undefined}
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
