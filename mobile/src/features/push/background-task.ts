import * as Notifications from 'expo-notifications';
import * as TaskManager from 'expo-task-manager';

// The backend sends data-only FCM messages on purpose
// (internal/notifications/fcm.go: "clients translate event_type and
// interpolate data themselves") — Android will not display anything for a
// data-only message on its own, so without this task, push notifications
// are silently received and never shown. Runs in foreground, background,
// and terminated states (that's the point of a TaskManager task, not a
// plain addNotificationReceivedListener, which only fires in foreground).
export const BACKGROUND_NOTIFICATION_TASK = 'adera-background-notification';

// Generic, per-event-type-prefix copy — not the real interpolated content
// (resolving the actual target/reviewer name) that a full activity inbox
// would show. That's Phase 2 (docs/mobile-plan.md §9); this only has to make
// sure *something* visible happens for each backend event_type prefix
// (see internal/*/,go's notifications.EnqueueTx call sites: response.*,
// review.*, target.*, evidence.*, claim.*, report.*, privacy.*).
const EVENT_COPY: Record<string, string> = {
  response: 'You have a new response to your review.',
  review: 'One of your reviews was updated.',
  target: 'A place you added was updated.',
  evidence: 'Your evidence submission was reviewed.',
  claim: 'Your business claim status changed.',
  report: 'A report you filed was reviewed.',
  privacy: 'Your data request was updated.',
};
const DEFAULT_BODY = 'You have a new update.';

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
  const prefix = eventType?.split('.')[0];
  const body = (prefix && EVENT_COPY[prefix]) || DEFAULT_BODY;
  await Notifications.scheduleNotificationAsync({ content: { title: 'Adera', body }, trigger: null });
});

// Fire-and-forget, unconditional (not gated to signed-in state like
// PushRegistration's token registration) — this only builds a visible
// notification from whatever arrives; it makes no network calls itself.
Notifications.registerTaskAsync(BACKGROUND_NOTIFICATION_TASK);
