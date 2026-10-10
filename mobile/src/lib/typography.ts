// Ethiopic script stacks marks above and below the base glyph, so lines set
// at Latin spacing (1.3–1.5×) clip or collide; docs/frontend-handoff.md asks
// for at least 1.6× when the interface is in Amharic.
export const ETHIOPIC_MIN_LINE_HEIGHT_RATIO = 1.6;

/** The line height to use for Ethiopic text: the given one, raised to the minimum ratio if needed. */
export function ethiopicLineHeight(fontSize: number, lineHeight: number | undefined): number {
  const minimum = Math.ceil(fontSize * ETHIOPIC_MIN_LINE_HEIGHT_RATIO);
  return lineHeight != null && lineHeight >= minimum ? lineHeight : minimum;
}
