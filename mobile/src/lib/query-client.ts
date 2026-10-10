import AsyncStorage from '@react-native-async-storage/async-storage';
import NetInfo from '@react-native-community/netinfo';
import { createAsyncStoragePersister } from '@tanstack/query-async-storage-persister';
import { defaultShouldDehydrateQuery, focusManager, onlineManager, QueryClient, type Query } from '@tanstack/react-query';
import type { PersistQueryClientOptions } from '@tanstack/react-query-persist-client';
import Constants from 'expo-constants';
import { AppState, Platform } from 'react-native';

// How long browse data stays usable offline. Persisted queries must also
// outlive it in memory (gcTime), or restored data is garbage-collected first.
export const PERSIST_MAX_AGE_MS = 7 * 24 * 60 * 60 * 1000;

// 5-minute default staleTime gives list/browse screens stale-while-revalidate
// behavior (docs/mobile-plan.md §5): cached data renders immediately, a
// background refetch keeps it current. Trust-critical numbers (aggregates,
// Reality Check) need network-first instead — those queries override this
// with staleTime: 0 individually rather than changing the default here.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,
      gcTime: PERSIST_MAX_AGE_MS,
    },
  },
});

// React Native has no window focus event, so TanStack never learns the app
// came back to the foreground. Feeding it AppState makes stale queries (the
// unread badge especially) refetch when the user returns to the app.
if (Platform.OS !== 'web') {
  AppState.addEventListener('change', (status) => focusManager.setFocused(status === 'active'));
  // Nor does it know when the device is offline; without this it keeps
  // firing requests that can only fail, instead of pausing until reconnect.
  onlineManager.setEventListener((setOnline) => NetInfo.addEventListener((state) => setOnline(!!state.isConnected)));
}

/**
 * What may be kept on the device between launches: public browse data only
 * (docs/mobile-plan.md §5 — stale-while-revalidate with a "saved N ago"
 * label). Never trust-critical aggregates (stats, Reality Check: network-
 * first, never shown stale) and never per-account data.
 */
export function shouldPersistQueryKey(queryKey: readonly unknown[]): boolean {
  const [scope, kind] = queryKey;
  if (scope === 'categories' || scope === 'locations') return true;
  if (scope !== 'targets') return false;
  return kind === 'detail' || kind === 'reviews' || kind === 'top-rated' || kind === 'trending' || kind === 'nearby';
}

export const queryPersister = createAsyncStoragePersister({
  storage: AsyncStorage,
  key: 'adera.query-cache.v1',
  throttleTime: 1000,
});

export const persistOptions: Omit<PersistQueryClientOptions, 'queryClient'> = {
  persister: queryPersister,
  maxAge: PERSIST_MAX_AGE_MS,
  // A new app version may change response shapes; start from a fresh cache.
  buster: Constants.expoConfig?.version ?? 'dev',
  dehydrateOptions: {
    shouldDehydrateQuery: (query: Query) => defaultShouldDehydrateQuery(query) && shouldPersistQueryKey(query.queryKey),
  },
};

/** Account change: drop cached data in memory and on the device. */
export async function clearQueryCache(): Promise<void> {
  queryClient.clear();
  await queryPersister.removeClient();
}
