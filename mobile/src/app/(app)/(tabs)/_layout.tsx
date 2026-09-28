import { Tabs } from 'expo-router';

import { useUnreadCount } from '@/features/activity/queries';

export default function TabsLayout() {
  const unreadCount = useUnreadCount();
  const unread = unreadCount.data?.unread_count ?? 0;

  return (
    <Tabs>
      <Tabs.Screen name="index" options={{ title: 'Home' }} />
      <Tabs.Screen name="activity" options={{ title: 'Activity', tabBarBadge: unread > 0 ? String(unread) : undefined }} />
      <Tabs.Screen name="account" options={{ title: 'Account' }} />
    </Tabs>
  );
}
