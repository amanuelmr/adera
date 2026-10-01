import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/api/client';
import type { components } from '@/api/schema';
import { deletePersistedPhoto } from './photos';

export type EvidenceKind = components['schemas']['Evidence']['kind'];

const EVIDENCE_KEY = (reviewId: string) => ['reviews', 'evidence', reviewId];

export function useEvidence(reviewId: string | undefined) {
  return useQuery({
    enabled: !!reviewId,
    queryKey: reviewId ? EVIDENCE_KEY(reviewId) : ['reviews', 'evidence'],
    queryFn: async () =>
      unwrap(await apiClient.GET('/api/v1/reviews/{id}/evidence', { params: { path: { id: reviewId! } } })).data,
  });
}

/** Mirrors {@link uploadReviewPhoto}'s two-phase shape, against the private evidence endpoints. */
async function submitEvidence(reviewId: string, kind: NonNullable<EvidenceKind>, localUri: string): Promise<void> {
  const ticket = unwrap(
    await apiClient.POST('/api/v1/reviews/{id}/evidence', {
      params: { path: { id: reviewId } },
      body: { kind, content_type: 'image/jpeg' },
    })
  );

  const uploadUrl = ticket.data.upload?.url;
  const uploadId = ticket.data.upload_id;
  if (!uploadUrl || !uploadId) throw new Error('Upload ticket missing url or id');

  const formData = new FormData();
  for (const [key, value] of Object.entries(ticket.data.upload?.fields ?? {})) {
    formData.append(key, value);
  }
  formData.append('file', { uri: localUri, name: 'evidence.jpg', type: 'image/jpeg' } as unknown as Blob);

  const uploadResponse = await fetch(uploadUrl, { method: 'POST', body: formData });
  if (!uploadResponse.ok) throw new Error(`Evidence upload failed with status ${uploadResponse.status}`);

  unwrap(await apiClient.POST('/api/v1/evidence/{id}/finalize', { params: { path: { id: uploadId } } }));
}

export function useSubmitEvidence(reviewId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ kind, localUri }: { kind: NonNullable<EvidenceKind>; localUri: string }) =>
      submitEvidence(reviewId!, kind, localUri),
    onSuccess: () => {
      if (reviewId) queryClient.invalidateQueries({ queryKey: EVIDENCE_KEY(reviewId) });
    },
    // Evidence is usually a receipt — names, phone numbers, payment refs. The
    // picked copy lives in the app's documents folder, so remove it whether
    // the upload worked or not (a failed one is re-picked, not retried).
    onSettled: (_data, _error, { localUri }) => deletePersistedPhoto(localUri),
  });
}

/** Evidence slots the server counts against its per-review limit: everything except rejected items. */
export function countsTowardEvidenceLimit(item: { status?: string }): boolean {
  return item.status !== 'rejected';
}
