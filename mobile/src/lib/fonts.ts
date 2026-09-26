import { NotoSansEthiopic_400Regular } from '@expo-google-fonts/noto-sans-ethiopic/400Regular';
import { NotoSansEthiopic_700Bold } from '@expo-google-fonts/noto-sans-ethiopic/700Bold';
import { useFonts } from 'expo-font';

// Two weights, not the package's full nine — Ethiopic text only ever
// appears bold (headings/emphasis) or regular in this app, and the other
// seven would add ~2.6 MB toward the APK budget (docs/mobile-plan.md §7)
// for weights nothing renders in. Not subsetted to just the Ethiopic block
// either (no subsetting pipeline exists yet) — a follow-up, not a blocker.
export const ETHIOPIC_FONT_REGULAR = 'NotoSansEthiopic_400Regular';
export const ETHIOPIC_FONT_BOLD = 'NotoSansEthiopic_700Bold';

export function useEthiopicFonts(): boolean {
  const [loaded] = useFonts({
    [ETHIOPIC_FONT_REGULAR]: NotoSansEthiopic_400Regular,
    [ETHIOPIC_FONT_BOLD]: NotoSansEthiopic_700Bold,
  });
  return loaded;
}
