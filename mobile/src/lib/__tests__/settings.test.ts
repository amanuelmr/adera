/* eslint-disable @typescript-eslint/no-require-imports -- fresh module state per test, simulating an app relaunch */
// Survives jest.resetModules(), like on-device storage survives a relaunch.
const mockDisk = new Map<string, string>();
jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: async (key: string) => mockDisk.get(key) ?? null,
  setItem: async (key: string, value: string) => void mockDisk.set(key, value),
}));

function launch(): typeof import('../settings') {
  jest.resetModules();
  return require('../settings');
}

beforeEach(() => mockDisk.clear());

it('defaults data saver to off', async () => {
  const settings = launch();
  await settings.loadSettings();
  expect(settings.getSettings().dataSaver).toBe(false);
});

it('persists data saver across launches', async () => {
  await launch().updateSettings({ dataSaver: true });

  const relaunched = launch();
  await relaunched.loadSettings();
  expect(relaunched.getSettings().dataSaver).toBe(true);
});

it('falls back to defaults when stored settings are corrupt', async () => {
  mockDisk.set('adera.settings.v1', '{not json');
  const settings = launch();
  await settings.loadSettings();
  expect(settings.getSettings().dataSaver).toBe(false);
});
