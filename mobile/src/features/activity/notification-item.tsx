import { useTranslation } from 'react-i18next';
import { Pressable, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import { Spacing } from '@/constants/theme';
import { formatDateTime, formatRelative } from '@/lib/format';
import { describeEventType } from './event-copy';
import type { components } from '@/api/schema';

type Notification = components['schemas']['Notification'];

export function NotificationItem({ notification, onPress }: { notification: Notification; onPress: () => void }) {
  const theme = useTheme();
  const { t } = useTranslation();
  const unread = !notification.read_at;

  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={`${unread ? `${t('activity.unread')}. ` : ''}${describeEventType(notification.event_type, notification.data)}`}
      style={[styles.row, { backgroundColor: theme.backgroundElement }]}>
      {unread ? <View style={[styles.dot, { backgroundColor: theme.text }]} /> : <View style={styles.dot} />}
      <View style={styles.body}>
        <ThemedText type={unread ? 'smallBold' : 'small'}>{describeEventType(notification.event_type, notification.data)}</ThemedText>
        <ThemedText type="small" themeColor="textSecondary" accessibilityLabel={formatDateTime(notification.created_at)}>
          {formatRelative(notification.created_at)}
        </ThemedText>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: Spacing.two,
    padding: Spacing.three,
    borderRadius: Spacing.two,
  },
  dot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    marginTop: Spacing.one,
  },
  body: {
    flex: 1,
    gap: Spacing.half,
  },
});
