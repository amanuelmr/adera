import { useState } from 'react';
import { ScrollView, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { apiClient, unwrap } from '@/api/client';
import { friendlyAuthError } from '@/auth/friendly-error';
import { Button } from '@/components/button';
import { TextField } from '@/components/text-field';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';

// Email-only for v1 (docs/mobile-plan.md §4.1); confirming isn't required to
// read or write, so there's no gate here — just a code round trip.
export default function VerifyScreen() {
  const [codeRequested, setCodeRequested] = useState(false);
  const [code, setCode] = useState('');
  const [confirmed, setConfirmed] = useState(false);
  const [error, setError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);

  async function requestCode() {
    setError(undefined);
    setSubmitting(true);
    try {
      unwrap(await apiClient.POST('/api/v1/auth/verify/request', { body: { channel: 'email' } }));
      setCodeRequested(true);
    } catch (err) {
      setError(friendlyAuthError(err));
    } finally {
      setSubmitting(false);
    }
  }

  async function confirmCode() {
    setError(undefined);
    setSubmitting(true);
    try {
      unwrap(await apiClient.POST('/api/v1/auth/verify/confirm', { body: { channel: 'email', code: code.trim() } }));
      setConfirmed(true);
    } catch (err) {
      setError(friendlyAuthError(err));
    } finally {
      setSubmitting(false);
    }
  }

  if (confirmed) {
    return (
      <ThemedView style={styles.container}>
        <SafeAreaView style={styles.safeArea}>
          <ScrollView contentContainerStyle={styles.content}>
            <ThemedText type="subtitle">Email verified</ThemedText>
          </ScrollView>
        </SafeAreaView>
      </ThemedView>
    );
  }

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea}>
        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
          <ThemedText type="small" themeColor="textSecondary">
            {codeRequested
              ? 'Enter the 6-digit code we emailed you.'
              : "We'll email you a 6-digit code to confirm your address."}
          </ThemedText>

          {codeRequested ? (
            <>
              <TextField label="6-digit code" value={code} onChangeText={setCode} keyboardType="number-pad" maxLength={6} error={error} />
              <Button title="Confirm" onPress={confirmCode} loading={submitting} disabled={!code} />
              <Button title="Resend code" variant="secondary" onPress={requestCode} loading={submitting} />
            </>
          ) : (
            <Button title="Send code" onPress={requestCode} loading={submitting} />
          )}
          {!codeRequested && error ? <ThemedText style={styles.error}>{error}</ThemedText> : null}
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
    flexGrow: 1,
    justifyContent: 'center',
    paddingHorizontal: Spacing.four,
    gap: Spacing.three,
  },
  error: {
    color: '#D64545',
  },
});
