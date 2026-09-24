import { Linking, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';

export function UpdateRequiredScreen({ storeUrl }: { storeUrl?: string }) {
  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea}>
        <ThemedText type="title">Update required</ThemedText>
        <ThemedText type="small" themeColor="textSecondary" style={styles.body}>
          This version of Adera is no longer supported. Update to keep using the app.
        </ThemedText>
        {storeUrl ? <Button title="Update now" onPress={() => Linking.openURL(storeUrl)} /> : null}
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
    gap: Spacing.three,
    padding: Spacing.four,
  },
  body: {
    textAlign: 'center',
  },
});
