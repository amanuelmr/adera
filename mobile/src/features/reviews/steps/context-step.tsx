import { useTranslation } from 'react-i18next';
import { StyleSheet, TextInput, View } from 'react-native';

import type { components } from '@/api/schema';
import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { ChipGroup } from '../chip-group';
import { DISCOVERY_SOURCES, EXPECTATION_MATCHES, SOCIAL_DISCOVERY_SOURCES } from '../types';

export function ContextStep({
  experienceDate,
  discoverySource,
  expectationMatch,
  pricePaid,
  onChangeExperienceDate,
  onChangeDiscoverySource,
  onChangeExpectationMatch,
  onChangePricePaid,
}: {
  experienceDate: string;
  discoverySource?: components['schemas']['DiscoverySource'];
  expectationMatch?: components['schemas']['ExpectationMatch'];
  pricePaid: string;
  onChangeExperienceDate: (value: string) => void;
  onChangeDiscoverySource: (value: components['schemas']['DiscoverySource']) => void;
  onChangeExpectationMatch: (value: components['schemas']['ExpectationMatch'] | undefined) => void;
  onChangePricePaid: (value: string) => void;
}) {
  const theme = useTheme();
  const { t } = useTranslation();
  const discoveryOptions = DISCOVERY_SOURCES.map((value) => ({ value, label: t(`review.discovery.${value}`) }));
  const expectationOptions = EXPECTATION_MATCHES.map((value) => ({ value, label: t(`review.expectation.${value}`) }));
  const isSocial = discoverySource ? SOCIAL_DISCOVERY_SOURCES.has(discoverySource) : false;

  return (
    <View style={styles.container}>
      <ThemedText type="subtitle">{t('review.contextTitle')}</ThemedText>

      <View style={styles.field}>
        <ThemedText type="smallBold">{t('review.whenLabel')}</ThemedText>
        <TextInput
          value={experienceDate}
          onChangeText={onChangeExperienceDate}
          placeholder="YYYY-MM-DD"
          placeholderTextColor={theme.textSecondary}
          keyboardType="numbers-and-punctuation"
          style={[styles.input, { color: theme.text, backgroundColor: theme.backgroundElement }]}
        />
      </View>

      <View style={styles.field}>
        <ThemedText type="smallBold">{t('review.discoveryLabel')}</ThemedText>
        <ChipGroup
          options={discoveryOptions}
          value={discoverySource}
          onChange={(value) => {
            onChangeDiscoverySource(value);
            if (!SOCIAL_DISCOVERY_SOURCES.has(value)) onChangeExpectationMatch(undefined);
          }}
        />
      </View>

      {isSocial ? (
        <View style={styles.field}>
          <ThemedText type="smallBold">{t('review.expectationLabel')}</ThemedText>
          <ChipGroup options={expectationOptions} value={expectationMatch} onChange={onChangeExpectationMatch} />
        </View>
      ) : null}

      <View style={styles.field}>
        <ThemedText type="smallBold">{t('review.priceLabel')}</ThemedText>
        <TextInput
          value={pricePaid}
          onChangeText={onChangePricePaid}
          placeholder="ETB"
          placeholderTextColor={theme.textSecondary}
          keyboardType="numeric"
          style={[styles.input, { color: theme.text, backgroundColor: theme.backgroundElement }]}
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    gap: Spacing.four,
  },
  field: {
    gap: Spacing.two,
  },
  input: {
    minHeight: 44,
    borderRadius: Spacing.two,
    paddingHorizontal: Spacing.three,
    fontSize: 16,
  },
});
