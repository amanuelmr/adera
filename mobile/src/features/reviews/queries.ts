import { useQuery } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/api/client';
import type { ReviewFormState } from './types';

export function useCriteria(categoryId: string | undefined) {
  return useQuery({
    enabled: !!categoryId,
    queryKey: ['categories', categoryId, 'criteria'],
    queryFn: async () =>
      unwrap(await apiClient.GET('/api/v1/categories/{idOrCode}/criteria', { params: { path: { idOrCode: categoryId! } } })).data,
  });
}

/**
 * Creates the review, then uploads each photo as its own two-phase job
 * (presign → upload → finalize) — media attaches to a review that must
 * already exist. A photo failure doesn't roll back the review: it's already
 * published, and the plan treats a lost photo as recoverable, not fatal
 * (docs/mobile-plan.md §5 — this is exactly the gap the offline queue,
 * next feature slice, closes by retrying just the upload).
 */
export async function submitReview(
  targetId: string,
  form: ReviewFormState,
  idempotencyKey: string
): Promise<{ reviewId: string; failedPhotoCount: number }> {
  const created = unwrap(
    await apiClient.POST('/api/v1/reviews', {
      headers: { 'Idempotency-Key': idempotencyKey },
      body: {
        target_id: targetId,
        overall_rating: form.overallRating!,
        title: form.title || undefined,
        body: form.body,
        experience_date: form.experienceDate || undefined,
        // Silently dropped rather than sent as NaN/null on non-numeric input
        // — the field is optional and low-stakes, not worth a validation error.
        price_paid: Number.isFinite(Number(form.pricePaid)) && form.pricePaid ? Number(form.pricePaid) : undefined,
        currency: 'ETB',
        discovery_source: form.discoverySource,
        expectation_match: form.expectationMatch,
        incentive_type: form.incentiveType,
        material_connection: form.materialConnection,
        disclosure_details: form.disclosureDetails || undefined,
        criterion_scores: form.criterionScores,
      },
    })
  );

  const reviewId = created.data.id!;
  let failedPhotoCount = 0;
  for (const photo of form.photos) {
    try {
      await uploadReviewPhoto(reviewId, photo.uri);
    } catch {
      failedPhotoCount += 1;
    }
  }

  return { reviewId, failedPhotoCount };
}

async function uploadReviewPhoto(reviewId: string, localUri: string): Promise<void> {
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
