import en from '@/lib/locales/en.json';
import i18n from '@/lib/i18n';

// Generic, per-event-type-prefix copy — not the real interpolated content
// (resolving the actual target/reviewer name) a richer activity feed could
// show, just enough that every notification means something to read.
// Shared between the push background task (src/features/push/background-task.ts,
// which needs *a* visible notification the instant a data-only FCM message
// arrives) and the in-app activity inbox list, rather than duplicated.
// Prefixes match internal/*/*.go's notifications.EnqueueTx call sites.
const KNOWN_PREFIXES = new Set(['response', 'review', 'target', 'evidence', 'claim', 'report', 'privacy']);

// Events whose meaning the prefix alone gets wrong: a place being approved
// isn't "updated", it's something the user can now act on.
const SPECIFIC: Record<string, keyof typeof en.events> = {
  'target.approve': 'targetApproved',
  'target.restore': 'targetApproved',
  // For business members: someone reviewed their place. Not "your review was updated".
  'review.received': 'reviewReceived',
};

export function describeEventType(eventType: string | undefined, data: Record<string, string> = {}): string {
  const prefix = eventType?.split('.')[0];
  const key = (eventType && SPECIFIC[eventType]) || (prefix && KNOWN_PREFIXES.has(prefix) ? prefix : 'default');
  // The background task can run headless before i18n has initialized; the
  // English copy is always available then, rather than a raw key.
  if (!i18n.isInitialized) return interpolate(en.events[key as keyof typeof en.events], data);
  return i18n.t(`events.${key}`, data);
}

function interpolate(text: string, data: Record<string, string>): string {
  return text.replace(/\{\{(\w+)\}\}/g, (match, name: string) => data[name] ?? match);
}
