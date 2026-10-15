import { Pressable, StyleSheet } from 'react-native';

import { Icon, type IconName } from '@/components/icon';
import { ThemedText } from '@/components/themed-text';
import { Radius, Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { selectionFeedback } from '@/lib/haptics';

export type FilterChipProps = {
  label: string;
  selected: boolean;
  onPress: () => void;
  icon?: IconName;
};

export function FilterChip({ label, selected, onPress, icon }: FilterChipProps) {
  const theme = useTheme();
  return (
    <Pressable
      onPress={() => {
        selectionFeedback();
        onPress();
      }}
      accessibilityRole="button"
      accessibilityState={{ selected }}
      // Visually 36px tall; the touch area extends to the 44px minimum.
      hitSlop={{ top: 4, bottom: 4 }}
      style={[
        styles.chip,
        selected
          ? { backgroundColor: theme.primaryContainer, borderColor: theme.primary }
          : { backgroundColor: theme.surface, borderColor: theme.border },
      ]}>
      {icon ? <Icon name={icon} size={16} color={selected ? 'onPrimaryContainer' : 'textSecondary'} /> : null}
      <ThemedText type={selected ? 'smallBold' : 'small'} themeColor={selected ? 'onPrimaryContainer' : 'text'}>
        {label}
      </ThemedText>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    minHeight: 36,
    borderRadius: Radius.full,
    borderWidth: 1,
    paddingHorizontal: Spacing.twoHalf,
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.one,
  },
});
