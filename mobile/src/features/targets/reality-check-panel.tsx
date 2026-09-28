import { StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import type { components } from '@/api/schema';

type RealityCheck = components['schemas']['RealityCheck'];

const CONFIDENCE_CAVEAT: Partial<Record<NonNullable<RealityCheck['confidence']>, string>> = {
  none: 'Based on very few reviews so far — take this as an early signal, not a verdict.',
  low: 'Based on a small number of reviews so far.',
};

export function RealityCheckPanel({ realityCheck }: { realityCheck: RealityCheck }) {
  const theme = useTheme();
  const expectations = realityCheck.expectation_distribution;
  const percentages = expectations?.percentages;
  const caveat = realityCheck.confidence ? CONFIDENCE_CAVEAT[realityCheck.confidence] : undefined;

  return (
    <View style={styles.container}>
      <ThemedText type="smallBold">Reality Check</ThemedText>
      <ThemedText type="small" themeColor="textSecondary">
        Did this place match what people expected going in?
      </ThemedText>

      {realityCheck.trend_available && realityCheck.recent_trend != null ? (
        <ThemedText type="small">
          Recent reviews average {realityCheck.recent_average?.toFixed(1)}, vs {realityCheck.historical_average?.toFixed(1)} overall
          {' '}
          ({realityCheck.recent_trend > 0 ? '▲' : realityCheck.recent_trend < 0 ? '▼' : '–'}
          {Math.abs(realityCheck.recent_trend).toFixed(1)}).
        </ThemedText>
      ) : null}

      {percentages ? (
        <View style={styles.bars}>
          <ThemedText type="small">
            {percentages.matched_or_better?.toFixed(0)}% said it matched or exceeded expectations
          </ThemedText>
          <ExpectationBar label="Better than expected" value={percentages.better} theme={theme} />
          <ExpectationBar label="As expected" value={percentages.as_expected} theme={theme} />
          <ExpectationBar label="Worse than expected" value={percentages.worse} theme={theme} />
          <ExpectationBar label="Very different" value={percentages.very_different} theme={theme} />
        </View>
      ) : realityCheck.social_review_count && realityCheck.social_review_count > 0 ? (
        <ThemedText type="small" themeColor="textSecondary">
          {realityCheck.social_review_count} people shared how it compared to what they expected online.
        </ThemedText>
      ) : null}

      {realityCheck.note ? (
        <ThemedText type="small" themeColor="textSecondary">
          {realityCheck.note}
        </ThemedText>
      ) : null}

      {caveat ? (
        <ThemedText type="small" themeColor="textSecondary">
          {caveat}
        </ThemedText>
      ) : null}
    </View>
  );
}

function ExpectationBar({ label, value, theme }: { label: string; value?: number; theme: ReturnType<typeof useTheme> }) {
  if (value == null) return null;
  return (
    <View style={styles.barRow}>
      <ThemedText type="small" style={styles.barLabel} numberOfLines={1}>
        {label}
      </ThemedText>
      <View style={[styles.track, { backgroundColor: theme.backgroundElement }]}>
        <View style={[styles.fill, { width: `${value}%`, backgroundColor: theme.backgroundSelected }]} />
      </View>
      <ThemedText type="small" themeColor="textSecondary" style={styles.barValue}>
        {value.toFixed(0)}%
      </ThemedText>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    gap: Spacing.two,
  },
  bars: {
    gap: Spacing.one,
    marginTop: Spacing.one,
  },
  barRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.two,
    minHeight: 24,
  },
  barLabel: {
    width: 128,
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
  barValue: {
    width: 36,
    textAlign: 'right',
  },
});
