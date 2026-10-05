import * as Notifications from 'expo-notifications';
import * as TaskManager from 'expo-task-manager';

import { describeEventType } from '@/features/activity/event-copy';
import { parsePushPayload } from './payload';

// The backend sends data-only FCM messages on purpose
// (internal/notifications/fcm.go: "clients translate event_type and
// interpolate data themselves") — Android will not display anything for a
// data-only message on its own, so without this task, push notifications
// are silently received and never shown. Runs in foreground, background,
// and terminated states (that's the point of a TaskManager task, not a
// plain addNotificationReceivedListener, which only fires in foreground).
export const BACKGROUND_NOTIFICATION_TASK = 'adera-background-notification';

TaskManager.defineTask<Notifications.NotificationTaskPayload>(BACKGROUND_NOTIFICATION_TASK, async ({ data, error }) => {
  if (error || !data || 'actionIdentifier' in data) return; // a tap response, not a received message
  const { eventType, data: eventData } = parsePushPayload((data as { data?: unknown }).data);
  const body = describeEventType(eventType, eventData);
  await Notifications.scheduleNotificationAsync({ content: { title: 'Adera', body }, trigger: null });
});

// Fire-and-forget, unconditional (not gated to signed-in state like
// PushRegistration's token registration) — this only builds a visible
// notification from whatever arrives; it makes no network calls itself.
Notifications.registerTaskAsync(BACKGROUND_NOTIFICATION_TASK);
