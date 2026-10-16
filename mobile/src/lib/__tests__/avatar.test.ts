import { AVATAR_COLORS, avatarColorFor, initialsOf } from '../avatar';

function contrastWithWhite(hex: string): number {
  const lin = (c: number) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4);
  const [r, g, b] = [1, 3, 5].map((i) => lin(parseInt(hex.slice(i, i + 2), 16) / 255));
  return 1.05 / (0.2126 * r + 0.7152 * g + 0.0722 * b + 0.05);
}

it('takes the first letters of the first and last name', () => {
  expect(initialsOf('Abebe Kebede')).toBe('AK');
  expect(initialsOf('  sara  ')).toBe('S');
  expect(initialsOf('Hana Tesfaye Alemu')).toBe('HA');
  expect(initialsOf('አበበ ከበደ')).toBe('አከ');
  expect(initialsOf('')).toBe('?');
});

it('gives a name the same colour every time', () => {
  expect(avatarColorFor('Abebe Kebede')).toBe(avatarColorFor('Abebe Kebede'));
  expect(AVATAR_COLORS).toContain(avatarColorFor('Former member'));
});

it.each(AVATAR_COLORS)('%s carries white initials at 4.5:1', (color) => {
  expect(contrastWithWhite(color)).toBeGreaterThanOrEqual(4.5);
});
