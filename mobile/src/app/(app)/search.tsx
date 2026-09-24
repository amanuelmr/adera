import { router, useLocalSearchParams } from 'expo-router';
import { useEffect, useState } from 'react';
import { ActivityIndicator, FlatList, Pressable, StyleSheet, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { FilterChip } from '@/features/search/filter-chip';
import { FiltersSheet } from '@/features/search/filters-sheet';
import { useSearchTargets } from '@/features/search/queries';
import type { SearchFilters } from '@/features/search/types';
import { TARGET_TYPE_LABELS } from '@/features/search/types';
import { useCategories } from '@/features/targets/queries';
import { TargetCard } from '@/features/targets/target-card';
import { useTheme } from '@/hooks/use-theme';

export default function SearchScreen() {
  const params = useLocalSearchParams<{ category?: string }>();
  const categories = useCategories();
  const theme = useTheme();

  const [query, setQuery] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  const [filters, setFilters] = useState<SearchFilters>(params.category ? { category: params.category } : {});
  const [sheetVisible, setSheetVisible] = useState(false);

  // Debounced so typing doesn't fire a rate-limited request per keystroke.
  useEffect(() => {
    const timeout = setTimeout(() => setDebouncedQuery(query), 400);
    return () => clearTimeout(timeout);
  }, [query]);

  const results = useSearchTargets(debouncedQuery, filters);
  // Discover links over a category id only; the name is derived here rather
  // than stored, so it can never drift from the categories list.
  const categoryName = categories.data?.find((category) => category.id === filters.category)?.name;
  const activeFilterCount = Object.values(filters).filter((value) => value !== undefined).length;

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <View style={styles.searchRow}>
          <TextInput
            autoFocus
            value={query}
            onChangeText={setQuery}
            placeholder="Search restaurants, salons, sellers…"
            placeholderTextColor={theme.textSecondary}
            style={[styles.input, { color: theme.text, backgroundColor: theme.backgroundElement }]}
            accessibilityLabel="Search"
          />
          <Pressable
            onPress={() => setSheetVisible(true)}
            accessibilityRole="button"
            accessibilityLabel={`Filters${activeFilterCount ? `, ${activeFilterCount} active` : ''}`}
            style={[styles.filterButton, { backgroundColor: theme.backgroundElement }]}>
            <ThemedText type="smallBold">Filters{activeFilterCount ? ` (${activeFilterCount})` : ''}</ThemedText>
          </Pressable>
        </View>

        {activeFilterCount > 0 ? (
          <View style={styles.chipsRow}>
            {filters.category ? (
              <FilterChip
                label={categoryName ?? 'Category'}
                selected
                onPress={() => setFilters((prev) => ({ ...prev, category: undefined }))}
              />
            ) : null}
            {filters.type ? (
              <FilterChip
                label={TARGET_TYPE_LABELS[filters.type]}
                selected
                onPress={() => setFilters((prev) => ({ ...prev, type: undefined }))}
              />
            ) : null}
            {filters.minRating ? (
              <FilterChip
                label={`${filters.minRating}+ ★`}
                selected
                onPress={() => setFilters((prev) => ({ ...prev, minRating: undefined }))}
              />
            ) : null}
            {filters.verified ? (
              <FilterChip label="Verified" selected onPress={() => setFilters((prev) => ({ ...prev, verified: undefined }))} />
            ) : null}
          </View>
        ) : null}

        {debouncedQuery.trim().length < 2 ? (
          <ThemedText type="small" themeColor="textSecondary" style={styles.centered}>
            Type at least 2 characters to search.
          </ThemedText>
        ) : results.isPending ? (
          <ActivityIndicator style={styles.centered} />
        ) : results.isError ? (
          <ThemedText type="small" themeColor="textSecondary" style={styles.centered}>
            Couldn&apos;t search right now. Check your connection and try again.
          </ThemedText>
        ) : results.data.length === 0 ? (
          <ThemedText type="small" themeColor="textSecondary" style={styles.centered}>
            No matches for &ldquo;{debouncedQuery}&rdquo;. Try a wider area or different spelling.
          </ThemedText>
        ) : (
          <FlatList
            data={results.data}
            keyExtractor={(item) => item.id ?? item.slug ?? ''}
            contentContainerStyle={styles.list}
            renderItem={({ item }) => (
              <TargetCard
                name={item.name ?? 'Unnamed'}
                averageRating={item.average_rating ?? null}
                reviewCount={item.review_count ?? 0}
                onPress={() => router.push(`/target/${item.slug ?? item.id}`)}
                style={styles.resultCard}
              />
            )}
          />
        )}
      </SafeAreaView>

      <FiltersSheet
        visible={sheetVisible}
        filters={filters}
        onApply={(next) => {
          setFilters(next);
          setSheetVisible(false);
        }}
        onClose={() => setSheetVisible(false)}
      />
    </ThemedView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  safeArea: {
    flex: 1,
    gap: Spacing.three,
  },
  searchRow: {
    flexDirection: 'row',
    gap: Spacing.two,
    paddingHorizontal: Spacing.four,
    paddingTop: Spacing.three,
  },
  input: {
    flex: 1,
    minHeight: 44,
    borderRadius: Spacing.five,
    paddingHorizontal: Spacing.four,
    fontSize: 16,
  },
  filterButton: {
    minHeight: 44,
    borderRadius: Spacing.five,
    paddingHorizontal: Spacing.three,
    justifyContent: 'center',
  },
  chipsRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing.two,
    paddingHorizontal: Spacing.four,
  },
  centered: {
    padding: Spacing.four,
    textAlign: 'center',
  },
  list: {
    padding: Spacing.four,
    gap: Spacing.three,
  },
  resultCard: {
    width: '100%',
  },
});
