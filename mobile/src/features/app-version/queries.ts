import { useQuery } from '@tanstack/react-query';
import Constants from 'expo-constants';
import { Platform } from 'react-native';

import { apiClient, unwrap } from '@/api/client';

export function useVersionGate() {
  return useQuery({
    queryKey: ['app', 'version'],
    queryFn: async () =>
      unwrap(
        await apiClient.GET('/api/v1/app/version', {
          params: {
            query: {
              platform: Platform.OS === 'ios' ? 'ios' : 'android',
              version: Constants.expoConfig?.version,
            },
          },
        })
      ).data,
    // Public, no auth needed, and cheap — but no reason to hammer it either.
    staleTime: 60 * 60 * 1000,
  });
}
