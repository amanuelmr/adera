import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/api/client';

const UNREAD_COUNT_KEY = ['activity', 'unread-count'];
const LIST_KEY_PREFIX = ['activity', 'list'];

export function useUnreadCount() {
  return useQuery({
    queryKey: UNREAD_COUNT_KEY,
    queryFn: async () => unwrap(await apiClient.GET('/api/v1/users/me/notifications/unread-count')).data,
  });
}

export function useNotifications(unreadOnly: boolean) {
  return useInfiniteQuery({
    queryKey: [...LIST_KEY_PREFIX, unreadOnly],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) =>
      unwrap(
        await apiClient.GET('/api/v1/users/me/notifications', {
          params: { query: { unread: unreadOnly || undefined, cursor: pageParam } },
        })
      ),
    getNextPageParam: (lastPage) => (lastPage.meta?.has_more ? lastPage.meta.next_cursor : undefined),
  });
}

export function useMarkNotificationRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) =>
      unwrap(await apiClient.PUT('/api/v1/users/me/notifications/{id}/read', { params: { path: { id } } })),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: LIST_KEY_PREFIX });
      queryClient.invalidateQueries({ queryKey: UNREAD_COUNT_KEY });
    },
  });
}

export function useMarkAllRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => unwrap(await apiClient.PUT('/api/v1/users/me/notifications/read-all')),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: LIST_KEY_PREFIX });
      queryClient.invalidateQueries({ queryKey: UNREAD_COUNT_KEY });
    },
  });
}
