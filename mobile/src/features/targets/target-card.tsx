import { Image } from 'expo-image';
import { useTranslation } from 'react-i18next';
import { StyleSheet, useWindowDimensions, View } from 'react-native';

import type { components } from '@/api/schema';
import { Badge } from '@/components/badge';
import { Card } from '@/components/card';
import { Icon } from '@/components/icon';
import { RatingStars } from '@/components/rating-stars';
import { SkeletonCards } from '@/components/skeleton';
import { ThemedText } from '@/components/themed-text';
import { Radius, Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { categoryIcon } from '@/features/categories/category-style';
import { useSettings } from '@/lib/settings';
import { categoryName } from './category-name';
import { useCategories } from './queries';

type CardTarget = Pick<
  components['schemas']['Target'],
  'name' | 'average_rating' | 'review_count' | 'category_id' | 'cover_thumb_url' | 'verification_status'
>;

export type TargetCardProps = {
  target: CardTarget;
  /** A short extra line: distance, reviews this month. */
  badge?: string;
  onPress: () => void;
  /** Carousel cards are wide with a cover on top; list rows put a square thumbnail beside the text. */
  layout?: 'carousel' | 'list';
};

/** Card width in a carousel: most of the screen, so the next card peeks in to show the row scrolls. */
export function useCarouselCardWidth(): number {
  const { width } = useWindowDimensions();
  return Math.min(300, Math.round(width * 0.72));
}

export function TargetCard({ target, badge, onPress, layout = 'carousel' }: TargetCardProps) {
  const { t } = useTranslation();
  const categories = useCategories();
  const cardWidth = useCarouselCardWidth();
  const category = categories.data?.find((c) => c.id === target.category_id);
  const name = target.name ?? t('common.unnamed');
  const rating = target.average_rating ?? null;
  const count = target.review_count ?? 0;
  const ratingLabel =
    rating != null ? t('rating.summary', { rating: rating.toFixed(1), count }) : t('rating.notYetRated');
  const meta = [categoryName(category), badge].filter(Boolean).join(' · ');
  const verified = target.verification_status === 'verified';

  const details = (
    <View style={styles.details}>
      <ThemedText type="headline" numberOfLines={1} style={styles.name}>
        {name}
      </ThemedText>
      {rating != null ? (
        <View style={styles.ratingRow}>
          <RatingStars rating={rating} size={14} />
          <ThemedText type="smallBold">{rating.toFixed(1)}</ThemedText>
          <ThemedText type="small" themeColor="textSecondary">
            ({count})
          </ThemedText>
        </View>
      ) : (
        <ThemedText type="small" themeColor="textSecondary">
          {t('rating.noReviewsYet')}
        </ThemedText>
      )}
      {meta ? (
        <ThemedText type="small" themeColor="textSecondary" numberOfLines={1}>
          {meta}
        </ThemedText>
      ) : null}
      {verified ? <Badge label={t('places.verified')} icon="verified" tone="verified" /> : null}
    </View>
  );

  return (
    <Card
      flush
      onPress={onPress}
      accessibilityLabel={`${name}. ${ratingLabel}${meta ? `. ${meta}` : ''}`}
      style={layout === 'carousel' ? { width: cardWidth } : styles.listCard}>
      {layout === 'carousel' ? (
        <>
          <Cover uri={target.cover_thumb_url} categoryCode={category?.code} style={styles.carouselCover} iconSize={36} />
          <View style={styles.carouselBody}>{details}</View>
        </>
      ) : (
        <View style={styles.listRow}>
          <Cover uri={target.cover_thumb_url} categoryCode={category?.code} style={styles.listCover} iconSize={28} />
          {details}
        </View>
      )}
    </Card>
  );
}

/**
 * The place's cover photo, or its category icon on a green tint when there
 * is no photo yet or data saver is on.
 */
function Cover({
  uri,
  categoryCode,
  style,
  iconSize,
}: {
  uri?: string;
  categoryCode?: string;
  style: object;
  iconSize: number;
}) {
  const theme = useTheme();
  const { dataSaver } = useSettings();
  if (uri && !dataSaver) {
    return (
      <Image
        source={{ uri }}
        style={[style, { backgroundColor: theme.backgroundElement }]}
        contentFit="cover"
        transition={150}
        recyclingKey={uri}
        accessible={false}
      />
    );
  }
  return (
    <View style={[style, styles.placeholder, { backgroundColor: theme.primaryContainer }]}>
      <Icon name={categoryIcon(categoryCode)} size={iconSize} color="onPrimaryContainer" />
    </View>
  );
}

const styles = StyleSheet.create({
  carouselCover: {
    width: '100%',
    aspectRatio: 16 / 10,
  },
  carouselBody: {
    padding: Spacing.twoHalf,
  },
  listCard: {
    width: '100%',
  },
  listRow: {
    flexDirection: 'row',
    gap: Spacing.twoHalf,
    padding: Spacing.twoHalf,
    alignItems: 'center',
  },
  listCover: {
    width: 76,
    height: 76,
    borderRadius: Radius.sm,
    overflow: 'hidden',
  },
  placeholder: {
    alignItems: 'center',
    justifyContent: 'center',
  },
  details: {
    flex: 1,
    gap: Spacing.one,
  },
  name: {
    fontSize: 16,
    lineHeight: 22,
  },
  ratingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.one,
  },
});

/** Loading placeholders in the shape of a carousel of place cards. */
export function TargetCardSkeletons() {
  const width = useCarouselCardWidth();
  return <SkeletonCards width={width} height={Math.round(width / 1.6) + 96} count={2} />;
}
