import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { ActivityIndicator, Alert, FlatList, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { apiClient, unwrap } from '@/api/client';
import { friendlyAuthError } from '@/auth/friendly-error';
import { Button } from '@/components/button';
import { QueryError } from '@/components/query-error';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { formatRelative } from '@/lib/format';

const SESSIONS_KEY = ['auth', 'sessions'];

export default function SessionsScreen() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const sessions = useQuery({
    queryKey: SESSIONS_KEY,
    queryFn: async () => unwrap(await apiClient.GET('/api/v1/auth/sessions')).data,
  });
  const revoke = useMutation({
    mutationFn: async (id: string) =>
      unwrap(await apiClient.DELETE('/api/v1/auth/sessions/{id}', { params: { path: { id } } })),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: SESSIONS_KEY }),
  });

  function confirmRevoke(id: string) {
    // Signs that device out immediately; easy to tap by accident.
    Alert.alert(t('sessions.revokeTitle'), t('sessions.revokeBody'), [
      { text: t('common.cancel'), style: 'cancel' },
      { text: t('sessions.revoke'), style: 'destructive', onPress: () => revoke.mutate(id) },
    ]);
  }

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <FlatList
          data={sessions.data ?? []}
          keyExtractor={(session, index) => session.id ?? String(index)}
          contentContainerStyle={styles.list}
          ListHeaderComponent={
            // A failed revoke is reported above the list, which stays usable.
            revoke.error ? <ThemedText style={styles.error}>{friendlyAuthError(revoke.error)}</ThemedText> : null
          }
          ListEmptyComponent={
            sessions.isPending ? (
              <ActivityIndicator style={styles.centered} />
            ) : sessions.isError ? (
              <QueryError onRetry={() => sessions.refetch()} retrying={sessions.isRefetching} />
            ) : null
          }
          renderItem={({ item }) => (
            <ThemedView type="backgroundElement" style={styles.row}>
              <ThemedView style={styles.rowText} type="backgroundElement">
                <ThemedText type="smallBold">{item.device_info || t('sessions.unknownDevice')}</ThemedText>
                <ThemedText type="small" themeColor="textSecondary">
                  {item.current
                    ? t('sessions.thisDevice')
                    : item.last_used_at
                      ? t('sessions.lastUsed', { when: formatRelative(item.last_used_at) })
                      : t('sessions.lastUsedUnknown')}
                </ThemedText>
              </ThemedView>
              {!item.current && item.id ? (
                <Button
                  title={t('sessions.revoke')}
                  variant="secondary"
                  onPress={() => confirmRevoke(item.id!)}
                  loading={revoke.isPending && revoke.variables === item.id}
                />
              ) : null}
            </ThemedView>
          )}
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
    padding: Spacing.three,
    gap: Spacing.two,
  },
  centered: {
    padding: Spacing.four,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    borderRadius: Spacing.two,
    padding: Spacing.three,
    gap: Spacing.two,
  },
  rowText: {
    flex: 1,
    gap: Spacing.half,
  },
  error: {
    color: '#D64545',
    paddingBottom: Spacing.two,
  },
});
