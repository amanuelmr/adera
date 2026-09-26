import { Pressable, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';

export type RatingHistogramProps = {
  /** Counts for 1★..5★, index 0 = 1 star. */
  distribution: number[];
  selectedRating?: number;
  onSelectRating: (rating: number | undefined) => void;
};

// Each bar is a real button, not just a visual — tapping filters the review
// list to that star rating, per docs/frontend-handoff.md §6 (accessible name
// "Filter: 2-star reviews, 14 reviews"; announce applied filters).
export function RatingHistogram({ distribution, selectedRating, onSelectRating }: RatingHistogramProps) {
  const theme = useTheme();
  const maxCount = Math.max(1, ...distribution);

  return (
    <View style={styles.container}>
      {[5, 4, 3, 2, 1].map((rating) => {
        const count = distribution[rating - 1] ?? 0;
        const selected = selectedRating === rating;
        return (
          <Pressable
            key={rating}
            onPress={() => onSelectRating(selected ? undefined : rating)}
            accessibilityRole="button"
            accessibilityState={{ selected }}
            accessibilityLabel={`Filter: ${rating}-star reviews, ${count} review${count === 1 ? '' : 's'}`}
            style={styles.row}>
            <ThemedText type="small" style={styles.label}>
              {rating}★
            </ThemedText>
            <View style={[styles.track, { backgroundColor: theme.backgroundElement }]}>
              <View
                style={[
                  styles.fill,
                  { width: `${(count / maxCount) * 100}%`, backgroundColor: selected ? theme.text : theme.backgroundSelected },
                ]}
              />
            </View>
            <ThemedText type="small" themeColor="textSecondary" style={styles.count}>
              {count}
            </ThemedText>
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    gap: Spacing.one,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.two,
    minHeight: 28,
  },
  label: {
    width: 28,
  },
  track: {
    flex: 1,
    height: 8,
    borderRadius: 4,
    overflow: 'hidden',
  },
  fill: {
    height: '100%',
    borderRadius: 4,
  },
  count: {
    width: 32,
    textAlign: 'right',
  },
});
