import { router, useLocalSearchParams } from 'expo-router';
import { ActivityIndicator, FlatList, Pressable, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { QueryError } from '@/components/query-error';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { useBusinessTargets } from '@/features/businesses/queries';

export default function BusinessTargetsScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const targets = useBusinessTargets(id);

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <FlatList
          data={targets.data ?? []}
          keyExtractor={(item) => item.id ?? ''}
          contentContainerStyle={styles.list}
          ItemSeparatorComponent={() => <View style={{ height: Spacing.two }} />}
          renderItem={({ item }) => (
            <Pressable
              onPress={() => id && item.id && router.push(`/businesses/${id}/targets/${item.id}`)}
              accessibilityRole="button">
              <ThemedView type="backgroundElement" style={styles.card}>
                <ThemedText type="smallBold">{item.name}</ThemedText>
                <ThemedText type="small" themeColor="textSecondary">
                  {item.review_count ?? 0} review{item.review_count === 1 ? '' : 's'}
                </ThemedText>
              </ThemedView>
            </Pressable>
          )}
          ListEmptyComponent={
            targets.isPending ? (
              <ActivityIndicator style={styles.centered} />
            ) : targets.isError ? (
              <QueryError onRetry={() => targets.refetch()} retrying={targets.isRefetching} />
            ) : (
              <ThemedText type="small" themeColor="textSecondary" style={styles.centered}>
                No published locations yet.
              </ThemedText>
            )
          }
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
  card: {
    borderRadius: Spacing.two,
    padding: Spacing.three,
    gap: Spacing.half,
  },
  centered: {
    padding: Spacing.four,
  },
});
