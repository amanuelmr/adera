import { Pressable, StyleSheet, View } from 'react-native';

import { Icon } from '@/components/icon';
import { ThemedText } from '@/components/themed-text';
import { Radius, Spacing } from '@/constants/theme';
import { categoryIcon } from '@/features/categories/category-style';
import { useTheme } from '@/hooks/use-theme';

export type CategoryTileProps = {
  name: string;
  code?: string;
  onPress: () => void;
};

/** A category as an icon-led pill, in a horizontal row on Discover. */
export function CategoryTile({ name, code, onPress }: CategoryTileProps) {
  const theme = useTheme();
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={name}
      style={({ pressed }) => [
        styles.tile,
        { backgroundColor: theme.surface, borderColor: theme.border, opacity: pressed ? 0.8 : 1 },
      ]}>
      <View style={[styles.iconWrap, { backgroundColor: theme.primaryContainer }]}>
        <Icon name={categoryIcon(code)} size={18} color="onPrimaryContainer" />
      </View>
      <ThemedText type="smallBold" numberOfLines={1} style={styles.label}>
        {name}
      </ThemedText>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  tile: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.two,
    minHeight: 48,
    borderRadius: Radius.full,
    borderWidth: StyleSheet.hairlineWidth,
    paddingLeft: Spacing.one + Spacing.half,
    paddingRight: Spacing.three,
  },
  iconWrap: {
    width: 36,
    height: 36,
    borderRadius: 18,
    alignItems: 'center',
    justifyContent: 'center',
  },
  label: {
    maxWidth: 200,
  },
});
