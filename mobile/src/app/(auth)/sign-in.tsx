import { Link, router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Pressable, ScrollView, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { useAuth } from '@/auth/context';
import { friendlyAuthError } from '@/auth/friendly-error';
import { isSignInReason } from '@/auth/pending-intent';
import { Button } from '@/components/button';
import { TextField } from '@/components/text-field';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';

export default function LoginScreen() {
  const { login } = useAuth();
  const { reason } = useLocalSearchParams<{ reason?: string }>();
  const { t } = useTranslation();
  const [identifier, setIdentifier] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit() {
    setError(undefined);
    setSubmitting(true);
    try {
      await login(identifier.trim(), password);
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
          <ThemedText type="title">አደራ</ThemedText>
          <ThemedText type="subtitle" themeColor="textSecondary" style={styles.subtitle}>
            {t('auth.signInTitle')}
          </ThemedText>
          {isSignInReason(reason) ? (
            <ThemedText type="small" themeColor="textSecondary">
              {t(`auth.reason.${reason}`)}
            </ThemedText>
          ) : null}

          <TextField
            label={t('auth.emailOrPhone')}
            value={identifier}
            onChangeText={setIdentifier}
            autoCapitalize="none"
            autoComplete="username"
            keyboardType="email-address"
            textContentType="username"
          />
          <TextField
            label={t('auth.password')}
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            autoComplete="password"
            textContentType="password"
            error={error}
          />

          <Button title={t('common.signIn')} onPress={handleSubmit} loading={submitting} disabled={!identifier || !password} />

          <Link href="/forgot-password" style={styles.link}>
            <ThemedText type="link">{t('auth.forgotPassword')}</ThemedText>
          </Link>
          <Link href="/register" style={styles.link}>
            <ThemedText type="link">{t('auth.noAccount')}</ThemedText>
          </Link>
          {router.canGoBack() ? (
            <Pressable onPress={() => router.back()} accessibilityRole="button" style={styles.link}>
              <ThemedText type="link">{t('auth.continueBrowsing')}</ThemedText>
            </Pressable>
          ) : null}
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
  subtitle: {
    marginBottom: Spacing.two,
  },
  link: {
    alignSelf: 'center',
  },
});
