import { Colors } from '../theme';

// WCAG 2.1 relative luminance and contrast ratio.
function luminance(hex: string): number {
  const [r, g, b] = [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255);
  const lin = (c: number) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4);
  return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
}

function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

// Text sits on the page, on cards, and on fills inside cards; buttons put
// their label on the brand colour.
const TEXT_PAIRS = [
  ['text', 'background'],
  ['text', 'surface'],
  ['text', 'backgroundElement'],
  ['text', 'backgroundSelected'],
  ['textSecondary', 'background'],
  ['textSecondary', 'surface'],
  ['textSecondary', 'backgroundElement'],
  ['primary', 'background'],
  ['primary', 'surface'],
  ['onPrimary', 'primary'],
  ['onPrimaryContainer', 'primaryContainer'],
  ['danger', 'background'],
  ['danger', 'surface'],
  ['warning', 'surface'],
  ['verified', 'surface'],
] as const;

describe.each(['light', 'dark'] as const)('%s theme', (mode) => {
  const theme = Colors[mode];

  it.each(TEXT_PAIRS)('%s on %s meets AA for body text (4.5:1)', (fg, bg) => {
    expect(contrast(theme[fg], theme[bg])).toBeGreaterThanOrEqual(4.5);
  });
});

// Amber stars on a light card can't reach the 3:1 graphics minimum without
// turning brown, so the light theme relies on the number always shown
// beside them (and the stars' accessibility label). Dark mode has no such
// trade-off, and must keep it.
it('dark-theme stars meet the 3:1 graphics minimum on cards', () => {
  expect(contrast(Colors.dark.rating, Colors.dark.surface)).toBeGreaterThanOrEqual(3);
});
