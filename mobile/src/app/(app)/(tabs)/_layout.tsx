import { NativeTabs } from 'expo-router/unstable-native-tabs';
import { useTranslation } from 'react-i18next';

import { useAuth } from '@/auth/context';
import { useUnreadCount } from '@/features/activity/queries';
import { useTheme } from '@/hooks/use-theme';

// The platform's own tab bar (UITabBar / Material bottom navigation), with
// SF Symbols on iOS and Material Symbols on Android, in the brand green.
// Native tabs have no header: each tab screen lays out its own title under
// the status bar.
export default function TabsLayout() {
  const { t } = useTranslation();
  const theme = useTheme();
  const { status } = useAuth();
  const signedIn = status === 'signedIn';
  const unreadCount = useUnreadCount(signedIn);
  const unread = unreadCount.data?.unread_count ?? 0;

  return (
    <NativeTabs
      tintColor={theme.primary}
      iconColor={{ default: theme.textSecondary, selected: theme.primary }}
      labelStyle={{ color: theme.textSecondary }}
      backgroundColor={theme.surface}>
      <NativeTabs.Trigger name="index">
        <NativeTabs.Trigger.Label>{t('tabs.home')}</NativeTabs.Trigger.Label>
        <NativeTabs.Trigger.Icon sf={{ default: 'map', selected: 'map.fill' }} md="explore" />
      </NativeTabs.Trigger>
      {/* Activity is only for signed-in readers. Hiding it remounts the tab
          bar, which only happens on sign-in and sign-out. */}
      <NativeTabs.Trigger name="activity" hidden={!signedIn}>
        <NativeTabs.Trigger.Label>{t('tabs.activity')}</NativeTabs.Trigger.Label>
        <NativeTabs.Trigger.Icon sf={{ default: 'bell', selected: 'bell.fill' }} md="notifications" />
        {unread > 0 ? <NativeTabs.Trigger.Badge>{unread > 99 ? '99+' : String(unread)}</NativeTabs.Trigger.Badge> : null}
      </NativeTabs.Trigger>
      <NativeTabs.Trigger name="account">
        <NativeTabs.Trigger.Label>{t('tabs.account')}</NativeTabs.Trigger.Label>
        <NativeTabs.Trigger.Icon
          sf={{ default: 'person.crop.circle', selected: 'person.crop.circle.fill' }}
          md="account_circle"
        />
      </NativeTabs.Trigger>
    </NativeTabs>
  );
}
