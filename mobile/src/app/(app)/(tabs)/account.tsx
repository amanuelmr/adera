import { Link } from 'expo-router';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { ActivityIndicator, FlatList, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { useAuth } from '@/auth/context';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { FilterChip } from '@/features/search/filter-chip';
import { useProfile, useUpdateProfile } from '@/features/profile/queries';
import { MyReviewCard } from '@/features/reviews/my-review-card';
import { useMyReviews } from '@/features/reviews/queries';
import { SUPPORTED_LANGUAGES, setAppLanguage, type AppLanguage } from '@/lib/i18n';

const LANGUAGE_NAMES: Record<AppLanguage, string> = { en: 'English', am: 'አማርኛ' };

export default function AccountScreen() {
  const { t, i18n } = useTranslation();
  const { logout, logoutAll } = useAuth();
  const profile = useProfile();
  const updateProfile = useUpdateProfile();
  const reviews = useMyReviews();
  const reviewItems = useMemo(() => reviews.data?.pages.flatMap((page) => page.data ?? []) ?? [], [reviews.data]);

  async function changeLanguage(language: AppLanguage) {
    await setAppLanguage(language);
    // Best-effort — the interface language and the account's
    // preferred_language are the same setting from the user's point of view
    // (docs/mobile-plan.md §7), but a sync failure shouldn't undo the switch
    // they just made locally.
    updateProfile.mutate({ preferredLanguage: language }, { onError: () => undefined });
  }

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

              <ThemedText type="smallBold">{t('settings.language')}</ThemedText>
              <View style={styles.languageRow}>
                {SUPPORTED_LANGUAGES.map((language) => (
                  <FilterChip
                    key={language}
                    label={LANGUAGE_NAMES[language]}
                    selected={i18n.language === language}
                    onPress={() => changeLanguage(language)}
                  />
                ))}
              </View>

              <Link href="/verify" style={styles.link}>
                <ThemedText type="link">{t('auth.verifyLink')}</ThemedText>
              </Link>
              <Link href="/sessions" style={styles.link}>
                <ThemedText type="link">{t('auth.devicesLink')}</ThemedText>
              </Link>
              <View style={styles.actions}>
                <Button title={t('common.signOut')} variant="secondary" onPress={() => logout()} />
                <Button title={t('common.signOutEverywhere')} variant="secondary" onPress={() => logoutAll()} />
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
  languageRow: {
    flexDirection: 'row',
    gap: Spacing.two,
    marginBottom: Spacing.two,
  },
  link: {
    paddingVertical: Spacing.one,
  },
  actions: {
    marginTop: Spacing.three,
    gap: Spacing.two,
  },
});
