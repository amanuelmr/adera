import { apiClient } from '@/api/client';

export type DevicePlatform = 'android' | 'ios' | 'web';

export async function registerDeviceToken(token: string, platform: DevicePlatform, appVersion?: string): Promise<void> {
  await apiClient.POST('/api/v1/users/me/devices', { body: { token, platform, app_version: appVersion } });
}

export async function unregisterDeviceToken(token: string): Promise<void> {
  await apiClient.DELETE('/api/v1/users/me/devices', { body: { token } });
}
