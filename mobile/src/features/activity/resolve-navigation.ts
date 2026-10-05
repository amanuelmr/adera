import { apiClient, unwrap } from '@/api/client';
import type { components } from '@/api/schema';

type Notification = components['schemas']['Notification'];

/**
 * Where tapping a notification should go, from the ids each event type puts
 * in `data` (internal/*: notifications.EnqueueTx call sites). Best-effort:
 * undefined means there's nowhere useful, and the tap just marks it read.
 */
export async function resolveNotificationRoute(notification: Notification): Promise<string | undefined> {
  const { data, event_type: eventType, subject_type: subjectType, subject_id: subjectId } = notification;

  // Evidence decisions: back to that review's evidence, where the status is.
  if (subjectType === 'evidence' && data.review_id) return `/review/${data.review_id}/evidence`;
  // An approved claim means a business to manage now; other outcomes leave
  // nothing to open.
  if (subjectType === 'claim') return eventType === 'claim.approved' ? '/businesses' : undefined;
  // A place the user added: open it only once it's live; hidden or removed
  // places aren't viewable.
  if (subjectType === 'target') {
    return eventType === 'target.approve' || eventType === 'target.restore' ? `/target/${subjectId}` : undefined;
  }

  const targetId = await resolveTargetId(notification);
  return targetId ? `/target/${targetId}` : undefined;

  async function resolveTargetId(n: Notification): Promise<string | undefined> {
    if (n.data.target_id) return n.data.target_id;
    if (subjectType === 'target') return subjectId;
    const reviewId = data.review_id ?? (subjectType === 'review' ? subjectId : undefined);
    if (!reviewId) return undefined;
    try {
      return unwrap(await apiClient.GET('/api/v1/reviews/{id}', { params: { path: { id: reviewId } } })).data.target_id;
    } catch {
      return undefined;
    }
  }
}
