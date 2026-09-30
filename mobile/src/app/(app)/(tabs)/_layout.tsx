import { Tabs } from 'expo-router';

import { useAuth } from '@/auth/context';
import { useUnreadCount } from '@/features/activity/queries';

export default function TabsLayout() {
  const { status } = useAuth();
  const signedIn = status === 'signedIn';
  const unreadCount = useUnreadCount(signedIn);
  const unread = unreadCount.data?.unread_count ?? 0;

  return (
    <Tabs>
      <Tabs.Screen name="index" options={{ title: 'Home' }} />
      <Tabs.Protected guard={signedIn}>
        <Tabs.Screen name="activity" options={{ title: 'Activity', tabBarBadge: unread > 0 ? String(unread) : undefined }} />
      </Tabs.Protected>
      <Tabs.Screen name="account" options={{ title: 'Account' }} />
    </Tabs>
  );
}
