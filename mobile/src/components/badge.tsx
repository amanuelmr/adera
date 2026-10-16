import { StyleSheet, View } from 'react-native';

import { Icon, type IconName } from '@/components/icon';
import { ThemedText } from '@/components/themed-text';
import { Radius, Spacing, type ThemeColor } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';

export type BadgeTone = 'verified' | 'neutral' | 'warning';

const TONES: Record<BadgeTone, { background: ThemeColor; foreground: ThemeColor }> = {
  verified: { background: 'primaryContainer', foreground: 'onPrimaryContainer' },
  neutral: { background: 'backgroundElement', foreground: 'textSecondary' },
  warning: { background: 'backgroundElement', foreground: 'warning' },
};

export type BadgeProps = {
  label: string;
  icon?: IconName;
  tone?: BadgeTone;
};

/** A small labelled marker: "Verified visit", "Owner replied", a category. */
export function Badge({ label, icon, tone = 'neutral' }: BadgeProps) {
  const theme = useTheme();
  const { background, foreground } = TONES[tone];
  return (
    <View style={[styles.badge, { backgroundColor: theme[background] }]}>
      {icon ? <Icon name={icon} size={14} color={foreground} /> : null}
      <ThemedText type="caption" themeColor={foreground}>
        {label}
      </ThemedText>
    </View>
  );
}

const styles = StyleSheet.create({
  badge: {
    flexDirection: 'row',
    alignItems: 'center',
    alignSelf: 'flex-start',
    gap: Spacing.one,
    paddingHorizontal: Spacing.two,
    paddingVertical: Spacing.half,
    borderRadius: Radius.full,
  },
});
