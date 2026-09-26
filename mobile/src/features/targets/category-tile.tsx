import { Pressable, StyleSheet } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';

export type CategoryTileProps = {
  name: string;
  onPress: () => void;
};

export function CategoryTile({ name, onPress }: CategoryTileProps) {
  return (
    <Pressable onPress={onPress} accessibilityRole="button" accessibilityLabel={name}>
      {({ pressed }) => (
        <ThemedView type="backgroundElement" style={[styles.tile, { opacity: pressed ? 0.8 : 1 }]}>
          <ThemedText type="smallBold" style={styles.text} numberOfLines={2}>
            {name}
          </ThemedText>
        </ThemedView>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  tile: {
    width: 120,
    minHeight: 64,
    borderRadius: Spacing.two,
    padding: Spacing.three,
    alignItems: 'center',
    justifyContent: 'center',
  },
  text: {
    textAlign: 'center',
  },
});
