import { Link } from 'expo-router';
import { StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { useAuth } from '@/auth/context';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';

export default function AccountScreen() {
  const { logout, logoutAll } = useAuth();

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <Link href="/verify" style={styles.link}>
          <ThemedText type="link">Verify your email</ThemedText>
        </Link>
        <Link href="/sessions" style={styles.link}>
          <ThemedText type="link">Your devices</ThemedText>
        </Link>

        <ThemedView style={styles.actions}>
          <Button title="Sign out" variant="secondary" onPress={() => logout()} />
          <Button title="Sign out everywhere" variant="secondary" onPress={() => logoutAll()} />
        </ThemedView>
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
    padding: Spacing.four,
    gap: Spacing.two,
  },
  link: {
    paddingVertical: Spacing.one,
  },
  actions: {
    marginTop: Spacing.five,
    gap: Spacing.two,
  },
});
