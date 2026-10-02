const mockStore = new Map<string, string>();
jest.mock('expo-secure-store', () => ({
  getItemAsync: jest.fn(async (key: string) => mockStore.get(key) ?? null),
  setItemAsync: jest.fn(async (key: string, value: string) => void mockStore.set(key, value)),
  deleteItemAsync: jest.fn(async (key: string) => void mockStore.delete(key)),
}));
jest.mock('expo-constants', () => ({ expoConfig: { version: '1.0.0' } }));
jest.mock('expo-device', () => ({ isDevice: true }));
jest.mock('expo-notifications', () => ({}));
jest.mock('../queries', () => ({ registerDeviceToken: jest.fn(async () => undefined), unregisterDeviceToken: jest.fn(async () => undefined) }));

/* eslint-disable import/first -- modules load after the mocks above */
import { registerDeviceToken, unregisterDeviceToken } from '../queries';
import { register, unregisterCurrentDevice } from '../register';

beforeEach(async () => {
  await unregisterCurrentDevice(); // forget the previous test's in-memory token
  mockStore.clear();
  mockStore.set('adera.user_id', 'user-1');
  jest.clearAllMocks();
});

it('registers a token once per account, not on every launch', async () => {
  await register('android', 'token-a');
  await register('android', 'token-a');
  expect(registerDeviceToken).toHaveBeenCalledTimes(1);

  await register('android', 'token-b'); // the OS rotated the token
  expect(registerDeviceToken).toHaveBeenCalledTimes(2);

  mockStore.set('adera.user_id', 'user-2'); // someone else signed in
  await register('android', 'token-b');
  expect(registerDeviceToken).toHaveBeenCalledTimes(3);
});

it('retries next time when registration failed', async () => {
  (registerDeviceToken as jest.Mock).mockRejectedValueOnce(new Error('offline'));
  jest.spyOn(console, 'warn').mockImplementation(() => undefined);
  await register('android', 'token-a');
  await register('android', 'token-a');
  expect(registerDeviceToken).toHaveBeenCalledTimes(2);
});

it('can unregister after a restart, from the stored registration', async () => {
  mockStore.set('adera.push_registration', JSON.stringify({ token: 'token-old', userId: 'user-1' }));
  await unregisterCurrentDevice();
  expect(unregisterDeviceToken).toHaveBeenCalledWith('token-old');
  expect(mockStore.has('adera.push_registration')).toBe(false);
});
