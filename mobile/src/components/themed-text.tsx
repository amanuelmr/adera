import { useTranslation } from 'react-i18next';
import { Platform, StyleSheet, Text, type StyleProp, type TextProps, type TextStyle } from 'react-native';

import { Fonts, ThemeColor } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { ETHIOPIC_FONT_BOLD, ETHIOPIC_FONT_REGULAR } from '@/lib/fonts';
import { ethiopicLineHeight } from '@/lib/typography';

export type ThemedTextProps = TextProps & {
  type?:
    | 'default'
    | 'title'
    | 'subtitle'
    | 'headline'
    | 'small'
    | 'smallBold'
    | 'caption'
    | 'link'
    | 'linkPrimary'
    | 'code';
  themeColor?: ThemeColor;
};

const BOLD_TYPES = new Set(['title', 'subtitle', 'headline', 'smallBold', 'link', 'linkPrimary']);
const LINK_TYPES = new Set(['link', 'linkPrimary']);

export function ThemedText({ style, type = 'default', themeColor, ...rest }: ThemedTextProps) {
  const theme = useTheme();
  const { i18n } = useTranslation();

  // `code` is excluded: it renders literal values (paths, numbers), which
  // don't appear in Amharic, and should stay monospace regardless of locale.
  const ethiopicFontFamily =
    i18n.language === 'am' && type !== 'code' ? (BOLD_TYPES.has(type) ? ETHIOPIC_FONT_BOLD : ETHIOPIC_FONT_REGULAR) : undefined;

  const composed: StyleProp<TextStyle>[] = [
    { color: theme[themeColor ?? (LINK_TYPES.has(type) ? 'primary' : 'text')] },
    styles[type === 'linkPrimary' ? 'link' : type],
    ethiopicFontFamily && { fontFamily: ethiopicFontFamily },
    style,
  ];

  // Applied after the caller's style, so screens that set their own size
  // (e.g. a large place name) get Ethiopic-safe spacing too.
  let ethiopicSpacing: { lineHeight: number } | undefined;
  if (ethiopicFontFamily) {
    const { fontSize, lineHeight } = StyleSheet.flatten(composed);
    if (typeof fontSize === 'number') ethiopicSpacing = { lineHeight: ethiopicLineHeight(fontSize, lineHeight) };
  }

  return <Text style={[...composed, ethiopicSpacing]} {...rest} />;
}

// The type scale. Body text is regular weight; hierarchy comes from size and
// weight steps rather than everything being small and bold.
const styles = StyleSheet.create({
  title: {
    fontSize: 30,
    lineHeight: 36,
    fontWeight: 700,
  },
  subtitle: {
    fontSize: 22,
    lineHeight: 28,
    fontWeight: 600,
  },
  headline: {
    fontSize: 18,
    lineHeight: 24,
    fontWeight: 600,
  },
  default: {
    fontSize: 16,
    lineHeight: 24,
    fontWeight: 400,
  },
  small: {
    fontSize: 14,
    lineHeight: 20,
    fontWeight: 400,
  },
  smallBold: {
    fontSize: 14,
    lineHeight: 20,
    fontWeight: 600,
  },
  caption: {
    fontSize: 12,
    lineHeight: 16,
    fontWeight: 500,
  },
  link: {
    fontSize: 14,
    lineHeight: 20,
    fontWeight: 600,
  },
  code: {
    fontFamily: Fonts.mono,
    fontWeight: Platform.select({ android: 700 }) ?? 500,
    fontSize: 12,
  },
});
