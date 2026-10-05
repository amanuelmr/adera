import { router } from 'expo-router';
import { useTranslation } from 'react-i18next';
import { StyleSheet, View } from 'react-native';

import { useAuth } from '@/auth/context';
import { useRequireSignIn } from '@/auth/use-require-sign-in';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { useReviewEligibility } from './queries';

/**
 * Checks the repeat cooldown and daily cap before the form opens
 * (docs/mobile-plan.md §5), so the user is offered "update your review"
 * instead of writing a whole review only to have it rejected. While the check
 * is loading or has failed, it falls back to the plain button — submit
 * enforces the same rules anyway.
 */
export function WriteReviewCta({ targetId, idOrSlug }: { targetId: string | undefined; idOrSlug: string }) {
  const { t } = useTranslation();
  const { status } = useAuth();
  const requireSignIn = useRequireSignIn();
  const eligibility = useReviewEligibility(status === 'signedIn' ? targetId : undefined);
  const result = eligibility.data;

  if (result?.reason === 'cooldown_active' && result.existing_review_id) {
    return (
      <View style={styles.container}>
        <ThemedText type="small" themeColor="textSecondary">
          {t('errors.reviewCooldown')}
        </ThemedText>
        <Button
          title={t('review.updateYours')}
          onPress={() => router.push(`/target/${idOrSlug}/review?reviewId=${result.existing_review_id}`)}
        />
      </View>
    );
  }

  if (result?.reason === 'rate_limited') {
    return (
      <View style={styles.container}>
        <ThemedText type="small" themeColor="textSecondary">
          {t('errors.reviewRateLimited')}
        </ThemedText>
        <Button title={t('nav.writeReview')} disabled onPress={() => undefined} />
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <Button
        title={t('nav.writeReview')}
        onPress={() => requireSignIn(() => router.push(`/target/${idOrSlug}/review`), 'writeReview')}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    padding: Spacing.three,
    gap: Spacing.two,
  },
});
