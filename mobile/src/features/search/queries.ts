import { useQuery } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/api/client';
import type { SearchFilters } from './types';

export function useSearchTargets(query: string, filters: SearchFilters) {
  const trimmed = query.trim();
  return useQuery({
    // Spec minimum is 2 chars (api/openapi.yaml) — shorter queries are never sent.
    enabled: trimmed.length >= 2,
    queryKey: ['search', 'targets', trimmed, filters],
    queryFn: async () =>
      unwrap(
        await apiClient.GET('/api/v1/search/targets', {
          params: {
            query: {
              q: trimmed,
              category: filters.category,
              type: filters.type,
              min_rating: filters.minRating,
              verified: filters.verified,
            },
          },
        })
      ).data,
  });
}
