import * as Location from 'expo-location';
import { useCallback, useState } from 'react';

export type DeviceLocationState =
  | { status: 'idle' }
  | { status: 'loading' }
  /** Denied, but the system will show the prompt again. */
  | { status: 'denied' }
  /** Denied permanently — only the system settings can change it. */
  | { status: 'blocked' }
  | { status: 'error' }
  | { status: 'ready'; latitude: number; longitude: number };

// getCurrentPositionAsync has no timeout of its own and can wait
// indefinitely indoors or with GPS off.
export const POSITION_TIMEOUT_MS = 10_000;
// A fix this recent and this accurate is good enough to rank what's nearby,
// and returns instantly instead of waiting on GPS.
const LAST_KNOWN = { maxAge: 10 * 60 * 1000, requiredAccuracy: 1000 };

function withTimeout<T>(promise: Promise<T>, ms: number): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error('Timed out getting location')), ms);
    promise.then(
      (value) => {
        clearTimeout(timer);
        resolve(value);
      },
      (err) => {
        clearTimeout(timer);
        reject(err);
      }
    );
  });
}

export async function locateDevice(): Promise<DeviceLocationState> {
  try {
    const permission = await Location.requestForegroundPermissionsAsync();
    if (!permission.granted) return { status: permission.canAskAgain ? 'denied' : 'blocked' };

    const position =
      (await Location.getLastKnownPositionAsync(LAST_KNOWN)) ??
      (await withTimeout(
        Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced }),
        POSITION_TIMEOUT_MS
      ));
    return { status: 'ready', latitude: position.coords.latitude, longitude: position.coords.longitude };
  } catch {
    return { status: 'error' };
  }
}

/**
 * Foreground, one-time position fix — not a background watch. Requesting is
 * explicit (call `request()` from a tap), not automatic on mount: asking for
 * location the instant Discover renders, before the user has done anything,
 * is the kind of permission prompt people reflexively deny.
 */
export function useDeviceLocation() {
  const [state, setState] = useState<DeviceLocationState>({ status: 'idle' });

  const request = useCallback(async () => {
    setState({ status: 'loading' });
    setState(await locateDevice());
  }, []);

  return { state, request };
}
