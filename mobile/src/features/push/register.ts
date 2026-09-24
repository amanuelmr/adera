import Constants from 'expo-constants';
import * as Device from 'expo-device';
import * as Notifications from 'expo-notifications';
import { Platform } from 'react-native';

import { registerDeviceToken, unregisterDeviceToken, type DevicePlatform } from './queries';

// Backend delivery is raw FCM (internal/notifications/fcm.go), not Expo's
// push relay — getDevicePushTokenAsync() returns the native FCM/APNs
// registration token, not an ExponentPushToken[...], which is what that
// endpoint expects.
let currentToken: string | undefined;

function toDevicePlatform(osName: string): DevicePlatform | undefined {
  if (osName === 'android' || osName === 'ios') return osName;
  return undefined; // web isn't a push target for this app (Android-first, docs/mobile-plan.md §1.2)
}

async function register(platform: DevicePlatform, token: string): Promise<void> {
  currentToken = token;
  await registerDeviceToken(token, platform, Constants.expoConfig?.version);
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
