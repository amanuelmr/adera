import { useMemo, useSyncExternalStore } from 'react';

import { getQueueSnapshot, subscribeQueue, type QueueJob } from './offline-queue';

/** Pending and server-rejected jobs, for a visible indicator (docs/mobile-plan.md §5). */
export function useOfflineQueue(): { pendingCount: number; failedJobs: QueueJob[] } {
  const queue = useSyncExternalStore(subscribeQueue, getQueueSnapshot);
  return useMemo(
    () => ({
      pendingCount: queue.filter((job) => !job.failure).length,
      failedJobs: queue.filter((job) => job.failure),
    }),
    [queue]
  );
}
