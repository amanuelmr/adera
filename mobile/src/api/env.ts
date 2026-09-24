import { Platform } from 'react-native';

// The Android emulator's loopback interface is 10.0.2.2, not localhost — the
// backend running via `docker compose up` on the host is unreachable at
// localhost from inside the emulator. iOS simulator and web share the host's
// network namespace, so localhost works there. A physical device needs the
// host's LAN IP, which EXPO_PUBLIC_API_BASE_URL must set explicitly.
const DEV_DEFAULT_BASE_URL = Platform.OS === 'android' ? 'http://10.0.2.2:8080' : 'http://localhost:8080';

export const apiBaseUrl = process.env.EXPO_PUBLIC_API_BASE_URL ?? DEV_DEFAULT_BASE_URL;
