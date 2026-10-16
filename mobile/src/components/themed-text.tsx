import { useTranslation } from 'react-i18next';
import { Platform, StyleSheet, Text, type StyleProp, type TextProps, type TextStyle } from 'react-native';

import { Fonts, ThemeColor } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { fontFamilyFor } from '@/lib/fonts';
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

const LINK_TYPES = new Set(['link', 'linkPrimary']);

export function ThemedText({ style, type = 'default', themeColor, ...rest }: ThemedTextProps) {
  const theme = useTheme();
  const { i18n } = useTranslation();

  const composed: StyleProp<TextStyle>[] = [
    { color: theme[themeColor ?? (LINK_TYPES.has(type) ? 'primary' : 'text')] },
    styles[type === 'linkPrimary' ? 'link' : type],
    style,
  ];

  // `code` keeps its monospace font: it renders literal values (paths,
  // numbers), which don't appear in Amharic. Everything else gets the app's
  // font for its weight (including a weight the caller set, such as a large
  // rating number), in Ethiopic when Amharic is active. Applied after the
  // caller's style, as is the Ethiopic-safe line height, so screens that set
  // their own size get both too.
  let fontStyle: TextStyle | undefined;
  if (type !== 'code') {
    const flat = StyleSheet.flatten(composed);
    if (!flat.fontFamily) {
      const ethiopic = i18n.language === 'am';
      fontStyle = { fontFamily: fontFamilyFor(Number(flat.fontWeight ?? 400), ethiopic), fontWeight: 'normal' };
      if (ethiopic && typeof flat.fontSize === 'number') {
        fontStyle.lineHeight = ethiopicLineHeight(flat.fontSize, flat.lineHeight);
      }
    }
  }

  return <Text style={[...composed, fontStyle]} {...rest} />;
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
