import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiClient, unwrap, ApiError } from '@/api/client';
import { enqueueHelpfulVote } from '@/features/reviews/offline-queue';

export type ReviewSort = 'newest' | 'highest' | 'lowest' | 'most_helpful';

export type ReviewFilters = {
  sort: ReviewSort;
  rating?: number;
};

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

export function useTarget(idOrSlug: string) {
  return useQuery({
    queryKey: ['targets', 'detail', idOrSlug],
    queryFn: async () =>
      unwrap(await apiClient.GET('/api/v1/targets/{idOrSlug}', { params: { path: { idOrSlug } } })).data,
  });
}

export function useTargetStats(targetId: string | undefined) {
  return useQuery({
    // Trust-critical numbers are network-first, never served stale without a
    // label (docs/mobile-plan.md §5) — staleTime 0 overrides the 5-minute
    // default so every mount/refocus refetches rather than showing a cached
    // aggregate silently.
    staleTime: 0,
    enabled: !!targetId,
    queryKey: ['targets', 'stats', targetId],
    queryFn: async () =>
      unwrap(await apiClient.GET('/api/v1/targets/{id}/stats', { params: { path: { id: targetId! } } })).data,
  });
}

export function useTargetReviews(targetId: string | undefined, filters: ReviewFilters) {
  return useInfiniteQuery({
    enabled: !!targetId,
    queryKey: ['targets', 'reviews', targetId, filters],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const result = unwrap(
        await apiClient.GET('/api/v1/targets/{id}/reviews', {
          params: {
            path: { id: targetId! },
            query: { sort: filters.sort, rating: filters.rating, cursor: pageParam },
          },
        })
      );
      return result;
    },
    getNextPageParam: (lastPage) => (lastPage.meta?.has_more ? lastPage.meta.next_cursor : undefined),
  });
}

export function useToggleHelpful(targetId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ reviewId, voted }: { reviewId: string; voted: boolean }) => {
      try {
        unwrap(
          await (voted
            ? apiClient.DELETE('/api/v1/reviews/{id}/helpful', { params: { path: { id: reviewId } } })
            : apiClient.PUT('/api/v1/reviews/{id}/helpful', { params: { path: { id: reviewId } } }))
        );
      } catch (err) {
        if (err instanceof ApiError) throw err; // a real rejection — surface it, don't queue a retry that will just fail again
        // Network failure — safe to queue since the vote is idempotent and last-write-wins.
        await enqueueHelpfulVote(reviewId, !voted);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['targets', 'reviews', targetId] });
    },
  });
}
