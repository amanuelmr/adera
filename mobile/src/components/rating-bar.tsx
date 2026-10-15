import { StyleSheet, View } from 'react-native';

import { Radius, type ThemeColor } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';

export type RatingBarProps = {
  /** 0 to 1. */
  fraction: number;
  color?: ThemeColor;
  height?: number;
};

/** A filled bar on a visible track, for star distributions and criterion scores. */
export function RatingBar({ fraction, color = 'rating', height = 8 }: RatingBarProps) {
  const theme = useTheme();
  const width = `${Math.round(Math.min(1, Math.max(0, fraction)) * 100)}%` as const;
  return (
    <View style={[styles.track, { height, backgroundColor: theme.backgroundSelected }]}>
      <View style={[styles.fill, { width, backgroundColor: theme[color] }]} />
    </View>
  );
}

const styles = StyleSheet.create({
  track: {
    flex: 1,
    borderRadius: Radius.full,
    overflow: 'hidden',
  },
  fill: {
    height: '100%',
    borderRadius: Radius.full,
  },
});
