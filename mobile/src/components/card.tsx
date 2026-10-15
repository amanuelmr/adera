import { Pressable, StyleSheet, View, type StyleProp, type ViewProps, type ViewStyle } from 'react-native';

import { Elevation, Radius, Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';

export type CardProps = ViewProps & {
  onPress?: () => void;
  /** No inner padding, for cards that start with a full-bleed image. */
  flush?: boolean;
  style?: StyleProp<ViewStyle>;
  accessibilityLabel?: string;
};

/** A white (in dark mode, raised) surface on the page, with a hairline border and light depth. */
export function Card({ onPress, flush, style, children, accessibilityLabel, ...rest }: CardProps) {
  const theme = useTheme();
  const base = [
    styles.card,
    Elevation.card,
    { backgroundColor: theme.surface, borderColor: theme.border },
    !flush && styles.padded,
    style,
  ];
  if (!onPress) {
    return (
      <View style={base} {...rest}>
        {children}
      </View>
    );
  }
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel}
      style={({ pressed }) => [...base, pressed && styles.pressed]}
      {...rest}>
      {children}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: {
    borderRadius: Radius.md,
    borderWidth: StyleSheet.hairlineWidth,
    overflow: 'hidden',
  },
  padded: {
    padding: Spacing.twoHalf,
  },
  pressed: {
    opacity: 0.85,
  },
});
