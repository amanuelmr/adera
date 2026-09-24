import { useSyncExternalStore } from 'react';

import { getQueueSnapshot, subscribeQueue } from './offline-queue';

/** Pending-sync count for a visible indicator (docs/mobile-plan.md §5). */
export function usePendingSyncCount(): number {
  return useSyncExternalStore(subscribeQueue, () => getQueueSnapshot().length);
}
