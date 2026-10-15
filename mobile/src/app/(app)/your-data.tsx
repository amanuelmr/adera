import { router } from 'expo-router';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, ScrollView, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { useAuth } from '@/auth/context';
import { friendlyAuthError } from '@/auth/friendly-error';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { shareDataExport } from '@/features/privacy/export';
import { openPrivacyNotice } from '@/features/privacy/notice';
import { formatDate } from '@/lib/format';

// The in-app side of the privacy notice's "Your rights": see the notice, get
// a copy of your data, erase your account. Google Play requires deletion to
// be reachable in the app, not only by email.
export default function YourDataScreen() {
  const { t, i18n } = useTranslation();
  const { deleteAccount } = useAuth();
  const [exporting, setExporting] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string>();

  async function handleExport() {
    setError(undefined);
    setExporting(true);
    try {
      await shareDataExport(t('data.exportTitle'));
    } catch (err) {
      setError(friendlyAuthError(err));
    } finally {
      setExporting(false);
    }
  }

  function confirmDelete() {
    Alert.alert(t('data.deleteConfirmTitle'), t('data.deleteConfirmBody'), [
      { text: t('common.cancel'), style: 'cancel' },
      { text: t('data.deleteConfirm'), style: 'destructive', onPress: () => void handleDelete() },
    ]);
  }

  async function handleDelete() {
    setError(undefined);
    setDeleting(true);
    try {
      const scheduledFor = await deleteAccount();
      Alert.alert(
        t('data.deletedTitle'),
        t('data.deletedBody', { date: scheduledFor ? formatDate(scheduledFor) : '' })
      );
      router.replace('/');
    } catch (err) {
      setError(friendlyAuthError(err));
      setDeleting(false);
    }
  }

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <ScrollView contentContainerStyle={styles.content}>
          <View style={styles.section}>
            <ThemedText type="smallBold">{t('data.noticeTitle')}</ThemedText>
            <ThemedText type="small" themeColor="textSecondary">
              {t('data.noticeBody')}
            </ThemedText>
            <Button
              title={t('data.readNotice')}
              variant="secondary"
              onPress={() => void openPrivacyNotice(i18n.language)}
            />
          </View>

          <View style={styles.section}>
            <ThemedText type="smallBold">{t('data.exportHeading')}</ThemedText>
            <ThemedText type="small" themeColor="textSecondary">
              {t('data.exportBody')}
            </ThemedText>
            <Button title={t('data.export')} variant="secondary" onPress={handleExport} loading={exporting} />
          </View>

          <View style={styles.section}>
            <ThemedText type="smallBold">{t('data.deleteHeading')}</ThemedText>
            <ThemedText type="small" themeColor="textSecondary">
              {t('data.deleteBody')}
            </ThemedText>
            <Button title={t('data.delete')} variant="destructive" icon="trash" onPress={confirmDelete} loading={deleting} />
          </View>

          {error ? <ThemedText themeColor="danger">{error}</ThemedText> : null}
        </ScrollView>
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
  },
  content: {
    padding: Spacing.four,
    gap: Spacing.five,
  },
  section: {
    gap: Spacing.two,
  },
});
