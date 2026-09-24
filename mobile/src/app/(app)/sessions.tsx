import { useCallback, useEffect, useState } from 'react';
import { FlatList, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { apiClient, unwrap } from '@/api/client';
import { friendlyAuthError } from '@/auth/friendly-error';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import type { components } from '@/api/schema';

type Session = components['schemas']['Session'];

export default function SessionsScreen() {
  const [sessions, setSessions] = useState<Session[]>();
  const [error, setError] = useState<string>();
  const [revokingId, setRevokingId] = useState<string>();

  const load = useCallback(async () => {
    setError(undefined);
    try {
      const { data } = unwrap(await apiClient.GET('/api/v1/auth/sessions'));
      setSessions(data);
    } catch (err) {
      setError(friendlyAuthError(err));
    }
  }, []);

  useEffect(() => {
    // Fetch-on-mount; a data-fetching library (TanStack Query, planned for
    // the data-heavy screens in the next feature slice) would own this
    // instead, but isn't installed yet for one simple list.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    load();
  }, [load]);

  async function revoke(id: string) {
    setRevokingId(id);
    try {
      unwrap(await apiClient.DELETE('/api/v1/auth/sessions/{id}', { params: { path: { id } } }));
      await load();
    } catch (err) {
      setError(friendlyAuthError(err));
    } finally {
      setRevokingId(undefined);
    }
  }

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        {error ? (
          <ThemedText style={styles.error}>{error}</ThemedText>
        ) : (
          <FlatList
            data={sessions}
            keyExtractor={(session) => session.id ?? ''}
            contentContainerStyle={styles.list}
            renderItem={({ item }) => (
              <ThemedView type="backgroundElement" style={styles.row}>
                <ThemedView style={styles.rowText} type="backgroundElement">
                  <ThemedText type="smallBold">{item.device_info ?? 'Unknown device'}</ThemedText>
                  <ThemedText type="small" themeColor="textSecondary">
                    {item.current ? 'This device' : `Last used ${item.last_used_at ?? 'unknown'}`}
                  </ThemedText>
                </ThemedView>
                {!item.current && item.id ? (
                  <Button
                    title="Revoke"
                    variant="secondary"
                    onPress={() => revoke(item.id!)}
                    loading={revokingId === item.id}
                  />
                ) : null}
              </ThemedView>
            )}
          />
        )}
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
    padding: Spacing.three,
  },
});
