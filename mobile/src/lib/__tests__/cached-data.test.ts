import { shouldShowCachedBanner } from '../cached-data';

const saved = { data: { name: 'Tomoca' }, dataUpdatedAt: 1_700_000_000_000, isRefetchError: false };

it('labels cached data while offline', () => {
  expect(shouldShowCachedBanner(saved, false)).toBe(true);
});

it('labels cached data whose refresh failed while online', () => {
  expect(shouldShowCachedBanner({ ...saved, isRefetchError: true }, true)).toBe(true);
});

it('stays hidden for fresh data, or when there is nothing on screen', () => {
  expect(shouldShowCachedBanner(saved, true)).toBe(false);
  expect(shouldShowCachedBanner({ data: undefined, dataUpdatedAt: 0, isRefetchError: false }, false)).toBe(false);
});
