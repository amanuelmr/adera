import { Link } from 'expo-router';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ScrollView, StyleSheet, Switch, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { useAuth } from '@/auth/context';
import { friendlyAuthError } from '@/auth/friendly-error';
import { Button } from '@/components/button';
import { TextField } from '@/components/text-field';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { useVersionGate } from '@/features/app-version/queries';
import { openPrivacyNotice } from '@/features/privacy/notice';

// Email-first for v1 (docs/mobile-plan.md §4.1, §6) — phone verification is
// deferred, so registration collects email rather than a channel picker.
export default function RegisterScreen() {
  const { register } = useAuth();
  const { t, i18n } = useTranslation();
  const versionGate = useVersionGate();
  const [acceptedPrivacy, setAcceptedPrivacy] = useState(false);
  const [displayName, setDisplayName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string>();
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit() {
    setError(undefined);
    setSubmitting(true);
    try {
      // The server publishes which notice is current; accepting is
      // accepting that version (the page the link opens).
      const privacyNoticeVersion =
        versionGate.data?.privacy_notice_version ?? (await versionGate.refetch()).data?.privacy_notice_version;
      if (!privacyNoticeVersion) {
        setError(t('auth.privacyUnavailable'));
        return;
      }
      // Registration returns an authenticated session immediately (email
      // verification isn't required to read or write, per
      // docs/mobile-plan.md §4.1) — the auth guard in the root layout takes
      // over navigation into the app from here.
      await register({ displayName: displayName.trim(), email: email.trim(), password, privacyNoticeVersion });
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
          <ThemedText type="subtitle">{t('auth.createAccountTitle')}</ThemedText>

          <TextField
            label={t('auth.name')}
            value={displayName}
            onChangeText={setDisplayName}
            autoComplete="name"
            textContentType="name"
          />
          <TextField
            label={t('auth.email')}
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            autoComplete="email"
            keyboardType="email-address"
            textContentType="emailAddress"
          />
          <TextField
            label={t('auth.password')}
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            autoComplete="new-password"
            textContentType="newPassword"
            error={error}
          />

          <View style={styles.consentRow}>
            <Switch
              value={acceptedPrivacy}
              onValueChange={setAcceptedPrivacy}
              accessibilityLabel={t('auth.acceptPrivacy')}
            />
            <ThemedText type="small" style={styles.consentText}>
              {t('auth.acceptPrivacy')}
            </ThemedText>
          </View>
          <ThemedText
            type="link"
            accessibilityRole="link"
            onPress={() => void openPrivacyNotice(i18n.language)}
            style={styles.privacyLink}>
            {t('auth.readPrivacy')}
          </ThemedText>

          <Button
            title={t('common.createAccount')}
            onPress={handleSubmit}
            loading={submitting}
            disabled={!displayName || !email || !password || !acceptedPrivacy}
          />

          <Link href="/sign-in" style={styles.link}>
            <ThemedText type="link">{t('auth.haveAccount')}</ThemedText>
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
  consentRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.three,
  },
  consentText: {
    flex: 1,
  },
  privacyLink: {
    minHeight: 44,
    textAlignVertical: 'center',
  },
});
