import { useQuery } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/api/client';

export function useMyBusinesses() {
  return useQuery({
    queryKey: ['businesses', 'mine'],
    queryFn: async () => unwrap(await apiClient.GET('/api/v1/businesses/mine')).data,
  });
}

export function useBusinessTargets(businessId: string | undefined) {
  return useQuery({
    enabled: !!businessId,
    queryKey: ['targets', 'by-business', businessId],
    queryFn: async () =>
      unwrap(await apiClient.GET('/api/v1/targets', { params: { query: { business: businessId! } } })).data,
  });
}
