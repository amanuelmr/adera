import { apiClient, unwrap } from '@/api/client';
import type { components } from '@/api/schema';

type Notification = components['schemas']['Notification'];

// Best-effort: a target id is either handed to us directly (most event
// types put it in `data`, or the notification's own subject is a target),
// or reachable by fetching the review first (its `target_id` field) when
// only a `review_id` is available. Anything else has nowhere to send the
// user, so we just mark it read.
export async function resolveNotificationTargetId(notification: Notification): Promise<string | undefined> {
  if (notification.data.target_id) return notification.data.target_id;
  if (notification.subject_type === 'target') return notification.subject_id;

  const reviewId = notification.data.review_id ?? (notification.subject_type === 'review' ? notification.subject_id : undefined);
  if (!reviewId) return undefined;

  try {
    const review = unwrap(await apiClient.GET('/api/v1/reviews/{id}', { params: { path: { id: reviewId } } })).data;
    return review.target_id;
  } catch {
    return undefined;
  }
}
