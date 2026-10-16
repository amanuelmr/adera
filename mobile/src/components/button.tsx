import { ActivityIndicator, Pressable, StyleSheet, type PressableProps } from 'react-native';

import { Icon, type IconName } from '@/components/icon';
import { ThemedText } from '@/components/themed-text';
import { Radius, Spacing, type ThemeColor } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'destructive';

const VARIANTS: Record<ButtonVariant, { background?: ThemeColor; foreground: ThemeColor }> = {
  // The one main action on a screen.
  primary: { background: 'primary', foreground: 'onPrimary' },
  // Supporting actions: a tinted fill, not competing with primary.
  secondary: { background: 'primaryContainer', foreground: 'onPrimaryContainer' },
  // Low-emphasis actions in rows and cards.
  ghost: { foreground: 'primary' },
  destructive: { background: 'backgroundElement', foreground: 'danger' },
};

export type ButtonProps = Omit<PressableProps, 'children'> & {
  title: string;
  loading?: boolean;
  variant?: ButtonVariant;
  icon?: IconName;
};

export function Button({ title, loading, variant = 'primary', icon, disabled, style, ...pressableProps }: ButtonProps) {
  const theme = useTheme();
  const isDisabled = disabled || loading;
  const { background, foreground } = VARIANTS[variant];

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ disabled: isDisabled, busy: loading }}
      disabled={isDisabled}
      style={(state) => [
        styles.base,
        {
          backgroundColor: background ? theme[background] : 'transparent',
          opacity: isDisabled ? 0.5 : state.pressed ? 0.8 : 1,
        },
        typeof style === 'function' ? style(state) : style,
      ]}
      {...pressableProps}>
      {loading ? (
        <ActivityIndicator color={theme[foreground]} />
      ) : (
        <>
          {icon ? <Icon name={icon} size={18} color={foreground} /> : null}
          <ThemedText type="smallBold" themeColor={foreground}>
            {title}
          </ThemedText>
        </>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    minHeight: 48,
    borderRadius: Radius.md,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: Spacing.two,
    paddingHorizontal: Spacing.four,
  },
});
