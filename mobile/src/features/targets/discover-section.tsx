import type { UseQueryResult } from '@tanstack/react-query';
import type { ReactElement } from 'react';
import { FlatList, StyleSheet, View } from 'react-native';

import { QueryError } from '@/components/query-error';
import { SectionHeader } from '@/components/section-header';
import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';

export type DiscoverSectionProps<T> = {
  title: string;
  query: UseQueryResult<T[]>;
  /** Without one, the whole section is left out while it has nothing to show. */
  emptyLabel?: string;
  /** Shown under the empty label, e.g. a way to add what's missing. */
  emptyAction?: ReactElement;
  /** Shown while loading, in the shape of the content (skeleton cards). */
  placeholder: ReactElement;
  keyExtractor: (item: T, index: number) => string;
  renderItem: (item: T) => ReactElement;
  /** Gap between items; carousels of cards use a wider one than chips. */
  gap?: number;
};

/** Shared loading/error/empty handling for Discover's horizontal lists. */
export function DiscoverSection<T>({
  title,
  query,
  emptyLabel,
  emptyAction,
  placeholder,
  keyExtractor,
  renderItem,
  gap = Spacing.twoHalf,
}: DiscoverSectionProps<T>) {
  if (!emptyLabel && query.isSuccess && query.data.length === 0) return null;

  return (
    <View style={styles.section}>
      <View style={styles.header}>
        <SectionHeader title={title} />
      </View>
      {query.isPending ? (
        <View style={styles.inset}>{placeholder}</View>
      ) : query.isError ? (
        <QueryError onRetry={() => query.refetch()} retrying={query.isRefetching} />
      ) : query.data.length === 0 ? (
        <View style={styles.inset}>
          <ThemedText type="small" themeColor="textSecondary">
            {emptyLabel}
          </ThemedText>
          {emptyAction}
        </View>
      ) : (
        <FlatList
          horizontal
          showsHorizontalScrollIndicator={false}
          data={query.data}
          keyExtractor={keyExtractor}
          contentContainerStyle={[styles.list, { gap }]}
          renderItem={({ item }) => renderItem(item)}
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  section: {
    gap: Spacing.one,
  },
  header: {
    paddingHorizontal: Spacing.four,
  },
  inset: {
    paddingHorizontal: Spacing.four,
    gap: Spacing.two,
  },
  list: {
    paddingHorizontal: Spacing.four,
    // Room for the cards' shadow, which a horizontal list would clip.
    paddingVertical: Spacing.one,
  },
});
