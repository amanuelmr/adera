import { useMutation } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/api/client';
import type { components } from '@/api/schema';

export type ReportReason = components['schemas']['ReportCreate']['reason'];

// Order shown to the user: the most common reasons first.
export const REPORT_REASONS: ReportReason[] = [
  'fake_experience',
  'conflict_of_interest',
  'spam',
  'irrelevant',
  'duplicate',
  'unsupported_accusation',
  'personal_information',
  'harassment',
  'hate_speech',
  'manipulated_evidence',
];

export const MAX_REPORT_DETAILS = 2000;

export function useReportReview(reviewId: string | undefined) {
  return useMutation({
    mutationFn: async ({ reason, details }: { reason: ReportReason; details: string }) =>
      unwrap(
        await apiClient.POST('/api/v1/reviews/{id}/reports', {
          params: { path: { id: reviewId! } },
          body: { reason, details: details.trim() || undefined },
        })
      ),
  });
}
