import { StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { StarPicker } from '../star-picker';
import { RATING_LABELS } from '../types';

export function RatingStep({ value, onChange }: { value?: number; onChange: (rating: number) => void }) {
  return (
    <View style={styles.container}>
      <ThemedText type="subtitle">Overall, how was it?</ThemedText>
      <StarPicker value={value} onChange={onChange} size={40} label="Overall rating" />
      {value ? <ThemedText themeColor="textSecondary">{RATING_LABELS[value]}</ThemedText> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    gap: Spacing.three,
    alignItems: 'center',
  },
});
