import { router } from 'expo-router';
import { ActivityIndicator, FlatList, Pressable, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { QueryError } from '@/components/query-error';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { useMyBusinesses } from '@/features/businesses/queries';

const VERIFICATION_LABELS: Record<string, string> = {
  unverified: 'Unverified',
  claimed: 'Claimed',
  verified: 'Verified',
};

export default function MyBusinessesScreen() {
  const businesses = useMyBusinesses();

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <FlatList
          data={businesses.data ?? []}
          keyExtractor={(item) => item.id ?? ''}
          contentContainerStyle={styles.list}
          ItemSeparatorComponent={() => <View style={{ height: Spacing.two }} />}
          renderItem={({ item }) => (
            <Pressable onPress={() => item.id && router.push(`/businesses/${item.id}`)} accessibilityRole="button">
              <ThemedView type="backgroundElement" style={styles.card}>
                <ThemedText type="smallBold">{item.name}</ThemedText>
                <ThemedText type="small" themeColor="textSecondary">
                  {VERIFICATION_LABELS[item.verification_status ?? 'unverified']}
                </ThemedText>
              </ThemedView>
            </Pressable>
          )}
          ListEmptyComponent={
            businesses.isPending ? (
              <ActivityIndicator style={styles.centered} />
            ) : businesses.isError ? (
              <QueryError onRetry={() => businesses.refetch()} retrying={businesses.isRefetching} />
            ) : (
              <ThemedText type="small" themeColor="textSecondary" style={styles.centered}>
                You don&apos;t manage any businesses yet.
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
