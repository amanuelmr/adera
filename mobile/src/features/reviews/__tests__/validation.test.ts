import { experienceDateError, parsePrice } from '../validation';

const today = new Date(2026, 9, 1); // Oct 1, 2026, local

it.each([
  ['', undefined],
  ['2026-09-30', undefined],
  ['2026-10-01', undefined],
  ['2026-10-02', 'future'],
  ['2026-02-31', 'format'],
  ['30/09/2026', 'format'],
  ['2026-9-3', 'format'],
])('experience date %p → %p', (value, expected) => {
  expect(experienceDateError(value, today)).toBe(expected);
});

it.each([
  ['', {}],
  ['1500', { amount: 1500 }],
  ['1,500', { amount: 1500 }],
  ['1 500', { amount: 1500 }],
  ['250.50', { amount: 250.5 }],
  ['abc', { error: 'invalid' }],
  ['-5', { error: 'invalid' }],
  ['1.234', { error: 'invalid' }],
  ['0', { error: 'invalid' }],
  ['200000000', { error: 'invalid' }],
])('price %p → %p', (value, expected) => {
  expect(parsePrice(value)).toEqual(expected);
});
