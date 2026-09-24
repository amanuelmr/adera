import { StyleSheet, TextInput, View } from 'react-native';

import type { components } from '@/api/schema';
import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { ChipGroup } from '../chip-group';
import { INCENTIVE_TYPE_LABELS, MATERIAL_CONNECTION_LABELS } from '../types';

const INCENTIVE_OPTIONS = (Object.keys(INCENTIVE_TYPE_LABELS) as (keyof typeof INCENTIVE_TYPE_LABELS)[]).map((value) => ({
  value,
  label: INCENTIVE_TYPE_LABELS[value],
}));
const MATERIAL_CONNECTION_OPTIONS = (
  Object.keys(MATERIAL_CONNECTION_LABELS) as (keyof typeof MATERIAL_CONNECTION_LABELS)[]
).map((value) => ({ value, label: MATERIAL_CONNECTION_LABELS[value] }));

export function DisclosureStep({
  incentiveType,
  materialConnection,
  disclosureDetails,
  onChangeIncentiveType,
  onChangeMaterialConnection,
  onChangeDisclosureDetails,
}: {
  incentiveType: components['schemas']['IncentiveType'];
  materialConnection: components['schemas']['MaterialConnection'];
  disclosureDetails: string;
  onChangeIncentiveType: (value: components['schemas']['IncentiveType']) => void;
  onChangeMaterialConnection: (value: components['schemas']['MaterialConnection']) => void;
  onChangeDisclosureDetails: (value: string) => void;
}) {
  const theme = useTheme();
  const needsDetails = incentiveType !== 'none' || materialConnection !== 'none';
  const detailsRequired = incentiveType === 'other' || materialConnection === 'other';

  return (
    <View style={styles.container}>
      <ThemedText type="subtitle">One more thing</ThemedText>
      <ThemedText type="small" themeColor="textSecondary">
        We ask everyone this — it keeps reviews trustworthy. A discount, free item, or relationship with the business gets a
        public label, not a removed review.
      </ThemedText>

      <View style={styles.field}>
        <ThemedText type="smallBold">Did you receive anything for this review?</ThemedText>
        <ChipGroup options={INCENTIVE_OPTIONS} value={incentiveType} onChange={onChangeIncentiveType} />
      </View>

      <View style={styles.field}>
        <ThemedText type="smallBold">Do you have a relationship with this business?</ThemedText>
        <ChipGroup options={MATERIAL_CONNECTION_OPTIONS} value={materialConnection} onChange={onChangeMaterialConnection} />
      </View>

      {needsDetails ? (
        <View style={styles.field}>
          <ThemedText type="smallBold">
            Details{detailsRequired ? '' : ' (optional)'}
          </ThemedText>
          <TextInput
            value={disclosureDetails}
            onChangeText={onChangeDisclosureDetails}
            placeholder="Briefly explain"
            placeholderTextColor={theme.textSecondary}
            style={[styles.input, { color: theme.text, backgroundColor: theme.backgroundElement }]}
          />
        </View>
      ) : null}
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
