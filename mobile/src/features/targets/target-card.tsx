import { Pressable, StyleSheet } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';

export type TargetCardProps = {
  name: string;
  averageRating: number | null;
  reviewCount: number;
  badge?: string;
  onPress: () => void;
};

export function TargetCard({ name, averageRating, reviewCount, badge, onPress }: TargetCardProps) {
  const ratingLabel =
    averageRating != null
      ? `Rated ${averageRating.toFixed(1)} out of 5 from ${reviewCount} review${reviewCount === 1 ? '' : 's'}`
      : 'Not yet rated';

  return (
    <Pressable onPress={onPress} accessibilityRole="button" accessibilityLabel={`${name}. ${ratingLabel}`}>
      {({ pressed }) => (
        <ThemedView type="backgroundElement" style={[styles.card, { opacity: pressed ? 0.8 : 1 }]}>
          <ThemedText type="smallBold" numberOfLines={1}>
            {name}
          </ThemedText>
          <ThemedText type="small" themeColor="textSecondary">
            {averageRating != null ? `★ ${averageRating.toFixed(1)} (${reviewCount})` : 'No reviews yet'}
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
