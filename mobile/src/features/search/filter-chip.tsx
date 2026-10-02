import { Pressable, StyleSheet } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import { Spacing } from '@/constants/theme';

export type FilterChipProps = {
  label: string;
  selected: boolean;
  onPress: () => void;
};

export function FilterChip({ label, selected, onPress }: FilterChipProps) {
  const theme = useTheme();
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityState={{ selected }}
      // Visually 36px tall; the touch area extends to the 44px minimum.
      hitSlop={{ top: 4, bottom: 4 }}
      style={[
        styles.chip,
        { backgroundColor: selected ? theme.text : theme.backgroundElement, borderColor: theme.backgroundSelected },
      ]}>
      <ThemedText type="small" themeColor={selected ? 'background' : 'text'}>
        {label}
      </ThemedText>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    minHeight: 36,
    borderRadius: Spacing.five,
    borderWidth: 1,
    paddingHorizontal: Spacing.three,
    justifyContent: 'center',
  },
});
