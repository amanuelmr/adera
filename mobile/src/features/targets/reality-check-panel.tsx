import { useTranslation } from 'react-i18next';
import { StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import type { components } from '@/api/schema';

type RealityCheck = components['schemas']['RealityCheck'];

// Only small samples get a caveat (confidence from internal/ratings).
const CAVEAT_KEYS: Partial<Record<NonNullable<RealityCheck['confidence']>, string>> = {
  none: 'realityCheck.caveatNone',
  low: 'realityCheck.caveatLow',
};

export function RealityCheckPanel({ realityCheck }: { realityCheck: RealityCheck }) {
  const theme = useTheme();
  const { t } = useTranslation();
  const expectations = realityCheck.expectation_distribution;
  const percentages = expectations?.percentages;
  const caveatKey = realityCheck.confidence ? CAVEAT_KEYS[realityCheck.confidence] : undefined;
  const socialCount = realityCheck.social_review_count ?? 0;

  return (
    <View style={styles.container}>
      <ThemedText type="smallBold">{t('realityCheck.title')}</ThemedText>
      <ThemedText type="small" themeColor="textSecondary">
        {t('realityCheck.question')}
      </ThemedText>

      {realityCheck.trend_available && realityCheck.recent_trend != null ? (
        <ThemedText type="small">
          {t('realityCheck.trend', {
            recent: realityCheck.recent_average?.toFixed(1),
            overall: realityCheck.historical_average?.toFixed(1),
            arrow: realityCheck.recent_trend > 0 ? '▲' : realityCheck.recent_trend < 0 ? '▼' : '–',
            delta: Math.abs(realityCheck.recent_trend).toFixed(1),
          })}
        </ThemedText>
      ) : null}

      {percentages ? (
        <View style={styles.bars}>
          <ThemedText type="small">
            {t('realityCheck.matched', { percent: percentages.matched_or_better?.toFixed(0) })}
          </ThemedText>
          <ExpectationBar label={t('review.expectation.better')} value={percentages.better} theme={theme} />
          <ExpectationBar label={t('review.expectation.as_expected')} value={percentages.as_expected} theme={theme} />
          <ExpectationBar label={t('review.expectation.worse')} value={percentages.worse} theme={theme} />
          <ExpectationBar label={t('review.expectation.very_different')} value={percentages.very_different} theme={theme} />
        </View>
      ) : (
        // Below 5 social-discovery answers the API withholds percentages
        // (internal/ratings); its English `note` explains that, so the same
        // facts are stated here in the user's language instead.
        <ThemedText type="small" themeColor="textSecondary">
          {socialCount > 0 ? t('realityCheck.notEnough', { count: socialCount }) : t('realityCheck.noneYet')}
        </ThemedText>
      )}

      {caveatKey ? (
        <ThemedText type="small" themeColor="textSecondary">
          {t(caveatKey)}
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
