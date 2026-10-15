export type StarFill = 'full' | 'half' | 'empty';

/**
 * Five stars for a rating, rounded to the nearest half: 4.3 shows four and
 * a half, 4.2 shows four. Out-of-range values are clamped.
 */
export function starsFor(rating: number): StarFill[] {
  const halves = Math.round(Math.min(5, Math.max(0, rating)) * 2);
  return Array.from({ length: 5 }, (_, i) => {
    const filled = halves - i * 2;
    return filled >= 2 ? 'full' : filled === 1 ? 'half' : 'empty';
  });
}
