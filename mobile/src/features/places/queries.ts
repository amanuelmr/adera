import { useMutation, useQuery } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/api/client';
import type { components } from '@/api/schema';

export type PlaceSubmission = components['schemas']['TargetCreate'];

export function useCities() {
  return useQuery({
    queryKey: ['locations', 'cities'],
    queryFn: async () => unwrap(await apiClient.GET('/api/v1/locations/cities')).data,
    staleTime: 24 * 60 * 60 * 1000, // reference data
  });
}

export function useAreas(cityId: string | undefined) {
  return useQuery({
    enabled: !!cityId,
    queryKey: ['locations', 'areas', cityId],
    queryFn: async () =>
      unwrap(await apiClient.GET('/api/v1/locations/cities/{id}/areas', { params: { path: { id: cityId! } } })).data,
    staleTime: 24 * 60 * 60 * 1000,
  });
}

/** Submits a place; an ordinary user's submission waits for a moderator. */
export function useSubmitPlace() {
  return useMutation({
    mutationFn: async (place: PlaceSubmission) => unwrap(await apiClient.POST('/api/v1/targets', { body: place })).data,
  });
}
