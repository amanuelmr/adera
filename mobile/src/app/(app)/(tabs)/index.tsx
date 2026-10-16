import { Image } from 'expo-image';
import { router } from 'expo-router';
import { useTranslation } from 'react-i18next';
import { Pressable, ScrollView, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { CachedDataBanner } from '@/components/cached-data-banner';
import { Icon } from '@/components/icon';
import { Skeleton } from '@/components/skeleton';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Elevation, Radius, Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { CategoryTile } from '@/features/targets/category-tile';
import { categoryName } from '@/features/targets/category-name';
import { DiscoverSection } from '@/features/targets/discover-section';
import { NearMeSection } from '@/features/targets/near-me-section';
import { TargetCard, TargetCardSkeletons } from '@/features/targets/target-card';
import { useCategories, useTopRatedTargets, useTrendingTargets } from '@/features/targets/queries';
import { PendingSyncBanner } from '@/features/reviews/pending-sync-banner';

const APP_ICON = require('../../../../assets/images/logo-small.png');

export default function DiscoverScreen() {
  const { t } = useTranslation();
  const theme = useTheme();
  const categories = useCategories();
  const topRated = useTopRatedTargets();
  const trending = useTrendingTargets();

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['top']}>
        <ScrollView contentContainerStyle={styles.content}>
          <View style={styles.header}>
            <View style={styles.brand}>
              <Image source={APP_ICON} style={styles.logo} accessible={false} />
              <View style={styles.brandText}>
                <ThemedText type="title" accessibilityRole="header">
                  አደራ
                </ThemedText>
                <ThemedText type="small" themeColor="textSecondary">
                  {t('discover.tagline')}
                </ThemedText>
              </View>
            </View>
            <Pressable
              onPress={() => router.push('/search')}
              accessibilityRole="search"
              accessibilityLabel={t('nav.search')}
              style={({ pressed }) => [
                styles.searchBar,
                Elevation.card,
                { backgroundColor: theme.surface, borderColor: theme.border, opacity: pressed ? 0.85 : 1 },
              ]}>
              <Icon name="search" size={20} color="textSecondary" />
              <ThemedText themeColor="textSecondary" numberOfLines={1} style={styles.searchText}>
                {t('search.placeholder')}
              </ThemedText>
            </Pressable>
          </View>

          <PendingSyncBanner />

          <View style={styles.inset}>
            <CachedDataBanner query={topRated} />
          </View>

          <DiscoverSection
            title={t('discover.categories')}
            query={categories}
            emptyLabel={t('discover.noCategories')}
            gap={Spacing.two}
            placeholder={
              <View style={styles.chipSkeletons}>
                <Skeleton width={140} height={48} radius={Radius.full} />
                <Skeleton width={160} height={48} radius={Radius.full} />
              </View>
            }
            keyExtractor={(category, index) => category.id ?? category.code ?? category.name ?? String(index)}
            renderItem={(category) => (
              <CategoryTile
                name={categoryName(category) ?? t('common.unnamed')}
                code={category.code}
                onPress={() => category.id && router.push(`/category/${category.id}`)}
              />
            )}
          />

          <NearMeSection />

          {/* A new city has no ratings yet: rather than two "nothing here"
              rows, these appear once there is something to show. */}
          <DiscoverSection
            title={t('discover.topRated')}
            query={topRated}
            placeholder={<TargetCardSkeletons />}
            keyExtractor={(target, index) => target.id ?? target.slug ?? String(index)}
            renderItem={(target) => (
              <TargetCard target={target} onPress={() => router.push(`/target/${target.slug ?? target.id}`)} />
            )}
          />

          <DiscoverSection
            title={t('discover.trending')}
            query={trending}
            placeholder={<TargetCardSkeletons />}
            keyExtractor={(target, index) => target.id ?? target.slug ?? String(index)}
            renderItem={(target) => (
              <TargetCard
                target={target}
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
    gap: Spacing.four,
    paddingTop: Spacing.three,
    paddingBottom: Spacing.six,
  },
  header: {
    paddingHorizontal: Spacing.four,
    gap: Spacing.three,
  },
  brand: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.twoHalf,
  },
  logo: {
    width: 44,
    height: 44,
    borderRadius: Radius.md,
  },
  brandText: {
    flex: 1,
  },
  inset: {
    paddingHorizontal: Spacing.four,
  },
  searchBar: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.two,
    minHeight: 52,
    borderRadius: Radius.full,
    borderWidth: StyleSheet.hairlineWidth,
    paddingHorizontal: Spacing.three,
  },
  searchText: {
    flex: 1,
  },
  chipSkeletons: {
    flexDirection: 'row',
    gap: Spacing.two,
  },
});
