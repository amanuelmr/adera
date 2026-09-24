import { ActivityIndicator, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { useCriteria } from '../queries';
import { StarPicker } from '../star-picker';

export function CriteriaStep({
  categoryId,
  scores,
  onChange,
}: {
  categoryId: string | undefined;
  scores: Record<string, number>;
  onChange: (code: string, rating: number) => void;
}) {
  const criteria = useCriteria(categoryId);

  if (criteria.isPending) return <ActivityIndicator />;
  if (criteria.isError) {
    return (
      <ThemedText type="small" themeColor="textSecondary">
        Couldn&apos;t load rating criteria — you can still continue without them.
      </ThemedText>
    );
  }
  if (criteria.data.length === 0) {
    return (
      <ThemedText type="small" themeColor="textSecondary">
        Nothing else to rate for this category.
      </ThemedText>
    );
  }

  return (
    <View style={styles.container}>
      <ThemedText type="subtitle">Rate a few specifics</ThemedText>
      {criteria.data.map((criterion) => (
        <View key={criterion.code} style={styles.row}>
          <View style={styles.label}>
            <ThemedText type="smallBold">{criterion.name}</ThemedText>
            {criterion.required === false ? (
              <ThemedText type="small" themeColor="textSecondary">
                Optional
              </ThemedText>
            ) : null}
          </View>
          <StarPicker
            value={criterion.code ? scores[criterion.code] : undefined}
            onChange={(rating) => criterion.code && onChange(criterion.code, rating)}
            label={criterion.name ?? 'Criterion'}
          />
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    gap: Spacing.four,
  },
  row: {
    gap: Spacing.two,
  },
  label: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
});
