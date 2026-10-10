/* eslint-disable @typescript-eslint/no-require-imports -- modules load after the mocks */
const mockDisk = new Map<string, string>();
jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: async (key: string) => mockDisk.get(key) ?? null,
  setItem: async (key: string, value: string) => void mockDisk.set(key, value),
  removeItem: async (key: string) => void mockDisk.delete(key),
}));
jest.mock('@react-native-community/netinfo', () => ({ addEventListener: () => () => undefined }));
jest.mock('expo-constants', () => ({ expoConfig: { version: '1.0.0' } }));

const { shouldPersistQueryKey, clearQueryCache, queryClient } = require('../query-client') as typeof import('../query-client');

it.each([
  [['categories'], true],
  [['locations', 'cities'], true],
  [['targets', 'detail', 'tomoca'], true],
  [['targets', 'reviews', 't1', { sort: 'newest' }], true],
  [['targets', 'top-rated', 'cat-1'], true],
  [['targets', 'trending'], true],
  [['targets', 'nearby', 9030, 38740], true],
  // Trust-critical: network-first, never shown stale (docs/mobile-plan.md §5).
  [['targets', 'stats', 't1'], false],
  [['targets', 'reality-check', 't1'], false],
  // Per-account or otherwise private.
  [['users', 'me'], false],
  [['users', 'me', 'reviews'], false],
  [['activity', 'list', false], false],
  [['reviews', 'eligibility', 't1'], false],
  [['reviews', 'evidence', 'r1'], false],
  [['businesses', 'mine'], false],
  [['targets', 'by-business', 'b1'], false],
  [['auth', 'sessions'], false],
  [['app', 'version'], false],
  [['search', 'targets', 'kitfo', {}], false],
])('persists %j: %s', (key, expected) => {
  expect(shouldPersistQueryKey(key)).toBe(expected);
});

it('an account change removes the cached data from the device, not only memory', async () => {
  mockDisk.set('adera.query-cache.v1', '{"clientState":{}}');
  queryClient.setQueryData(['categories'], [{ id: 'c1' }]);

  await clearQueryCache();

  expect(queryClient.getQueryData(['categories'])).toBeUndefined();
  expect(mockDisk.has('adera.query-cache.v1')).toBe(false);
});
