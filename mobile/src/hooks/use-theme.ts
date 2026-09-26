/**
 * Learn more about light and dark modes:
 * https://docs.expo.dev/guides/color-schemes/
 */

import { Colors } from '@/constants/theme';
import { useColorScheme } from '@/hooks/use-color-scheme';

export function useTheme() {
  const scheme = useColorScheme();
  // Normalize everything but an explicit 'dark' to 'light' — useColorScheme()
  // can also return 'unspecified', null, or undefined (e.g. before the web
  // build hydrates), none of which are keys in Colors.
  const theme = scheme === 'dark' ? 'dark' : 'light';

  return Colors[theme];
}
