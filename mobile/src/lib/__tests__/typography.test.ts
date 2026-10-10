import { ethiopicLineHeight } from '../typography';

it.each([
  // The app's text styles: small, default, subtitle, title.
  [14, 20, 23],
  [16, 24, 26],
  [32, 44, 52],
  [48, 52, 77],
  // Already generous, or unset.
  [14, 30, 30],
  [16, undefined, 26],
])('font %p with line height %p → %p', (fontSize, lineHeight, expected) => {
  expect(ethiopicLineHeight(fontSize, lineHeight)).toBe(expected);
  expect(expected / fontSize).toBeGreaterThanOrEqual(1.6);
});
