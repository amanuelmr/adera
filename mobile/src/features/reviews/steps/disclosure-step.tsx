import { useTranslation } from 'react-i18next';
import { StyleSheet, TextInput, View } from 'react-native';

import type { components } from '@/api/schema';
import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { ChipGroup } from '../chip-group';
import { INCENTIVE_TYPES, MATERIAL_CONNECTIONS } from '../types';

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
  const { t } = useTranslation();
  const incentiveOptions = INCENTIVE_TYPES.map((value) => ({ value, label: t(`disclosure.incentiveOption.${value}`) }));
  const connectionOptions = MATERIAL_CONNECTIONS.map((value) => ({
    value,
    label: t(`disclosure.connectionOption.${value}`),
  }));
  const needsDetails = incentiveType !== 'none' || materialConnection !== 'none';
  const detailsRequired = incentiveType === 'other' || materialConnection === 'other';

  return (
    <View style={styles.container}>
      <ThemedText type="subtitle">{t('disclosure.title')}</ThemedText>
      <ThemedText type="small" themeColor="textSecondary">
        {t('disclosure.body')}
      </ThemedText>

      <View style={styles.field}>
        <ThemedText type="smallBold">{t('disclosure.incentiveQuestion')}</ThemedText>
        <ChipGroup options={incentiveOptions} value={incentiveType} onChange={onChangeIncentiveType} />
      </View>

      <View style={styles.field}>
        <ThemedText type="smallBold">{t('disclosure.relationshipQuestion')}</ThemedText>
        <ChipGroup options={connectionOptions} value={materialConnection} onChange={onChangeMaterialConnection} />
      </View>

      {needsDetails ? (
        <View style={styles.field}>
          <ThemedText type="smallBold">{t(detailsRequired ? 'disclosure.details' : 'disclosure.detailsOptional')}</ThemedText>
          <TextInput
            value={disclosureDetails}
            onChangeText={onChangeDisclosureDetails}
            placeholder={t('disclosure.detailsPlaceholder')}
            placeholderTextColor={theme.textSecondary}
            accessibilityLabel={t(detailsRequired ? 'disclosure.details' : 'disclosure.detailsOptional')}
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
