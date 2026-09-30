import { formatDate, formatNumber, formatRelative } from '../format';

jest.mock('../i18n', () => ({ __esModule: true, default: { language: 'en' } }));

const date = '2026-03-05T10:30:00Z';

it('formats dates in the app language with the Gregorian calendar and Western digits', () => {
  expect(formatDate(date, 'en')).toBe('Mar 5, 2026');
  const am = formatDate(date, 'am');
  expect(am).toMatch(/2026/);
  expect(am).not.toMatch(/[፩-፼]/); // no Ge'ez numerals
});

it('formats numbers with Western digits in Amharic too', () => {
  expect(formatNumber(4.25, 'am', 1)).toMatch(/^4\.[23]$/);
});

it('formats relative times', () => {
  const now = new Date('2026-03-08T10:30:00Z');
  expect(formatRelative(date, 'en', now)).toBe('3 days ago');
  expect(formatRelative(now, 'en', now)).toBe('this minute');
});
