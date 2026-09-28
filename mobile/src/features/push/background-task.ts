import * as Notifications from 'expo-notifications';
import * as TaskManager from 'expo-task-manager';

import { describeEventType } from '@/features/activity/event-copy';

// The backend sends data-only FCM messages on purpose
// (internal/notifications/fcm.go: "clients translate event_type and
// interpolate data themselves") — Android will not display anything for a
// data-only message on its own, so without this task, push notifications
// are silently received and never shown. Runs in foreground, background,
// and terminated states (that's the point of a TaskManager task, not a
// plain addNotificationReceivedListener, which only fires in foreground).
export const BACKGROUND_NOTIFICATION_TASK = 'adera-background-notification';

// Unverified on a real device: FCM's data payload can surface either as
// direct keys on the task payload's `data` object, or JSON-encoded under
// `data.dataString`, depending on platform/OS version. Both are checked
// defensively since only a real push can confirm which one actually
// applies here.
function extractEventType(payloadData: unknown): string | undefined {
  if (!payloadData || typeof payloadData !== 'object') return undefined;
  const record = payloadData as Record<string, unknown>;
  if (typeof record.event_type === 'string') return record.event_type;
  if (typeof record.dataString === 'string') {
    try {
      const parsed = JSON.parse(record.dataString);
      if (typeof parsed.event_type === 'string') return parsed.event_type;
    } catch {
      // Not JSON — nothing more to try.
    }
  }
  return undefined;
}

TaskManager.defineTask<Notifications.NotificationTaskPayload>(BACKGROUND_NOTIFICATION_TASK, async ({ data, error }) => {
  if (error || !data || 'actionIdentifier' in data) return; // a tap response, not a received message
  const eventType = extractEventType((data as { data?: unknown }).data);
  const body = describeEventType(eventType);
  await Notifications.scheduleNotificationAsync({ content: { title: 'Adera', body }, trigger: null });
});

// Fire-and-forget, unconditional (not gated to signed-in state like
// PushRegistration's token registration) — this only builds a visible
// notification from whatever arrives; it makes no network calls itself.
Notifications.registerTaskAsync(BACKGROUND_NOTIFICATION_TASK);
