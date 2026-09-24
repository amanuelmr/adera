import { router, useLocalSearchParams } from 'expo-router';
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

export default function ResetPasswordScreen() {
  const { identifier: identifierParam } = useLocalSearchParams<{ identifier?: string }>();
  const [identifier, setIdentifier] = useState(identifierParam ?? '');
  const [code, setCode] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [error, setError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit() {
    setError(undefined);
    setSubmitting(true);
    try {
      const result = await apiClient.POST('/api/v1/auth/password-reset/confirm', {
        body: { identifier: identifier.trim(), code: code.trim(), new_password: newPassword },
      });
      unwrap(result);
      // Resetting revokes every session (internal/auth), so the user signs
      // in fresh rather than being auto-logged-in here.
      router.replace('/');
    } catch (err) {
      setError(friendlyAuthError(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea}>
        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
          <ThemedText type="subtitle">Enter your code</ThemedText>

          <TextField label="Email or phone" value={identifier} onChangeText={setIdentifier} autoCapitalize="none" />
          <TextField label="6-digit code" value={code} onChangeText={setCode} keyboardType="number-pad" maxLength={6} />
          <TextField
            label="New password"
            value={newPassword}
            onChangeText={setNewPassword}
            secureTextEntry
            autoComplete="new-password"
            textContentType="newPassword"
            error={error}
          />

          <Button
            title="Reset password"
            onPress={handleSubmit}
            loading={submitting}
            disabled={!identifier || !code || !newPassword}
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
  content: {
    flexGrow: 1,
    justifyContent: 'center',
    paddingHorizontal: Spacing.four,
    gap: Spacing.three,
  },
});
