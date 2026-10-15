import { useTranslation } from 'react-i18next';
import { StyleSheet, View } from 'react-native';

import { Icon } from '@/components/icon';
import { starsFor } from '@/lib/stars';

export type RatingStarsProps = {
  rating: number;
  size?: number;
};

/**
 * Five amber stars for a rating, to the nearest half. Read by screen readers
 * as "Rated 4.3 out of 5"; screens show the number beside it as well, so the
 * colour is never the only way to read the rating.
 */
export function RatingStars({ rating, size = 16 }: RatingStarsProps) {
  const { t } = useTranslation();
  return (
    <View
      style={styles.row}
      accessible
      accessibilityRole="image"
      accessibilityLabel={t('rating.outOfFive', { rating: rating.toFixed(1) })}>
      {starsFor(rating).map((fill, i) => (
        <View key={i} style={{ width: size, height: size }}>
          <Icon name="star" size={size} color={fill === 'full' ? 'rating' : 'border'} filled />
          {fill === 'half' ? (
            <View style={StyleSheet.absoluteFill}>
              <Icon name="star-half" size={size} color="rating" filled />
            </View>
          ) : null}
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    gap: 2,
  },
});
