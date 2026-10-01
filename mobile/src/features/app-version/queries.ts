import { useQuery } from '@tanstack/react-query';
import Constants from 'expo-constants';
import { Platform } from 'react-native';

import { apiClient, unwrap } from '@/api/client';

// The splash screen waits on this check (app/_layout.tsx), so a stalled —
// not refused — connection must fail fast rather than hold startup through
// fetch's own multi-minute timeout and TanStack's default three retries.
export const VERSION_CHECK_TIMEOUT_MS = 5000;

export async function fetchVersionGate(signal?: AbortSignal) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), VERSION_CHECK_TIMEOUT_MS);
  signal?.addEventListener('abort', () => controller.abort());
  try {
    return unwrap(
      await apiClient.GET('/api/v1/app/version', {
        params: {
          query: {
            platform: Platform.OS === 'ios' ? 'ios' : 'android',
            version: Constants.expoConfig?.version,
          },
        },
        signal: controller.signal,
      })
    ).data;
  } finally {
    clearTimeout(timer);
  }
}

export function useVersionGate() {
  return useQuery({
    queryKey: ['app', 'version'],
    queryFn: ({ signal }) => fetchVersionGate(signal),
    retry: 1,
    retryDelay: 500,
    // Public, no auth needed, and cheap — but no reason to hammer it either.
    staleTime: 60 * 60 * 1000,
  });
}
