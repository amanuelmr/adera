import { useLocalSearchParams } from 'expo-router';
import { StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';

// Placeholder — replaced by the target profile screen (docs/mobile-plan.md
// §9 Phase 1) in a later feature slice. Exists now so Discover's target
// cards have a real destination rather than a dangling route.
export default function TargetScreen() {
  const { idOrSlug } = useLocalSearchParams<{ idOrSlug: string }>();

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <ThemedText type="small" themeColor="textSecondary">
          Target profile for &ldquo;{idOrSlug}&rdquo; is coming soon.
        </ThemedText>
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
    padding: Spacing.four,
  },
});
