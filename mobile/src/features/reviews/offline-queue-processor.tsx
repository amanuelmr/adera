import { useEffect } from 'react';

import { onConnectivityChange } from '@/lib/network-status';
import { processQueue } from './offline-queue';

/** Drains the offline queue on app start and whenever connectivity returns. */
export function OfflineQueueProcessor() {
  useEffect(() => {
    processQueue();
    let wasConnected: boolean | undefined;
    return onConnectivityChange((connected) => {
      if (connected && wasConnected === false) processQueue();
      wasConnected = connected;
    });
  }, []);

  return null;
}
