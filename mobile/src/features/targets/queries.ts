import { useQuery } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/api/client';

export function useCategories() {
  return useQuery({
    queryKey: ['categories'],
    queryFn: async () => unwrap(await apiClient.GET('/api/v1/categories')).data,
  });
}

export function useTopRatedTargets() {
  return useQuery({
    queryKey: ['targets', 'top-rated'],
    queryFn: async () => unwrap(await apiClient.GET('/api/v1/targets/top-rated')).data,
  });
}

export function useTrendingTargets() {
  return useQuery({
    queryKey: ['targets', 'trending'],
    queryFn: async () => unwrap(await apiClient.GET('/api/v1/targets/trending')).data,
  });
}
