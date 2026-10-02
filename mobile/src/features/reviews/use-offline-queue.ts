import { useEffect, useMemo, useState, useSyncExternalStore } from 'react';

import { getCurrentUserId } from '@/auth/storage';
import { getQueueSnapshot, subscribeQueue, type QueueJob } from './offline-queue';

/**
 * The signed-in account's pending and failed jobs, for a visible indicator
 * (docs/mobile-plan.md §5). Other accounts' queued jobs on a shared device
 * are theirs to see, not this user's.
 */
export function useOfflineQueue(): { pendingCount: number; failedJobs: QueueJob[] } {
  const queue = useSyncExternalStore(subscribeQueue, getQueueSnapshot);
  const [userId, setUserId] = useState<string | null>();
  useEffect(() => {
    let cancelled = false;
    getCurrentUserId().then((id) => {
      if (!cancelled) setUserId(id);
    });
    return () => {
      cancelled = true;
    };
  }, [queue]);
  return useMemo(() => {
    const mine = queue.filter((job) => job.userId === userId);
    return {
      pendingCount: mine.filter((job) => !job.failure).length,
      failedJobs: mine.filter((job) => job.failure),
    };
  }, [queue, userId]);
}
