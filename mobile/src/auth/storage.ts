import * as SecureStore from 'expo-secure-store';

// Keystore-backed on Android, Keychain-backed on iOS (per docs/mobile-plan.md
// §6) — never SharedPreferences/plain storage. expo-secure-store has no web
// implementation; this app is Android-first (§1.2) and auth intentionally
// doesn't fall back to an insecure web store.
const ACCESS_TOKEN_KEY = 'adera.access_token';
const REFRESH_TOKEN_KEY = 'adera.refresh_token';
const USER_ID_KEY = 'adera.user_id';

export type TokenPair = { accessToken: string; refreshToken: string };

export async function getAccessToken(): Promise<string | null> {
  return SecureStore.getItemAsync(ACCESS_TOKEN_KEY);
}

export async function getRefreshToken(): Promise<string | null> {
  return SecureStore.getItemAsync(REFRESH_TOKEN_KEY);
}

export async function setTokens({ accessToken, refreshToken }: TokenPair): Promise<void> {
  await Promise.all([
    SecureStore.setItemAsync(ACCESS_TOKEN_KEY, accessToken),
    SecureStore.setItemAsync(REFRESH_TOKEN_KEY, refreshToken),
  ]);
}

export async function clearTokens(): Promise<void> {
  await Promise.all([
    SecureStore.deleteItemAsync(ACCESS_TOKEN_KEY),
    SecureStore.deleteItemAsync(REFRESH_TOKEN_KEY),
    SecureStore.deleteItemAsync(USER_ID_KEY),
  ]);
}

// Tags offline-queue jobs (src/features/reviews/offline-queue.ts) so a job
// queued by one account is never replayed under a different one signed in
// later on the same device.
export async function getCurrentUserId(): Promise<string | null> {
  return SecureStore.getItemAsync(USER_ID_KEY);
}

export async function setCurrentUserId(userId: string): Promise<void> {
  await SecureStore.setItemAsync(USER_ID_KEY, userId);
}
