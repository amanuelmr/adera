import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ScrollView, StyleSheet, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ApiError } from '@/api/client';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { FilterChip } from '@/features/search/filter-chip';
import { MAX_REPORT_DETAILS, REPORT_REASONS, useReportReview, type ReportReason } from '@/features/reviews/report-queries';

export default function ReportReviewScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { t } = useTranslation();
  const theme = useTheme();
  const report = useReportReview(id);
  const [reason, setReason] = useState<ReportReason>();
  const [details, setDetails] = useState('');

  if (report.isSuccess) {
    return (
      <ThemedView style={styles.centered}>
        <ThemedText type="subtitle">{t('report.sent')}</ThemedText>
        <Button title={t('common.back')} onPress={() => router.back()} />
      </ThemedView>
    );
  }

  const error = report.error
    ? report.error instanceof ApiError && report.error.status === 409
      ? t('report.already')
      : report.error instanceof ApiError && report.error.status === 429
        ? t('report.rateLimited')
        : t('errors.generic')
    : undefined;

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
          <ThemedText type="small" themeColor="textSecondary">
            {t('report.body')}
          </ThemedText>

          <ThemedText type="smallBold">{t('report.reasonLabel')}</ThemedText>
          <View style={styles.reasons}>
            {REPORT_REASONS.map((value) => (
              <FilterChip
                key={value}
                label={t(`report.reasons.${value}`)}
                selected={reason === value}
                onPress={() => setReason(value)}
              />
            ))}
          </View>

          <ThemedText type="smallBold">{t('report.detailsLabel')}</ThemedText>
          <TextInput
            value={details}
            onChangeText={setDetails}
            placeholder={t('report.detailsPlaceholder')}
            placeholderTextColor={theme.textSecondary}
            multiline
            maxLength={MAX_REPORT_DETAILS}
            accessibilityLabel={t('report.detailsLabel')}
            style={[styles.input, { color: theme.text, backgroundColor: theme.backgroundElement }]}
          />

          {error ? (
            <ThemedText type="small" style={styles.error}>
              {error}
            </ThemedText>
          ) : null}

          <Button
            title={t('report.submit')}
            disabled={!reason}
            loading={report.isPending}
            onPress={() => reason && report.mutate({ reason, details })}
          />
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
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: Spacing.three,
    padding: Spacing.four,
  },
  content: {
    padding: Spacing.four,
    gap: Spacing.three,
  },
  reasons: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing.two,
  },
  input: {
    minHeight: 100,
    borderRadius: Spacing.two,
    padding: Spacing.three,
    fontSize: 16,
    textAlignVertical: 'top',
  },
  error: {
    color: '#D64545',
  },
});
