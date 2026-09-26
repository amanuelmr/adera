import { useTranslation } from 'react-i18next';
import { Linking, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';

export function UpdateRequiredScreen({ storeUrl }: { storeUrl?: string }) {
  const { t } = useTranslation();
  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea}>
        <ThemedText type="title">{t('updateRequired.title')}</ThemedText>
        <ThemedText type="small" themeColor="textSecondary" style={styles.body}>
          {t('updateRequired.body')}
        </ThemedText>
        {storeUrl ? (
          <Button title={t('common.updateNow')} onPress={() => Linking.openURL(storeUrl)} />
        ) : (
          // The backend can set update_required without a store_url (the
          // schema allows it) — there's no real store listing to fall back
          // to yet (no android.package configured in app.json), so this
          // stays a dead end for now rather than a button that opens
          // nothing or a guessed, possibly-wrong URL. At least tell the
          // user what to do instead of leaving a silent blocking screen.
          <ThemedText type="small" themeColor="textSecondary" style={styles.body}>
            {t('updateRequired.checkStore')}
          </ThemedText>
        )}
      </SafeAreaView>
    </ThemedView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  safeArea: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: Spacing.three,
    padding: Spacing.four,
  },
  body: {
    textAlign: 'center',
  },
});
