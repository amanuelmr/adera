import { starsFor } from '../stars';

it('rounds to the nearest half star', () => {
  expect(starsFor(4.3)).toEqual(['full', 'full', 'full', 'full', 'half']);
  expect(starsFor(4.2)).toEqual(['full', 'full', 'full', 'full', 'empty']);
  expect(starsFor(4.75)).toEqual(['full', 'full', 'full', 'full', 'full']);
  expect(starsFor(1)).toEqual(['full', 'empty', 'empty', 'empty', 'empty']);
});

it('clamps out-of-range ratings', () => {
  expect(starsFor(-1)).toEqual(['empty', 'empty', 'empty', 'empty', 'empty']);
  expect(starsFor(9)).toEqual(['full', 'full', 'full', 'full', 'full']);
});
