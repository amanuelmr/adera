import { Inter_400Regular } from '@expo-google-fonts/inter/400Regular';
import { Inter_600SemiBold } from '@expo-google-fonts/inter/600SemiBold';
import { Inter_700Bold } from '@expo-google-fonts/inter/700Bold';
import { NotoSansEthiopic_400Regular } from '@expo-google-fonts/noto-sans-ethiopic/400Regular';
import { NotoSansEthiopic_700Bold } from '@expo-google-fonts/noto-sans-ethiopic/700Bold';
import { useFonts } from 'expo-font';

// Latin text uses Inter in the three weights the type scale uses (about
// 1 MB together). Ethiopic text uses Noto Sans Ethiopic in two — regular
// and bold — not the package's full nine: the other weights would add about
// 2.6 MB toward the APK budget (docs/mobile-plan.md §7), and semibold
// headings read fine in bold. Not subsetted to just the Ethiopic block
// either (no subsetting pipeline exists yet) — a follow-up, not a blocker.
export const ETHIOPIC_FONT_REGULAR = 'NotoSansEthiopic_400Regular';
export const ETHIOPIC_FONT_BOLD = 'NotoSansEthiopic_700Bold';

const LATIN_FONTS = {
  400: 'Inter_400Regular',
  600: 'Inter_600SemiBold',
  700: 'Inter_700Bold',
} as const;

/**
 * The font family for a weight. Custom fonts carry their weight in the file,
 * so callers set this family instead of fontWeight (Android otherwise
 * synthesises bold, or falls back to the system font).
 */
export function fontFamilyFor(weight: number, ethiopic: boolean): string {
  if (ethiopic) return weight >= 600 ? ETHIOPIC_FONT_BOLD : ETHIOPIC_FONT_REGULAR;
  return weight >= 700 ? LATIN_FONTS[700] : weight >= 600 ? LATIN_FONTS[600] : LATIN_FONTS[400];
}

/** Inter: always needed, so the splash screen waits for it. */
export function useLatinFonts(): boolean {
  const [loaded, error] = useFonts({
    [LATIN_FONTS[400]]: Inter_400Regular,
    [LATIN_FONTS[600]]: Inter_600SemiBold,
    [LATIN_FONTS[700]]: Inter_700Bold,
  });
  // A font that fails to load falls back to the system font: better than
  // holding the splash screen forever.
  return loaded || !!error;
}

export function useEthiopicFonts(): boolean {
  const [loaded] = useFonts({
    [ETHIOPIC_FONT_REGULAR]: NotoSansEthiopic_400Regular,
    [ETHIOPIC_FONT_BOLD]: NotoSansEthiopic_700Bold,
  });
  return loaded;
}
