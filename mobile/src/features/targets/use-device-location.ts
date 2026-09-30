import * as Location from 'expo-location';
import { useCallback, useState } from 'react';

export type DeviceLocationState =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'denied' }
  | { status: 'error' }
  | { status: 'ready'; latitude: number; longitude: number };

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
    const permission = await Location.requestForegroundPermissionsAsync();
    if (!permission.granted) {
      setState({ status: 'denied' });
      return;
    }
    try {
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      setState({ status: 'ready', latitude: position.coords.latitude, longitude: position.coords.longitude });
    } catch {
      setState({ status: 'error' });
    }
  }, []);

  return { state, request };
}
