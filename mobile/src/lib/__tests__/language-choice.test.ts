/* eslint-disable @typescript-eslint/no-require-imports -- fresh module state per test */
const mockDisk = new Map<string, string>();
jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: async (key: string) => mockDisk.get(key) ?? null,
  setItem: async (key: string, value: string) => void mockDisk.set(key, value),
}));
let mockDeviceLanguage = 'am';
jest.mock('expo-localization', () => ({ getLocales: () => [{ languageCode: mockDeviceLanguage }] }));

function loadI18n() {
  let mod: typeof import('../i18n');
  jest.isolateModules(() => {
    mod = require('../i18n');
  });
  return mod!;
}

beforeEach(() => {
  mockDisk.clear();
  mockDeviceLanguage = 'am';
});

it('asks on first run, starting from the device language', async () => {
  const i18n = loadI18n();
  await i18n.initI18n();
  expect(i18n.needsLanguageChoice()).toBe(true);
  expect(i18n.default.language).toBe('am');
});

it('stops asking once a language is chosen, and remembers it', async () => {
  const i18n = loadI18n();
  await i18n.initI18n();
  await i18n.setAppLanguage('en');
  expect(i18n.needsLanguageChoice()).toBe(false);

  const relaunched = loadI18n();
  await relaunched.initI18n();
  expect(relaunched.needsLanguageChoice()).toBe(false);
  expect(relaunched.default.language).toBe('en');
});

it('does not ask before i18n has loaded', () => {
  expect(loadI18n().needsLanguageChoice()).toBe(false);
});
