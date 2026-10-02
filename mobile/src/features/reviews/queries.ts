import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { ApiError, apiClient, unwrap, isTransientFailure } from '@/api/client';
import { File } from 'expo-file-system';

import { queryClient } from '@/lib/query-client';
import { deletePersistedPhoto } from './photos';
import type { ReviewFormState } from './types';
import { parsePrice } from './validation';

export function useCriteria(categoryId: string | undefined) {
  return useQuery({
    enabled: !!categoryId,
    queryKey: ['categories', categoryId, 'criteria'],
    queryFn: async () =>
      unwrap(await apiClient.GET('/api/v1/categories/{idOrCode}/criteria', { params: { path: { idOrCode: categoryId! } } })).data,
  });
}

function buildReviewBody(targetId: string, form: ReviewFormState) {
  return {
    target_id: targetId,
    overall_rating: form.overallRating!,
    title: form.title || undefined,
    body: form.body,
    experience_date: form.experienceDate || undefined,
    // Validated on the context step (parsePrice), so an invalid value never
    // reaches here; "1,500" is sent as 1500 rather than dropped.
    price_paid: parsePrice(form.pricePaid).amount,
    currency: 'ETB',
    discovery_source: form.discoverySource,
    expectation_match: form.expectationMatch,
    incentive_type: form.incentiveType,
    material_connection: form.materialConnection,
    disclosure_details: form.disclosureDetails || undefined,
    criterion_scores: form.criterionScores,
  };
}

export async function createReview(targetId: string, form: ReviewFormState, idempotencyKey: string): Promise<string> {
  const created = unwrap(
    await apiClient.POST('/api/v1/reviews', {
      headers: { 'Idempotency-Key': idempotencyKey },
      body: buildReviewBody(targetId, form),
    })
  );
  return created.data.id!;
}

/** Full replace (the API has no partial-patch) — version must be the one just fetched, or this 412s. */
export async function updateReview(reviewId: string, targetId: string, form: ReviewFormState, version: number): Promise<void> {
  unwrap(
    await apiClient.PUT('/api/v1/reviews/{id}', {
      params: { path: { id: reviewId } },
      body: { ...buildReviewBody(targetId, form), version },
    })
  );
}

/**
 * After a review is created or edited: the target's list, aggregates and
 * counts, the author's own list/profile, and the review itself (whose
 * `version` the next edit must send, or it 412s) are all out of date.
 */
export function invalidateReviewCaches(targetId: string, reviewId?: string): void {
  queryClient.invalidateQueries({ queryKey: ['targets', 'reviews', targetId] });
  queryClient.invalidateQueries({ queryKey: ['targets', 'stats', targetId] });
  queryClient.invalidateQueries({ queryKey: ['targets', 'reality-check', targetId] });
  queryClient.invalidateQueries({ queryKey: ['targets', 'detail'] });
  queryClient.invalidateQueries({ queryKey: ['users', 'me'] });
  if (reviewId) queryClient.invalidateQueries({ queryKey: ['reviews', 'detail', reviewId] });
  queryClient.invalidateQueries({ queryKey: ['reviews', 'eligibility', targetId] });
}

/** Advisory pre-check of the cooldown / daily cap; submit still enforces them. */
export function useReviewEligibility(targetId: string | undefined) {
  return useQuery({
    enabled: !!targetId,
    staleTime: 0,
    queryKey: ['reviews', 'eligibility', targetId],
    queryFn: async () =>
      unwrap(await apiClient.GET('/api/v1/targets/{id}/review-eligibility', { params: { path: { id: targetId! } } }))
        .data,
  });
}

export function useReview(reviewId: string | undefined) {
  return useQuery({
    enabled: !!reviewId,
    queryKey: ['reviews', 'detail', reviewId],
    queryFn: async () => unwrap(await apiClient.GET('/api/v1/reviews/{id}', { params: { path: { id: reviewId! } } })).data,
  });
}

/**
 * Creates the review, then uploads each photo as its own two-phase job
 * (presign → upload → finalize) — media attaches to a review that must
 * already exist. Photos that fail with a real server rejection (bad file,
 * etc.) are dropped; photos that fail because the network dropped mid-upload
 * are reported back as retryable so the caller can hand them to the offline
 * queue instead of losing them.
 */
export async function submitReview(
  targetId: string,
  form: ReviewFormState,
  idempotencyKey: string
): Promise<{ reviewId: string; retryablePhotoUris: string[] }> {
  const reviewId = await createReview(targetId, form, idempotencyKey);
  const retryablePhotoUris = await uploadPhotos(reviewId, form.photos.map((photo) => photo.uri));
  return { reviewId, retryablePhotoUris };
}

/** Same shape as {@link submitReview}, for the edit path — existing media is untouched; form.photos are only the newly added ones. */
export async function submitReviewEdit(
  reviewId: string,
  targetId: string,
  form: ReviewFormState,
  version: number
): Promise<{ retryablePhotoUris: string[] }> {
  await updateReview(reviewId, targetId, form, version);
  const retryablePhotoUris = await uploadPhotos(reviewId, form.photos.map((photo) => photo.uri));
  return { retryablePhotoUris };
}

/** Uploads each URI, returning the ones that failed for a reason worth retrying later. */
export async function uploadPhotos(reviewId: string, uris: string[]): Promise<string[]> {
  const retryable: string[] = [];
  for (const uri of uris) {
    // Gone already (cleared storage, or a crash between upload and
    // bookkeeping): nothing left to upload, and retrying would never succeed.
    if (!new File(uri).exists) continue;
    try {
      await uploadReviewPhoto(reviewId, uri);
      deletePersistedPhoto(uri);
    } catch (err) {
      if (isTransientFailure(err)) {
        retryable.push(uri);
      } else {
        // The server rejected this photo outright — retrying it unchanged
        // would just fail again, so give up and clean up.
        deletePersistedPhoto(uri);
      }
    }
  }
  return retryable;
}

export async function uploadReviewPhoto(reviewId: string, localUri: string): Promise<void> {
  const ticket = unwrap(
    await apiClient.POST('/api/v1/reviews/{id}/media', {
      params: { path: { id: reviewId } },
      body: { content_type: 'image/jpeg' },
    })
  );

  const uploadUrl = ticket.data.upload?.url;
  const uploadId = ticket.data.upload_id;
  if (!uploadUrl || !uploadId) throw new Error('Upload ticket missing url or id');

  const formData = new FormData();
  for (const [key, value] of Object.entries(ticket.data.upload?.fields ?? {})) {
    formData.append(key, value);
  }
  formData.append('file', { uri: localUri, name: 'photo.jpg', type: 'image/jpeg' } as unknown as Blob);

  const uploadResponse = await fetch(uploadUrl, { method: 'POST', body: formData });
  if (!uploadResponse.ok) throw new Error(`Photo upload failed with status ${uploadResponse.status}`);

  unwrap(await apiClient.POST('/api/v1/media/uploads/{id}/finalize', { params: { path: { id: uploadId } } }));
}

export function useMyReviews() {
  return useInfiniteQuery({
    queryKey: ['users', 'me', 'reviews'],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) =>
      unwrap(await apiClient.GET('/api/v1/users/me/reviews', { params: { query: { cursor: pageParam } } })),
    getNextPageParam: (lastPage) => (lastPage.meta?.has_more ? lastPage.meta.next_cursor : undefined),
  });
}

/** Owner response, posted by a member of the target's owning business. Invalidates the target's review pages (all filter variants) so the new response shows up. */
export function usePostResponse(targetId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ reviewId, body }: { reviewId: string; body: string }) =>
      unwrap(await apiClient.POST('/api/v1/reviews/{id}/response', { params: { path: { id: reviewId } }, body: { body } })),
    onSuccess: () => {
      if (targetId) queryClient.invalidateQueries({ queryKey: ['targets', 'reviews', targetId] });
    },
    onError: (err) => {
      // 409: another member of the business responded first. Refetch so the
      // card shows that response (editable) instead of retrying a POST that
      // can only fail again.
      if (targetId && err instanceof ApiError && err.status === 409) {
        queryClient.invalidateQueries({ queryKey: ['targets', 'reviews', targetId] });
      }
    },
  });
}

export function useEditResponse(targetId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ responseId, body }: { responseId: string; body: string }) =>
      unwrap(await apiClient.PUT('/api/v1/responses/{id}', { params: { path: { id: responseId } }, body: { body } })),
    onSuccess: () => {
      if (targetId) queryClient.invalidateQueries({ queryKey: ['targets', 'reviews', targetId] });
    },
  });
}

export function useDeleteReview() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (reviewId: string) =>
      unwrap(await apiClient.DELETE('/api/v1/reviews/{id}', { params: { path: { id: reviewId } } })),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users', 'me', 'reviews'] });
    },
  });
}
