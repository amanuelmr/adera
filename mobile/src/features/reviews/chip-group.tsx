import { StyleSheet, View } from 'react-native';

import { FilterChip } from '@/features/search/filter-chip';
import { Spacing } from '@/constants/theme';

export function ChipGroup<T extends string>({
  options,
  value,
  onChange,
}: {
  options: { value: T; label: string }[];
  value?: T;
  onChange: (value: T) => void;
}) {
  return (
    <View style={styles.row}>
      {options.map((option) => (
        <FilterChip
          key={option.value}
          label={option.label}
          selected={value === option.value}
          onPress={() => onChange(option.value)}
        />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing.two,
  },
});
