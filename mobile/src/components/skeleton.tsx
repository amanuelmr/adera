import { useEffect, useState } from 'react';
import { AccessibilityInfo, Animated, type DimensionValue, StyleSheet, View } from 'react-native';

import { Radius } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';

export type SkeletonProps = {
  width?: DimensionValue;
  height: number;
  radius?: number;
};

/**
 * A placeholder block in the shape of content still loading. It pulses
 * gently (opacity only, on the native driver, so it costs little on
 * low-end phones) and stays still when Reduce Motion is on.
 */
export function Skeleton({ width = '100%', height, radius = Radius.sm }: SkeletonProps) {
  const theme = useTheme();
  const [opacity] = useState(() => new Animated.Value(1));
  const [reduceMotion, setReduceMotion] = useState(false);

  useEffect(() => {
    AccessibilityInfo.isReduceMotionEnabled().then(setReduceMotion, () => undefined);
  }, []);

  useEffect(() => {
    if (reduceMotion) return;
    const pulse = Animated.loop(
      Animated.sequence([
        Animated.timing(opacity, { toValue: 0.5, duration: 700, useNativeDriver: true }),
        Animated.timing(opacity, { toValue: 1, duration: 700, useNativeDriver: true }),
      ])
    );
    pulse.start();
    return () => pulse.stop();
  }, [opacity, reduceMotion]);

  return (
    <Animated.View
      accessibilityElementsHidden
      importantForAccessibility="no-hide-descendants"
      style={{ width, height, borderRadius: radius, backgroundColor: theme.backgroundSelected, opacity }}
    />
  );
}

/** A loading row of cards, for horizontal carousels. */
export function SkeletonCards({ count = 3, width, height }: { count?: number; width: number; height: number }) {
  return (
    <View style={styles.row} accessibilityLabel="Loading" accessibilityRole="progressbar">
      {Array.from({ length: count }, (_, i) => (
        <Skeleton key={i} width={width} height={height} radius={Radius.md} />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    gap: 12,
  },
});
