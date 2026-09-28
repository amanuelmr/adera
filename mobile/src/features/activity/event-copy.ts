// Generic, per-event-type-prefix copy — not the real interpolated content
// (resolving the actual target/reviewer name) a richer activity feed could
// show, just enough that every notification means something to read.
// Shared between the push background task (src/features/push/background-task.ts,
// which needs *a* visible notification the instant a data-only FCM message
// arrives) and the in-app activity inbox list, rather than duplicated.
// Prefixes match internal/*/*.go's notifications.EnqueueTx call sites:
// response.*, review.*, target.*, evidence.*, claim.*, report.*, privacy.*.
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

export function describeEventType(eventType: string | undefined): string {
  const prefix = eventType?.split('.')[0];
  return (prefix && EVENT_COPY[prefix]) || DEFAULT_BODY;
}
