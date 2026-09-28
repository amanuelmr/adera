import { useState } from 'react';
import { Image, Pressable, StyleSheet, View } from 'react-native';

import type { components } from '@/api/schema';
import { useRequireSignIn } from '@/auth/use-require-sign-in';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { shareReview } from '@/features/share/share';
import { useToggleHelpful } from './queries';

type ListedReview = components['schemas']['ListedReview'];

// Long reviews collapse (docs/frontend-handoff.md §4) — a character count is
// a simpler, good-enough proxy for "~5 lines" than measuring real layout.
const COLLAPSE_AT_CHARS = 280;

export function ReviewCard({ review, targetId, targetName }: { review: ListedReview; targetId: string; targetName: string }) {
  const [expanded, setExpanded] = useState(false);
  const toggleHelpful = useToggleHelpful(targetId);
  const requireSignIn = useRequireSignIn();

  const rating = review.overall_rating ?? 0;
  const body = review.body ?? '';
  const isLong = body.length > COLLAPSE_AT_CHARS;
  const shownBody = isLong && !expanded ? `${body.slice(0, COLLAPSE_AT_CHARS)}…` : body;

  return (
    <ThemedView type="backgroundElement" style={styles.card}>
      <View style={styles.header}>
        <ThemedText type="smallBold">{review.reviewer_name ?? 'Anonymous'}</ThemedText>
        <ThemedText type="small" themeColor="textSecondary" accessibilityLabel={`Rated ${rating} out of 5`}>
          {'★'.repeat(rating)}
          {'☆'.repeat(Math.max(0, 5 - rating))}
        </ThemedText>
      </View>

      {review.title ? <ThemedText type="smallBold">{review.title}</ThemedText> : null}
      <ThemedText type="small">{shownBody}</ThemedText>
      {isLong ? (
        <Pressable onPress={() => setExpanded((prev) => !prev)} accessibilityRole="button">
          <ThemedText type="linkPrimary">{expanded ? 'Show less' : 'Read more'}</ThemedText>
        </Pressable>
      ) : null}

      {review.media && review.media.length > 0 ? (
        <View style={styles.mediaRow}>
          {review.media.map((item) => (
            <Image key={item.id} source={{ uri: item.thumb_url ?? item.url }} style={styles.thumbnail} />
          ))}
        </View>
      ) : null}

      {review.business_response ? (
        <ThemedView type="backgroundSelected" style={styles.response}>
          <ThemedText type="smallBold">Owner response</ThemedText>
          <ThemedText type="small">{review.business_response.body}</ThemedText>
        </ThemedView>
      ) : null}

      <View style={styles.actionsRow}>
        <Pressable
          onPress={() =>
            requireSignIn(() => review.id && toggleHelpful.mutate({ reviewId: review.id, voted: !!review.viewer_voted }))
          }
          disabled={toggleHelpful.isPending}
          accessibilityRole="button"
          accessibilityState={{ selected: !!review.viewer_voted }}
          style={styles.helpfulRow}>
          <ThemedText type="small" themeColor={review.viewer_voted ? 'text' : 'textSecondary'}>
            👍 Helpful{review.helpful_count ? ` (${review.helpful_count})` : ''}
          </ThemedText>
        </Pressable>
        <Pressable onPress={() => shareReview(review, targetName)} accessibilityRole="button" style={styles.helpfulRow}>
          <ThemedText type="small" themeColor="textSecondary">
            Share
          </ThemedText>
        </Pressable>
      </View>
    </ThemedView>
  );
}

const styles = StyleSheet.create({
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
    paddingVertical: Spacing.one,
  },
});
