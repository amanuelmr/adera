import { Stack, router, useLocalSearchParams } from 'expo-router';
import { useTranslation } from 'react-i18next';
import { ActivityIndicator, FlatList, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { QueryError } from '@/components/query-error';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { useCategories, useCategoryTargets } from '@/features/targets/queries';
import { TargetCard } from '@/features/targets/target-card';
import { categoryName } from '@/features/targets/category-name';

export default function CategoryScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { t } = useTranslation();
  const categories = useCategories();
  const targets = useCategoryTargets(id);
  const category = categories.data?.find((c) => c.id === id);

  return (
    <ThemedView style={styles.container}>
      <Stack.Screen options={{ title: categoryName(category) ?? '' }} />
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <FlatList
          data={targets.data ?? []}
          keyExtractor={(item, index) => item.id ?? String(index)}
          contentContainerStyle={styles.list}
          ItemSeparatorComponent={() => <View style={{ height: Spacing.two }} />}
          renderItem={({ item }) => (
            <TargetCard
              name={item.name ?? ''}
              averageRating={item.average_rating ?? null}
              reviewCount={item.review_count ?? 0}
              onPress={() => router.push(`/target/${item.slug ?? item.id}`)}
            />
          )}
          ListEmptyComponent={
            targets.isPending ? (
              <ActivityIndicator style={styles.centered} />
            ) : targets.isError ? (
              <QueryError onRetry={() => targets.refetch()} retrying={targets.isRefetching} />
            ) : (
              <ThemedText type="small" themeColor="textSecondary" style={styles.centered}>
                {t('category.empty')}
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
  },
  centered: {
    padding: Spacing.four,
  },
});
