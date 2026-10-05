import type { UseQueryResult } from '@tanstack/react-query';
import type { ReactElement } from 'react';
import { ActivityIndicator, FlatList, StyleSheet } from 'react-native';

import { QueryError } from '@/components/query-error';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';

export type DiscoverSectionProps<T> = {
  title: string;
  query: UseQueryResult<T[]>;
  emptyLabel: string;
  /** Shown under the empty label, e.g. a way to add what's missing. */
  emptyAction?: ReactElement;
  keyExtractor: (item: T, index: number) => string;
  renderItem: (item: T) => ReactElement;
};

/** Shared loading/error/empty handling for Discover's horizontal lists. */
export function DiscoverSection<T>({
  title,
  query,
  emptyLabel,
  emptyAction,
  keyExtractor,
  renderItem,
}: DiscoverSectionProps<T>) {
  return (
    <ThemedView>
      <ThemedText type="smallBold" style={styles.title}>
        {title}
      </ThemedText>
      {query.isPending ? (
        <ActivityIndicator style={styles.centered} />
      ) : query.isError ? (
        <QueryError onRetry={() => query.refetch()} retrying={query.isRefetching} />
      ) : query.data.length === 0 ? (
        <>
          <ThemedText type="small" themeColor="textSecondary" style={styles.centered}>
            {emptyLabel}
          </ThemedText>
          {emptyAction}
        </>
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
