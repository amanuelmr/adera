import { router } from 'expo-router';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Image, Pressable, StyleSheet, View } from 'react-native';

import type { components } from '@/api/schema';
import { useRequireSignIn } from '@/auth/use-require-sign-in';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { shareReview } from '@/features/share/share';
import { useSettings } from '@/lib/settings';
import { useToggleHelpful } from './queries';

type ListedReview = components['schemas']['ListedReview'];

// Long reviews collapse (docs/frontend-handoff.md §4) — a character count is
// a simpler, good-enough proxy for "~5 lines" than measuring real layout.
const COLLAPSE_AT_CHARS = 280;

export function ReviewCard({ review, targetId, targetName }: { review: ListedReview; targetId: string; targetName: string }) {
  const [expanded, setExpanded] = useState(false);
  const [photosRequested, setPhotosRequested] = useState(false);
  const { dataSaver } = useSettings();
  const toggleHelpful = useToggleHelpful(targetId);
  const requireSignIn = useRequireSignIn();
  const { t } = useTranslation();

  const rating = review.overall_rating ?? 0;
  const body = review.body ?? '';
  const isLong = body.length > COLLAPSE_AT_CHARS;
  const shownBody = isLong && !expanded ? `${body.slice(0, COLLAPSE_AT_CHARS)}…` : body;

  return (
    <ThemedView type="backgroundElement" style={styles.card}>
      <View style={styles.header}>
        <ThemedText type="smallBold">{review.reviewer_name ?? t('review.anonymous')}</ThemedText>
        <ThemedText type="small" themeColor="textSecondary" accessibilityLabel={t('rating.outOfFive', { rating })}>
          {'★'.repeat(rating)}
          {'☆'.repeat(Math.max(0, 5 - rating))}
        </ThemedText>
      </View>

      <DisclosureLabel review={review} />

      {review.title ? <ThemedText type="smallBold">{review.title}</ThemedText> : null}
      <ThemedText type="small">{shownBody}</ThemedText>
      {isLong ? (
        <Pressable onPress={() => setExpanded((prev) => !prev)} accessibilityRole="button">
          <ThemedText type="linkPrimary">{t(expanded ? 'review.showLess' : 'review.readMore')}</ThemedText>
        </Pressable>
      ) : null}

      {review.media && review.media.length > 0 ? (
        dataSaver && !photosRequested ? (
          // Data saver: nothing is downloaded until the reader asks.
          <Pressable onPress={() => setPhotosRequested(true)} accessibilityRole="button" style={styles.helpfulRow}>
            <ThemedText type="linkPrimary">{t('review.showPhotos', { count: review.media.length })}</ThemedText>
          </Pressable>
        ) : (
          <View style={styles.mediaRow}>
            {review.media.map((item) => (
              <Image key={item.id} source={{ uri: item.thumb_url ?? item.url }} style={styles.thumbnail} />
            ))}
          </View>
        )
      ) : null}

      {review.business_response ? (
        <ThemedView type="backgroundSelected" style={styles.response}>
          <ThemedText type="smallBold">{t('review.ownerResponse')}</ThemedText>
          <ThemedText type="small">{review.business_response.body}</ThemedText>
        </ThemedView>
      ) : null}

      <View style={styles.actionsRow}>
        <Pressable
          onPress={() =>
            requireSignIn(
              () => review.id && toggleHelpful.mutate({ reviewId: review.id, voted: !!review.viewer_voted }),
              'vote'
            )
          }
          disabled={toggleHelpful.isPending}
          accessibilityRole="button"
          accessibilityState={{ selected: !!review.viewer_voted }}
          style={styles.helpfulRow}>
          <ThemedText type="small" themeColor={review.viewer_voted ? 'text' : 'textSecondary'}>
            👍 {t('review.helpful')}
            {review.helpful_count ? ` (${review.helpful_count})` : ''}
          </ThemedText>
        </Pressable>
        <Pressable
          onPress={() => requireSignIn(() => review.id && router.push(`/review/${review.id}/report`), 'report')}
          accessibilityRole="button"
          style={styles.helpfulRow}>
          <ThemedText type="small" themeColor="textSecondary">
            {t('report.action')}
          </ThemedText>
        </Pressable>
        <Pressable onPress={() => shareReview(review, targetName)} accessibilityRole="button" style={styles.helpfulRow}>
          <ThemedText type="small" themeColor="textSecondary">
            {t('common.share')}
          </ThemedText>
        </Pressable>
      </View>
    </ThemedView>
  );
}

// Required on every public surface (docs/moderation-policy.md §6): any
// incentive or material connection other than `none` is labeled, never hidden.
function DisclosureLabel({ review }: { review: ListedReview }) {
  const { t } = useTranslation();
  const incentive = review.incentive_type && review.incentive_type !== 'none' ? review.incentive_type : undefined;
  const connection =
    review.material_connection && review.material_connection !== 'none' ? review.material_connection : undefined;
  if (!incentive && !connection) return null;

  return (
    <ThemedView type="backgroundSelected" style={styles.disclosure} accessibilityRole="text">
      {incentive ? (
        <ThemedText type="smallBold">{t('disclosure.badgeIncentive', { value: t(`disclosure.incentive.${incentive}`) })}</ThemedText>
      ) : null}
      {connection ? (
        <ThemedText type="smallBold">
          {t('disclosure.badgeConnection', { value: t(`disclosure.connection.${connection}`) })}
        </ThemedText>
      ) : null}
      {review.disclosure_details ? <ThemedText type="small">{review.disclosure_details}</ThemedText> : null}
    </ThemedView>
  );
}

const styles = StyleSheet.create({
  disclosure: {
    borderRadius: Spacing.two,
    padding: Spacing.two,
    gap: Spacing.half,
  },
  card: {
    borderRadius: Spacing.two,
    padding: Spacing.three,
    gap: Spacing.two,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  mediaRow: {
    flexDirection: 'row',
    gap: Spacing.two,
  },
  thumbnail: {
    width: 64,
    height: 64,
    borderRadius: Spacing.one,
  },
  response: {
    borderRadius: Spacing.two,
    padding: Spacing.two,
    gap: Spacing.half,
  },
  actionsRow: {
    flexDirection: 'row',
    gap: Spacing.three,
  },
  helpfulRow: {
    alignSelf: 'flex-start',
    minHeight: 44,
    justifyContent: 'center',
  },
});
