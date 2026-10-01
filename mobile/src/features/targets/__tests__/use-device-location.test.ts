import * as Location from 'expo-location';

import { locateDevice, POSITION_TIMEOUT_MS } from '../use-device-location';

jest.mock('expo-location', () => ({
  Accuracy: { Balanced: 3 },
  requestForegroundPermissionsAsync: jest.fn(),
  getLastKnownPositionAsync: jest.fn(),
  getCurrentPositionAsync: jest.fn(),
}));

const mocked = Location as jest.Mocked<typeof Location>;
const fix = { coords: { latitude: 9.03, longitude: 38.74 } } as Location.LocationObject;
const grant = (granted: boolean, canAskAgain = true) =>
  mocked.requestForegroundPermissionsAsync.mockResolvedValue({ granted, canAskAgain } as never);

beforeEach(() => jest.resetAllMocks());

it('reports a permanent denial separately so the UI can offer Settings', async () => {
  grant(false, false);
  expect(await locateDevice()).toEqual({ status: 'blocked' });
});

it('reports a re-promptable denial as denied', async () => {
  grant(false, true);
  expect(await locateDevice()).toEqual({ status: 'denied' });
});

it('uses a recent last-known fix without waiting on GPS', async () => {
  grant(true);
  mocked.getLastKnownPositionAsync.mockResolvedValue(fix);
  expect(await locateDevice()).toEqual({ status: 'ready', latitude: 9.03, longitude: 38.74 });
  expect(mocked.getCurrentPositionAsync).not.toHaveBeenCalled();
});

it('gives up with an error instead of spinning forever when no fix arrives', async () => {
  jest.useFakeTimers();
  grant(true);
  mocked.getLastKnownPositionAsync.mockResolvedValue(null);
  mocked.getCurrentPositionAsync.mockReturnValue(new Promise(() => {}));

  const result = locateDevice();
  await jest.advanceTimersByTimeAsync(POSITION_TIMEOUT_MS);

  expect(await result).toEqual({ status: 'error' });
  jest.useRealTimers();
});

it('returns an error when the permission request itself throws', async () => {
  mocked.requestForegroundPermissionsAsync.mockRejectedValue(new Error('no provider'));
  expect(await locateDevice()).toEqual({ status: 'error' });
});
