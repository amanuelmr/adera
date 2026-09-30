import { useTranslation } from 'react-i18next';
import { StyleSheet, View } from 'react-native';

import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';

/**
 * A failed load, shown with a retry — distinct from an empty result, so a
 * network error never reads as "you have none".
 */
export function QueryError({ onRetry, retrying }: { onRetry: () => void; retrying?: boolean }) {
  const { t } = useTranslation();
  return (
    <View style={styles.container}>
      <ThemedText type="small" themeColor="textSecondary">
        {t('errors.loadFailed')}
      </ThemedText>
      <Button title={t('common.retry')} variant="secondary" onPress={onRetry} loading={retrying} />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    padding: Spacing.four,
    gap: Spacing.two,
    alignItems: 'flex-start',
  },
});
