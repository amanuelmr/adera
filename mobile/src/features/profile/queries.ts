import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/api/client';
import type { components } from '@/api/schema';

export function useProfile() {
  return useQuery({
    queryKey: ['users', 'me'],
    queryFn: async () => unwrap(await apiClient.GET('/api/v1/users/me')).data,
  });
}

export function useUpdateProfile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: { displayName?: string; preferredLanguage?: components['schemas']['Language'] }) =>
      unwrap(
        await apiClient.PATCH('/api/v1/users/me', {
          body: { display_name: input.displayName, preferred_language: input.preferredLanguage },
        })
      ).data,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users', 'me'] });
    },
  });
}
