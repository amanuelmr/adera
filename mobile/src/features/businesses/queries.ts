import { useInfiniteQuery, useQuery } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/api/client';

export function useMyBusinesses() {
  return useQuery({
    queryKey: ['businesses', 'mine'],
    queryFn: async () => unwrap(await apiClient.GET('/api/v1/businesses/mine')).data,
  });
}

export function useBusinessTargets(businessId: string | undefined) {
  return useInfiniteQuery({
    enabled: !!businessId,
    queryKey: ['targets', 'by-business', businessId],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) =>
      unwrap(
        await apiClient.GET('/api/v1/targets', { params: { query: { business: businessId!, cursor: pageParam } } })
      ),
    getNextPageParam: (lastPage) => (lastPage.meta?.has_more ? lastPage.meta.next_cursor : undefined),
  });
}
