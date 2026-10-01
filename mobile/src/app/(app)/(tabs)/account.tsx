import { Link, router } from 'expo-router';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { ActivityIndicator, FlatList, StyleSheet, Switch, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { useAuth } from '@/auth/context';
import { Button } from '@/components/button';
import { QueryError } from '@/components/query-error';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { FilterChip } from '@/features/search/filter-chip';
import { useProfile, useUpdateProfile } from '@/features/profile/queries';
import { MyReviewCard } from '@/features/reviews/my-review-card';
import { useMyReviews } from '@/features/reviews/queries';
import { formatDate } from '@/lib/format';
import { updateSettings, useSettings } from '@/lib/settings';
import { SUPPORTED_LANGUAGES, setAppLanguage, type AppLanguage } from '@/lib/i18n';

const LANGUAGE_NAMES: Record<AppLanguage, string> = { en: 'English', am: 'አማርኛ' };

export default function AccountScreen() {
  const { status } = useAuth();
  return status === 'signedIn' ? <SignedInAccount /> : <SignedOutAccount />;
}

function LanguagePicker({ onChange }: { onChange: (language: AppLanguage) => void }) {
  const { t, i18n } = useTranslation();
  return (
    <>
      <ThemedText type="smallBold">{t('settings.language')}</ThemedText>
      <View style={styles.languageRow}>
        {SUPPORTED_LANGUAGES.map((language) => (
          <FilterChip
            key={language}
            label={LANGUAGE_NAMES[language]}
            selected={i18n.language === language}
            onPress={() => onChange(language)}
          />
        ))}
      </View>
    </>
  );
}

function DataSaverToggle() {
  const { t } = useTranslation();
  const { dataSaver } = useSettings();
  return (
    <View style={styles.toggleRow}>
      <View style={styles.toggleText}>
        <ThemedText type="smallBold">{t('settings.dataSaver')}</ThemedText>
        <ThemedText type="small" themeColor="textSecondary">
          {t('settings.dataSaverBody')}
        </ThemedText>
      </View>
      <Switch
        value={dataSaver}
        onValueChange={(value) => updateSettings({ dataSaver: value })}
        accessibilityLabel={t('settings.dataSaver')}
      />
    </View>
  );
}

function SignedOutAccount() {
  const { t } = useTranslation();
  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <View style={[styles.list, styles.settings]}>
          <ThemedText type="small" themeColor="textSecondary">
            {t('auth.signInToContinue')}
          </ThemedText>
          <View style={styles.actions}>
            <Button title={t('common.signIn')} onPress={() => router.push('/sign-in')} />
            <Button title={t('common.createAccount')} variant="secondary" onPress={() => router.push('/register')} />
          </View>
          <LanguagePicker onChange={(language) => setAppLanguage(language)} />
          <DataSaverToggle />
        </View>
      </SafeAreaView>
    </ThemedView>
  );
}

function SignedInAccount() {
  const { t } = useTranslation();
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
                    {t('account.reviewCount', { count: profile.data.review_count ?? 0 })}
                    {profile.data.created_at ? ` · ${t('account.joined', { date: formatDate(profile.data.created_at) })}` : ''}
                  </ThemedText>
                  {profile.data.email_verified ? (
                    <ThemedText type="small" themeColor="textSecondary">
                      {t('account.emailVerified')}
                    </ThemedText>
                  ) : null}
                </View>
              ) : profile.isPending ? (
                <ActivityIndicator />
              ) : profile.isError ? (
                <QueryError onRetry={() => profile.refetch()} retrying={profile.isRefetching} />
              ) : null}

              <ThemedText type="smallBold">{t('account.yourReviews')}</ThemedText>
            </View>
          }
          ListEmptyComponent={
            reviews.isPending ? (
              <ActivityIndicator style={styles.centered} />
            ) : reviews.isError ? (
              <QueryError onRetry={() => reviews.refetch()} retrying={reviews.isRefetching} />
            ) : (
              <ThemedText type="small" themeColor="textSecondary" style={styles.centered}>
                {t('account.noReviews')}
              </ThemedText>
            )
          }
          ListFooterComponent={
            <View style={styles.settings}>
              {reviews.isFetchingNextPage ? <ActivityIndicator /> : null}

              <LanguagePicker onChange={changeLanguage} />
              <DataSaverToggle />

              <Link href="/verify" style={styles.link}>
                <ThemedText type="link">{t('auth.verifyLink')}</ThemedText>
              </Link>
              <Link href="/businesses" style={styles.link}>
                <ThemedText type="link">{t('nav.yourBusinesses')}</ThemedText>
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
  toggleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.three,
    marginBottom: Spacing.two,
  },
  toggleText: {
    flex: 1,
    gap: Spacing.half,
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
