import { StyleSheet, TextInput, View } from 'react-native';

import type { components } from '@/api/schema';
import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { ChipGroup } from '../chip-group';
import { DISCOVERY_SOURCE_LABELS, EXPECTATION_MATCH_LABELS, SOCIAL_DISCOVERY_SOURCES } from '../types';

const DISCOVERY_SOURCE_OPTIONS = (Object.keys(DISCOVERY_SOURCE_LABELS) as (keyof typeof DISCOVERY_SOURCE_LABELS)[]).map(
  (value) => ({ value, label: DISCOVERY_SOURCE_LABELS[value] })
);
const EXPECTATION_MATCH_OPTIONS = (Object.keys(EXPECTATION_MATCH_LABELS) as (keyof typeof EXPECTATION_MATCH_LABELS)[]).map(
  (value) => ({ value, label: EXPECTATION_MATCH_LABELS[value] })
);

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
  const isSocial = discoverySource ? SOCIAL_DISCOVERY_SOURCES.has(discoverySource) : false;

  return (
    <View style={styles.container}>
      <ThemedText type="subtitle">A bit of context</ThemedText>

      <View style={styles.field}>
        <ThemedText type="smallBold">When was this? (optional)</ThemedText>
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
        <ThemedText type="smallBold">How did you find this place?</ThemedText>
        <ChipGroup
          options={DISCOVERY_SOURCE_OPTIONS}
          value={discoverySource}
          onChange={(value) => {
            onChangeDiscoverySource(value);
            if (!SOCIAL_DISCOVERY_SOURCES.has(value)) onChangeExpectationMatch(undefined);
          }}
        />
      </View>

      {isSocial ? (
        <View style={styles.field}>
          <ThemedText type="smallBold">Did it match what you saw online?</ThemedText>
          <ChipGroup options={EXPECTATION_MATCH_OPTIONS} value={expectationMatch} onChange={onChangeExpectationMatch} />
        </View>
      ) : null}

      <View style={styles.field}>
        <ThemedText type="smallBold">Price paid (optional)</ThemedText>
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
