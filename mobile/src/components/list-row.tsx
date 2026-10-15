import { Pressable, StyleSheet, View } from 'react-native';

import { Icon, type IconName } from '@/components/icon';
import { ThemedText } from '@/components/themed-text';
import { Radius, Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';

export type ListRowProps = {
  icon: IconName;
  label: string;
  detail?: string;
  onPress: () => void;
  destructive?: boolean;
};

/** A settings-style row: tinted icon, label, optional detail, chevron. */
export function ListRow({ icon, label, detail, onPress, destructive }: ListRowProps) {
  const theme = useTheme();
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={detail ? `${label}, ${detail}` : label}
      style={({ pressed }) => [styles.row, pressed && { backgroundColor: theme.backgroundElement }]}>
      <View style={[styles.iconWrap, { backgroundColor: destructive ? theme.backgroundElement : theme.primaryContainer }]}>
        <Icon name={icon} size={18} color={destructive ? 'danger' : 'onPrimaryContainer'} />
      </View>
      <View style={styles.text}>
        <ThemedText themeColor={destructive ? 'danger' : 'text'}>{label}</ThemedText>
        {detail ? (
          <ThemedText type="small" themeColor="textSecondary">
            {detail}
          </ThemedText>
        ) : null}
      </View>
      {destructive ? null : <Icon name="chevron-right" size={18} color="textSecondary" />}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.twoHalf,
    minHeight: 56,
    paddingHorizontal: Spacing.twoHalf,
    paddingVertical: Spacing.two,
    borderRadius: Radius.sm,
  },
  iconWrap: {
    width: 32,
    height: 32,
    borderRadius: Radius.sm,
    alignItems: 'center',
    justifyContent: 'center',
  },
  text: {
    flex: 1,
    gap: Spacing.half,
  },
});
