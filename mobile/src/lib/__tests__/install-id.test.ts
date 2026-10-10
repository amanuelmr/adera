/* eslint-disable @typescript-eslint/no-require-imports -- fresh module state per test */
const mockStore = new Map<string, string>();
let mockFailReads = false;
jest.mock('expo-secure-store', () => ({
  getItemAsync: async (key: string) => {
    if (mockFailReads) throw new Error('keystore unavailable');
    return mockStore.get(key) ?? null;
  },
  setItemAsync: async (key: string, value: string) => void mockStore.set(key, value),
}));
let mockUuidCount = 0;
jest.mock('expo-crypto', () => ({ randomUUID: () => `00000000-0000-4000-8000-00000000000${++mockUuidCount}` }));

function load() {
  let mod: typeof import('../install-id');
  jest.isolateModules(() => {
    mod = require('../install-id');
  });
  return mod!;
}

beforeEach(() => {
  mockStore.clear();
  mockFailReads = false;
  mockUuidCount = 0;
});

it('creates the ID once and keeps it across launches', async () => {
  const first = await load().getInstallId();
  expect(first).toBe('00000000-0000-4000-8000-000000000001');
  expect(await load().getInstallId()).toBe(first);
  expect(mockUuidCount).toBe(1);
});

it('gives up quietly when secure storage fails, and tries again later', async () => {
  const installId = load();
  mockFailReads = true;
  expect(await installId.getInstallId()).toBeUndefined();
  mockFailReads = false;
  expect(await installId.getInstallId()).toBe('00000000-0000-4000-8000-000000000001');
});
