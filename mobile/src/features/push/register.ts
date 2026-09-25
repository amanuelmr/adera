import Constants from 'expo-constants';
import * as Device from 'expo-device';
import * as Notifications from 'expo-notifications';
import { Platform } from 'react-native';

import { registerDeviceToken, unregisterDeviceToken, type DevicePlatform } from './queries';

// Backend delivery is raw FCM (internal/notifications/fcm.go), not Expo's
// push relay — getDevicePushTokenAsync() returns the native device
// registration token, which on Android *is* an FCM token but on iOS is an
// APNs token instead. The backend's FCM provider can't deliver to an APNs
// token, so registering one would just store a destination nothing can ever
// send to. Android-only for v1 anyway (docs/mobile-plan.md §1.2) — iOS
// support needs its own Firebase-Cloud-Messaging-for-iOS integration to get
// a real FCM token, not just skipping this check.
let currentToken: string | undefined;

function toDevicePlatform(osName: string): DevicePlatform | undefined {
  return osName === 'android' ? 'android' : undefined;
}

// Best-effort and logged, not thrown — a failed registration (transient
// network issue, backend rejecting a malformed token) shouldn't crash app
// startup or the token-rotation listener below, but silently swallowing it
// entirely made the failure invisible even for debugging.
async function register(platform: DevicePlatform, token: string): Promise<void> {
  currentToken = token;
  try {
    await registerDeviceToken(token, platform, Constants.expoConfig?.version);
  } catch (err) {
    console.warn('Failed to register push device token', err);
  }
}

/**
 * Requests permission, registers this device's current token, and keeps it
 * registered if the OS rotates it (FCM tokens aren't permanent). No-ops on
 * simulators/emulators and web. Returns an unsubscribe for the rotation
 * listener.
 */
export async function setupPushNotifications(): Promise<() => void> {
  const platform = toDevicePlatform(Platform.OS);
  if (!platform || !Device.isDevice) return () => undefined;

  // Required before requesting permission on Android 13+, which won't show
  // the prompt until at least one channel exists.
  await Notifications.setNotificationChannelAsync('default', {
    name: 'Default',
    importance: Notifications.AndroidImportance.DEFAULT,
  });

  const permission = await Notifications.requestPermissionsAsync();
  if (!permission.granted) return () => undefined;

  const { data: token } = await Notifications.getDevicePushTokenAsync();
  await register(platform, token);

  const subscription = Notifications.addPushTokenListener(({ data }) => {
    register(platform, data);
  });
  return () => subscription.remove();
}

/** Called on logout — stops this device from receiving pushes for the account being signed out of. */
export async function unregisterCurrentDevice(): Promise<void> {
  if (!currentToken) return;
  await unregisterDeviceToken(currentToken).catch(() => undefined); // best-effort, matching auth's own logout
  currentToken = undefined;
}
