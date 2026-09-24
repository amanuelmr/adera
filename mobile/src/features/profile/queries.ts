import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/api/client';

export function useProfile() {
  return useQuery({
    queryKey: ['users', 'me'],
    queryFn: async () => unwrap(await apiClient.GET('/api/v1/users/me')).data,
  });
}

export function useUpdateProfile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: { displayName?: string }) =>
      unwrap(await apiClient.PATCH('/api/v1/users/me', { body: { display_name: input.displayName } })).data,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users', 'me'] });
    },
  });
}
