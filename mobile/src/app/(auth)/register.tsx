import { Link } from 'expo-router';
import { useState } from 'react';
import { ScrollView, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { useAuth } from '@/auth/context';
import { friendlyAuthError } from '@/auth/friendly-error';
import { Button } from '@/components/button';
import { TextField } from '@/components/text-field';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';

// Email-first for v1 (docs/mobile-plan.md §4.1, §6) — phone verification is
// deferred, so registration collects email rather than a channel picker.
export default function RegisterScreen() {
  const { register } = useAuth();
  const [displayName, setDisplayName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit() {
    setError(undefined);
    setSubmitting(true);
    try {
      // Registration returns an authenticated session immediately (email
      // verification isn't required to read or write, per
      // docs/mobile-plan.md §4.1) — the auth guard in the root layout takes
      // over navigation into the app from here.
      await register({ displayName: displayName.trim(), email: email.trim(), password });
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
          <ThemedText type="subtitle">Create your account</ThemedText>

          <TextField label="Name" value={displayName} onChangeText={setDisplayName} autoComplete="name" textContentType="name" />
          <TextField
            label="Email"
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            autoComplete="email"
            keyboardType="email-address"
            textContentType="emailAddress"
          />
          <TextField
            label="Password"
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            autoComplete="new-password"
            textContentType="newPassword"
            error={error}
          />

          <Button
            title="Create account"
            onPress={handleSubmit}
            loading={submitting}
            disabled={!displayName || !email || !password}
          />

          <Link href="/" style={styles.link}>
            <ThemedText type="link">Already have an account? Sign in</ThemedText>
          </Link>
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
  link: {
    alignSelf: 'center',
  },
});
