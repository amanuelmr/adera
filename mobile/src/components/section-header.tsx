import { Pressable, StyleSheet, View } from 'react-native';

import { Icon } from '@/components/icon';
import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';

export type SectionHeaderProps = {
  title: string;
  /** e.g. "See all", with what it opens. */
  action?: { label: string; onPress: () => void };
};

export function SectionHeader({ title, action }: SectionHeaderProps) {
  return (
    <View style={styles.row}>
      <ThemedText type="headline" accessibilityRole="header" style={styles.title}>
        {title}
      </ThemedText>
      {action ? (
        <Pressable onPress={action.onPress} accessibilityRole="link" hitSlop={8} style={styles.action}>
          <ThemedText type="link">{action.label}</ThemedText>
          <Icon name="chevron-right" size={16} color="primary" />
        </Pressable>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: Spacing.two,
  },
  title: {
    flexShrink: 1,
  },
  action: {
    flexDirection: 'row',
    alignItems: 'center',
    minHeight: 44,
  },
});
