import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { StyleSheet, TextInput, View } from 'react-native';

import type { components } from '@/api/schema';
import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { ChipGroup } from '../chip-group';
import { dateChoiceFor, daysAgo, type DateChoice } from '../form-steps';
import { DISCOVERY_SOURCES, EXPECTATION_MATCHES, SOCIAL_DISCOVERY_SOURCES } from '../types';
import { experienceDateError, parsePrice } from '../validation';

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
  const dateError = experienceDateError(experienceDate);
  const priceError = parsePrice(pricePaid).error;
  const isSocial = discoverySource ? SOCIAL_DISCOVERY_SOURCES.has(discoverySource) : false;
  // "Pick a date" with nothing typed yet has no stored value to derive from.
  const [picking, setPicking] = useState(false);
  const dateChoice = dateChoiceFor(experienceDate) ?? (picking ? 'pick' : undefined);
  const dateOptions: { value: DateChoice; label: string }[] = [
    { value: 'today', label: t('review.dateToday') },
    { value: 'yesterday', label: t('review.dateYesterday') },
    { value: 'pick', label: t('review.datePick') },
  ];

  function chooseDate(choice: DateChoice) {
    // The date is optional: tapping the selected chip again clears it.
    if (choice === dateChoice) {
      setPicking(false);
      onChangeExperienceDate('');
      return;
    }
    setPicking(choice === 'pick');
    onChangeExperienceDate(choice === 'today' ? daysAgo(0) : choice === 'yesterday' ? daysAgo(1) : '');
  }

  return (
    <View style={styles.container}>
      <ThemedText type="subtitle">{t('review.contextTitle')}</ThemedText>

      <View style={styles.field}>
        <ThemedText type="smallBold">{t('review.whenLabel')}</ThemedText>
        <ChipGroup options={dateOptions} value={dateChoice} onChange={chooseDate} />
        {dateChoice === 'pick' ? (
          <TextInput
            value={experienceDate}
            onChangeText={onChangeExperienceDate}
            placeholder="YYYY-MM-DD"
            placeholderTextColor={theme.textSecondary}
            keyboardType="numbers-and-punctuation"
            autoFocus={picking && !experienceDate}
            accessibilityLabel={t('review.dateFieldLabel')}
            accessibilityHint={t('review.dateFormat')}
            style={[styles.input, { color: theme.text, backgroundColor: theme.backgroundElement }]}
          />
        ) : null}
        {dateError ? (
          <ThemedText type="small" style={styles.error}>
            {t(dateError === 'future' ? 'review.dateFuture' : 'review.dateFormat')}
          </ThemedText>
        ) : null}
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
          accessibilityLabel={t('review.priceLabel')}
          style={[styles.input, { color: theme.text, backgroundColor: theme.backgroundElement }]}
        />
        {priceError ? (
          <ThemedText type="small" style={styles.error}>
            {t('review.priceInvalid')}
          </ThemedText>
        ) : null}
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
  error: {
    color: '#D64545',
  },
  input: {
    minHeight: 44,
    borderRadius: Spacing.two,
    paddingHorizontal: Spacing.three,
    fontSize: 16,
  },
});
