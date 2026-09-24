import { Link } from 'expo-router';
import { StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { useAuth } from '@/auth/context';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';

// Placeholder landing screen for a signed-in user — replaced by the real
// Discover screen (docs/mobile-plan.md §9 Phase 1) in the next feature slice.
export default function HomeScreen() {
  const { logout, logoutAll } = useAuth();

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea}>
        <ThemedText type="title">አደራ</ThemedText>
        <ThemedText type="subtitle" themeColor="textSecondary">
          You&apos;re signed in
        </ThemedText>

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
    alignItems: 'center',
    justifyContent: 'center',
    gap: Spacing.two,
    paddingHorizontal: Spacing.four,
  },
  link: {
    marginTop: Spacing.two,
  },
  actions: {
    marginTop: Spacing.five,
    gap: Spacing.two,
    alignSelf: 'stretch',
  },
});
