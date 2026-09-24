import { Link } from 'expo-router';
import { useMemo } from 'react';
import { ActivityIndicator, FlatList, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { useAuth } from '@/auth/context';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { useProfile } from '@/features/profile/queries';
import { MyReviewCard } from '@/features/reviews/my-review-card';
import { useMyReviews } from '@/features/reviews/queries';

export default function AccountScreen() {
  const { logout, logoutAll } = useAuth();
  const profile = useProfile();
  const reviews = useMyReviews();
  const reviewItems = useMemo(() => reviews.data?.pages.flatMap((page) => page.data ?? []) ?? [], [reviews.data]);

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <FlatList
          data={reviewItems}
          keyExtractor={(item) => item.id ?? ''}
          contentContainerStyle={styles.list}
          onEndReachedThreshold={0.5}
          onEndReached={() => {
            if (reviews.hasNextPage && !reviews.isFetchingNextPage) reviews.fetchNextPage();
          }}
          renderItem={({ item }) => <MyReviewCard review={item} />}
          ItemSeparatorComponent={() => <View style={{ height: Spacing.two }} />}
          ListHeaderComponent={
            <View style={styles.header}>
              {profile.data ? (
                <View style={styles.profile}>
                  <ThemedText type="title">{profile.data.display_name}</ThemedText>
                  <ThemedText type="small" themeColor="textSecondary">
                    {profile.data.review_count ?? 0} review{profile.data.review_count === 1 ? '' : 's'}
                    {profile.data.created_at ? ` · Joined ${new Date(profile.data.created_at).toLocaleDateString()}` : ''}
                  </ThemedText>
                  {profile.data.email_verified ? (
                    <ThemedText type="small" themeColor="textSecondary">
                      ✓ Email verified
                    </ThemedText>
                  ) : null}
                </View>
              ) : profile.isPending ? (
                <ActivityIndicator />
              ) : null}

              <ThemedText type="smallBold">Your reviews</ThemedText>
            </View>
          }
          ListEmptyComponent={
            reviews.isPending ? (
              <ActivityIndicator style={styles.centered} />
            ) : (
              <ThemedText type="small" themeColor="textSecondary" style={styles.centered}>
                You haven&apos;t written any reviews yet.
              </ThemedText>
            )
          }
          ListFooterComponent={
            <View style={styles.settings}>
              {reviews.isFetchingNextPage ? <ActivityIndicator /> : null}
              <Link href="/verify" style={styles.link}>
                <ThemedText type="link">Verify your email</ThemedText>
              </Link>
              <Link href="/sessions" style={styles.link}>
                <ThemedText type="link">Your devices</ThemedText>
              </Link>
              <View style={styles.actions}>
                <Button title="Sign out" variant="secondary" onPress={() => logout()} />
                <Button title="Sign out everywhere" variant="secondary" onPress={() => logoutAll()} />
              </View>
            </View>
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
  header: {
    gap: Spacing.three,
    marginBottom: Spacing.three,
  },
  profile: {
    gap: Spacing.half,
  },
  centered: {
    padding: Spacing.four,
  },
  settings: {
    marginTop: Spacing.five,
    gap: Spacing.two,
  },
  link: {
    paddingVertical: Spacing.one,
  },
  actions: {
    marginTop: Spacing.three,
    gap: Spacing.two,
  },
});
