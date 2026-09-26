import { StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';

export type Criterion = {
  code?: string;
  name?: string;
  average?: number;
  count?: number;
};

export function CriterionBars({ criteria }: { criteria: Criterion[] }) {
  const theme = useTheme();
  if (criteria.length === 0) return null;

  return (
    <View style={styles.container}>
      {criteria.map((criterion) => (
        <View key={criterion.code} style={styles.row}>
          <ThemedText type="small" style={styles.label} numberOfLines={1}>
            {criterion.name ?? criterion.code}
          </ThemedText>
          <View style={[styles.track, { backgroundColor: theme.backgroundElement }]}>
            <View
              style={[
                styles.fill,
                { width: `${((criterion.average ?? 0) / 5) * 100}%`, backgroundColor: theme.backgroundSelected },
              ]}
            />
          </View>
          <ThemedText type="small" themeColor="textSecondary" style={styles.value}>
            {criterion.average != null ? criterion.average.toFixed(1) : '—'}
          </ThemedText>
        </View>
      ))}
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
    minHeight: 24,
  },
  label: {
    width: 96,
  },
  track: {
    flex: 1,
    height: 6,
    borderRadius: 3,
    overflow: 'hidden',
  },
  fill: {
    height: '100%',
    borderRadius: 3,
  },
  value: {
    width: 28,
    textAlign: 'right',
  },
});
