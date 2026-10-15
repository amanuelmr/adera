/**
 * Adera's design tokens: colours for light and dark, spacing, radii,
 * elevation and the type scale. Screens use these through useTheme() and
 * ThemedText rather than literal values.
 *
 * Brand: a deep green for trust and actions, amber for ratings, on a warm
 * off-white — the same green as the website (internal/pages/static/site.css).
 * Every text/background pair meets WCAG AA (src/constants/__tests__/theme.test.ts).
 */

import '@/global.css';

import { Platform } from 'react-native';

const palette = {
  green900: '#08452F',
  green700: '#0B6E4F',
  green300: '#4FC79A',
  green100: '#DCEFE6',
  green950: '#062B1E',
  greenDark: '#12382B',
  greenDarkText: '#BDEBD6',
  amber: '#F2A900',
  amberLight: '#F5B82E',
  ink: '#1B1F1D',
  warmWhite: '#FAF7F2',
  white: '#FFFFFF',
  sand100: '#F1ECE3',
  sand200: '#E5DED2',
  sand300: '#DDD5C8',
  stone600: '#5A615D',
  night: '#111413',
  night800: '#1A1E1C',
  night700: '#242927',
  night600: '#2F3532',
  night500: '#3A413D',
  mist: '#F2F1EC',
  mist400: '#A9B0AC',
  red700: '#B3261E',
  red300: '#F2857B',
  orange800: '#9A4D00',
  orange300: '#F2A65A',
} as const;

export const Colors = {
  light: {
    // Surfaces: the page, then cards on it, then fills inside cards.
    background: palette.warmWhite,
    surface: palette.white,
    backgroundElement: palette.sand100,
    backgroundSelected: palette.sand200,
    border: palette.sand300,
    // Text.
    text: palette.ink,
    textSecondary: palette.stone600,
    // Brand.
    primary: palette.green700,
    onPrimary: palette.white,
    primaryContainer: palette.green100,
    onPrimaryContainer: palette.green900,
    // Stars and rating bars. Always shown with the number beside them, so
    // the colour is never the only carrier of the rating.
    rating: palette.amber,
    // States.
    verified: palette.green700,
    warning: palette.orange800,
    danger: palette.red700,
    scrim: 'rgba(0,0,0,0.45)',
  },
  dark: {
    background: palette.night,
    surface: palette.night800,
    backgroundElement: palette.night700,
    backgroundSelected: palette.night600,
    border: palette.night500,
    text: palette.mist,
    textSecondary: palette.mist400,
    primary: palette.green300,
    onPrimary: palette.green950,
    primaryContainer: palette.greenDark,
    onPrimaryContainer: palette.greenDarkText,
    rating: palette.amberLight,
    verified: palette.green300,
    warning: palette.orange300,
    danger: palette.red300,
    scrim: 'rgba(0,0,0,0.6)',
  },
} as const;

export type ThemeColor = keyof typeof Colors.light & keyof typeof Colors.dark;
export type Theme = (typeof Colors)['light'] | (typeof Colors)['dark'];

export const Fonts = Platform.select({
  ios: {
    /** iOS `UIFontDescriptorSystemDesignDefault` */
    sans: 'system-ui',
    /** iOS `UIFontDescriptorSystemDesignSerif` */
    serif: 'ui-serif',
    /** iOS `UIFontDescriptorSystemDesignRounded` */
    rounded: 'ui-rounded',
    /** iOS `UIFontDescriptorSystemDesignMonospaced` */
    mono: 'ui-monospace',
  },
  default: {
    sans: 'normal',
    serif: 'serif',
    rounded: 'normal',
    mono: 'monospace',
  },
  web: {
    sans: 'var(--font-display)',
    serif: 'var(--font-serif)',
    rounded: 'var(--font-rounded)',
    mono: 'var(--font-mono)',
  },
});

export const Spacing = {
  half: 2,
  one: 4,
  two: 8,
  /** 12: card padding, gaps inside cards. */
  twoHalf: 12,
  three: 16,
  four: 24,
  five: 32,
  six: 64,
} as const;

export const Radius = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  full: 999,
} as const;

/**
 * Depth for raised surfaces (cards on the page, sticky bars). Kept to two
 * light levels: shadows cost frames on low-end Android, and list rows stay
 * flat on their surface colour.
 */
export const Elevation = {
  card: Platform.select({
    android: { elevation: 1 },
    default: { shadowColor: '#000', shadowOpacity: 0.06, shadowRadius: 8, shadowOffset: { width: 0, height: 2 } },
  }),
  raised: Platform.select({
    android: { elevation: 3 },
    default: { shadowColor: '#000', shadowOpacity: 0.12, shadowRadius: 12, shadowOffset: { width: 0, height: 4 } },
  }),
} as const;

export const BottomTabInset = Platform.select({ ios: 50, android: 80 }) ?? 0;
export const MaxContentWidth = 800;
