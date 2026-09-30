import { router } from 'expo-router';
import { useMemo, useState } from 'react';
import { ActivityIndicator, FlatList, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { Button } from '@/components/button';
import { QueryError } from '@/components/query-error';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { FilterChip } from '@/features/search/filter-chip';
import { NotificationItem } from '@/features/activity/notification-item';
import { resolveNotificationTargetId } from '@/features/activity/resolve-navigation';
import { useMarkAllRead, useMarkNotificationRead, useNotifications, useUnreadCount } from '@/features/activity/queries';
import type { components } from '@/api/schema';

type Notification = components['schemas']['Notification'];

export default function ActivityScreen() {
  const [unreadOnly, setUnreadOnly] = useState(false);
  const notifications = useNotifications(unreadOnly);
  const unreadCount = useUnreadCount();
  const markRead = useMarkNotificationRead();
  const markAllRead = useMarkAllRead();

  const items = useMemo(() => notifications.data?.pages.flatMap((page) => page.data ?? []) ?? [], [notifications.data]);

  async function handlePress(notification: Notification) {
    if (!notification.read_at) markRead.mutate(notification.id);

    const targetId = await resolveNotificationTargetId(notification);
    if (targetId) router.push(`/target/${targetId}`);
  }

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['top', 'bottom']}>
        <FlatList
          data={items}
          keyExtractor={(item) => item.id}
          contentContainerStyle={styles.list}
          onEndReachedThreshold={0.5}
          onEndReached={() => {
            if (notifications.hasNextPage && !notifications.isFetchingNextPage) notifications.fetchNextPage();
          }}
          renderItem={({ item }) => <NotificationItem notification={item} onPress={() => handlePress(item)} />}
          ItemSeparatorComponent={() => <View style={{ height: Spacing.two }} />}
          ListHeaderComponent={
            <View style={styles.header}>
              <View style={styles.headerRow}>
                <ThemedText type="title">Activity</ThemedText>
                {(unreadCount.data?.unread_count ?? 0) > 0 ? (
                  <Button
                    title="Mark all read"
                    variant="secondary"
                    onPress={() => markAllRead.mutate()}
                    disabled={markAllRead.isPending}
                  />
                ) : null}
              </View>
              <View style={styles.filterRow}>
                <FilterChip label="All" selected={!unreadOnly} onPress={() => setUnreadOnly(false)} />
                <FilterChip label="Unread" selected={unreadOnly} onPress={() => setUnreadOnly(true)} />
              </View>
            </View>
          }
          ListEmptyComponent={
            notifications.isPending ? (
              <ActivityIndicator style={styles.centered} />
            ) : notifications.isError ? (
              <QueryError onRetry={() => notifications.refetch()} retrying={notifications.isRefetching} />
            ) : (
              <ThemedText type="small" themeColor="textSecondary" style={styles.centered}>
                {unreadOnly ? "You're all caught up." : 'Nothing here yet.'}
              </ThemedText>
            )
          }
          ListFooterComponent={notifications.isFetchingNextPage ? <ActivityIndicator style={styles.centered} /> : null}
        />
      </SafeAreaView>
    </ThemedView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  safeArea: {
    flex: 1,
  },
  list: {
    padding: Spacing.four,
    gap: Spacing.three,
  },
  header: {
    gap: Spacing.three,
    marginBottom: Spacing.two,
  },
  headerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  filterRow: {
    flexDirection: 'row',
    gap: Spacing.two,
  },
  centered: {
    padding: Spacing.four,
  },
});
