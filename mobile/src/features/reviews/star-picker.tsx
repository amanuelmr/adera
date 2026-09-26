import { Pressable, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';

export function StarPicker({
  value,
  onChange,
  size = 28,
  label,
}: {
  value?: number;
  onChange: (rating: number) => void;
  size?: number;
  label: string;
}) {
  return (
    <View style={styles.row}>
      {[1, 2, 3, 4, 5].map((star) => (
        <Pressable
          key={star}
          onPress={() => onChange(star)}
          accessibilityRole="radio"
          accessibilityState={{ checked: value === star }}
          accessibilityLabel={`${label}: ${star} stars`}
          hitSlop={6}>
          <ThemedText style={{ fontSize: size }}>{value != null && star <= value ? '★' : '☆'}</ThemedText>
        </Pressable>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    gap: Spacing.one,
  },
});
