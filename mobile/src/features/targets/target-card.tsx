import { useTranslation } from 'react-i18next';
import { Pressable, StyleSheet, type StyleProp, type ViewStyle } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';

export type TargetCardProps = {
  name: string;
  averageRating: number | null;
  reviewCount: number;
  badge?: string;
  onPress: () => void;
  style?: StyleProp<ViewStyle>;
};

export function TargetCard({ name, averageRating, reviewCount, badge, onPress, style }: TargetCardProps) {
  const { t } = useTranslation();
  const ratingLabel =
    averageRating != null
      ? t('rating.summary', { rating: averageRating.toFixed(1), count: reviewCount })
      : t('rating.notYetRated');

  return (
    <Pressable onPress={onPress} accessibilityRole="button" accessibilityLabel={`${name}. ${ratingLabel}`}>
      {({ pressed }) => (
        <ThemedView type="backgroundElement" style={[styles.card, { opacity: pressed ? 0.8 : 1 }, style]}>
          <ThemedText type="smallBold" numberOfLines={1}>
            {name}
          </ThemedText>
          <ThemedText type="small" themeColor="textSecondary">
            {averageRating != null ? `★ ${averageRating.toFixed(1)} (${reviewCount})` : t('rating.noReviewsYet')}
          </ThemedText>
          {badge ? (
            <ThemedText type="small" themeColor="textSecondary">
              {badge}
            </ThemedText>
          ) : null}
        </ThemedView>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: {
    width: 180,
    borderRadius: Spacing.two,
    padding: Spacing.three,
    gap: Spacing.half,
  },
});
