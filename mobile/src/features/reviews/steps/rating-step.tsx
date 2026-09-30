import { useTranslation } from 'react-i18next';
import { StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { StarPicker } from '../star-picker';

export function RatingStep({ value, onChange }: { value?: number; onChange: (rating: number) => void }) {
  const { t } = useTranslation();
  return (
    <View style={styles.container}>
      <ThemedText type="subtitle">{t('review.ratingTitle')}</ThemedText>
      <StarPicker value={value} onChange={onChange} size={40} label={t('review.ratingLabel')} />
      {value ? <ThemedText themeColor="textSecondary">{t(`review.rating.${value}`)}</ThemedText> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    gap: Spacing.three,
    alignItems: 'center',
  },
});
