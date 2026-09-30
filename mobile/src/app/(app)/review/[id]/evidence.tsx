import { useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ActivityIndicator, ScrollView, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ApiError } from '@/api/client';
import { Button } from '@/components/button';
import { QueryError } from '@/components/query-error';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { FilterChip } from '@/features/search/filter-chip';
import { pickAndCompressPhoto } from '@/features/reviews/photos';
import { useEvidence, useSubmitEvidence, type EvidenceKind } from '@/features/reviews/evidence-queries';

const MAX_EVIDENCE = 5;

const KIND_OPTIONS: { value: NonNullable<EvidenceKind>; labelKey: string }[] = [
  { value: 'receipt', labelKey: 'evidence.kindReceipt' },
  { value: 'order_screenshot', labelKey: 'evidence.kindOrderScreenshot' },
  { value: 'product_photo', labelKey: 'evidence.kindProductPhoto' },
  { value: 'service_result', labelKey: 'evidence.kindServiceResult' },
  { value: 'location_qr', labelKey: 'evidence.kindLocationQr' },
];

const STATUS_LABEL_KEYS: Record<string, string> = {
  staged: 'evidence.statusStaged',
  submitted: 'evidence.statusSubmitted',
  accepted: 'evidence.statusAccepted',
  rejected: 'evidence.statusRejected',
};

const KIND_LABEL_KEYS: Record<string, string> = Object.fromEntries(KIND_OPTIONS.map((option) => [option.value, option.labelKey]));

export default function EvidenceScreen() {
  const { id: reviewId } = useLocalSearchParams<{ id: string }>();
  const { t } = useTranslation();
  const evidence = useEvidence(reviewId);
  const submitEvidence = useSubmitEvidence(reviewId);

  const [kind, setKind] = useState<NonNullable<EvidenceKind>>('receipt');
  const [pickError, setPickError] = useState<string>();

  const items = evidence.data ?? [];
  const atLimit = items.length >= MAX_EVIDENCE;

  async function addEvidence() {
    setPickError(undefined);
    const picked = await pickAndCompressPhoto();
    if ('error' in picked) {
      setPickError(picked.error);
      return;
    }
    if ('canceled' in picked) return;
    submitEvidence.mutate({ kind, localUri: picked.uri });
  }

  const submitError = submitEvidence.error
    ? submitEvidence.error instanceof ApiError
      ? submitEvidence.error.message
      : t('errors.generic')
    : undefined;

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <ScrollView contentContainerStyle={styles.content}>
          <ThemedText type="small" themeColor="textSecondary">
            {t('evidence.privacyNotice')}
          </ThemedText>
          <ThemedText type="small" themeColor="textSecondary">
            {t('evidence.body')}
          </ThemedText>

          <View style={styles.kindRow}>
            {KIND_OPTIONS.map((option) => (
              <FilterChip
                key={option.value}
                label={t(option.labelKey)}
                selected={kind === option.value}
                onPress={() => setKind(option.value)}
              />
            ))}
          </View>

          {atLimit ? (
            <ThemedText type="small" themeColor="textSecondary">
              {t('evidence.limitReached')}
            </ThemedText>
          ) : (
            <Button title={t('evidence.addButton')} onPress={addEvidence} loading={submitEvidence.isPending} />
          )}

          {pickError ? (
            <ThemedText type="small" style={styles.error}>
              {pickError}
            </ThemedText>
          ) : null}
          {submitError ? (
            <ThemedText type="small" style={styles.error}>
              {submitError}
            </ThemedText>
          ) : null}

          <View style={styles.list}>
            {evidence.isPending ? (
              <ActivityIndicator />
            ) : evidence.isError ? (
              <QueryError onRetry={() => evidence.refetch()} retrying={evidence.isRefetching} />
            ) : items.length === 0 ? (
              <ThemedText type="small" themeColor="textSecondary">
                {t('evidence.empty')}
              </ThemedText>
            ) : (
              items.map((item) => (
                <ThemedView key={item.id} type="backgroundElement" style={styles.item}>
                  <ThemedText type="small">{item.kind && KIND_LABEL_KEYS[item.kind] ? t(KIND_LABEL_KEYS[item.kind]) : item.kind}</ThemedText>
                  <ThemedText type="small" themeColor="textSecondary">
                    {item.status && STATUS_LABEL_KEYS[item.status] ? t(STATUS_LABEL_KEYS[item.status]) : item.status}
                  </ThemedText>
                </ThemedView>
              ))
            )}
          </View>
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
    gap: Spacing.three,
  },
  kindRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing.two,
  },
  error: {
    color: '#D64545',
  },
  list: {
    gap: Spacing.two,
    marginTop: Spacing.two,
  },
  item: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: Spacing.three,
    borderRadius: Spacing.two,
  },
});
