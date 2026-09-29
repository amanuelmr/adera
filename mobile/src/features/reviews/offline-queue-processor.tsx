import { useEffect } from 'react';
import { AppState } from 'react-native';

import { onConnectivityChange } from '@/lib/network-status';
import { processQueue } from './offline-queue';

function drain(): void {
  processQueue().catch((err) => console.warn('Offline queue run failed', err));
}

/**
 * Drains the offline queue on app start, whenever connectivity returns, and
 * whenever the app comes back to the foreground — the last covers jobs held
 * back by a server outage while the device itself stayed online.
 */
export function OfflineQueueProcessor() {
  useEffect(() => {
    drain();
    let wasConnected: boolean | undefined;
    const unsubscribeConnectivity = onConnectivityChange((connected) => {
      if (connected && wasConnected === false) drain();
      wasConnected = connected;
    });
    const appState = AppState.addEventListener('change', (state) => {
      if (state === 'active') drain();
    });
    return () => {
      unsubscribeConnectivity();
      appState.remove();
    };
  }, []);

  return null;
}
