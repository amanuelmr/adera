import { focusManager, QueryClient } from '@tanstack/react-query';
import { AppState, Platform } from 'react-native';

// 5-minute default staleTime gives list/browse screens stale-while-revalidate
// behavior (docs/mobile-plan.md §5): cached data renders immediately, a
// background refetch keeps it current. Trust-critical numbers (aggregates,
// Reality Check) need network-first instead — those queries override this
// with staleTime: 0 individually rather than changing the default here.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,
    },
  },
});

// React Native has no window focus event, so TanStack never learns the app
// came back to the foreground. Feeding it AppState makes stale queries (the
// unread badge especially) refetch when the user returns to the app.
if (Platform.OS !== 'web') {
  AppState.addEventListener('change', (status) => focusManager.setFocused(status === 'active'));
}
