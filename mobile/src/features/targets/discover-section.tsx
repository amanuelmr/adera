import type { UseQueryResult } from '@tanstack/react-query';
import type { ReactElement } from 'react';
import { ActivityIndicator, FlatList, StyleSheet } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';

export type DiscoverSectionProps<T> = {
  title: string;
  query: UseQueryResult<T[]>;
  emptyLabel: string;
  keyExtractor: (item: T) => string;
  renderItem: (item: T) => ReactElement;
};

/** Shared loading/error/empty handling for Discover's horizontal lists. */
export function DiscoverSection<T>({ title, query, emptyLabel, keyExtractor, renderItem }: DiscoverSectionProps<T>) {
  return (
    <ThemedView>
      <ThemedText type="smallBold" style={styles.title}>
        {title}
      </ThemedText>
      {query.isPending ? (
        <ActivityIndicator style={styles.centered} />
      ) : query.isError ? (
        <ThemedText type="small" themeColor="textSecondary" style={styles.centered}>
          Couldn&apos;t load {title.toLowerCase()}. Pull down to retry.
        </ThemedText>
      ) : query.data.length === 0 ? (
        <ThemedText type="small" themeColor="textSecondary" style={styles.centered}>
          {emptyLabel}
        </ThemedText>
      ) : (
        <FlatList
          horizontal
          showsHorizontalScrollIndicator={false}
          data={query.data}
          keyExtractor={keyExtractor}
          contentContainerStyle={styles.list}
          renderItem={({ item }) => renderItem(item)}
        />
      )}
    </ThemedView>
  );
}

const styles = StyleSheet.create({
  title: {
    paddingHorizontal: Spacing.four,
    marginBottom: Spacing.two,
  },
  centered: {
    paddingHorizontal: Spacing.four,
  },
  list: {
    paddingHorizontal: Spacing.four,
    gap: Spacing.three,
  },
});
