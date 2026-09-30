import { useState } from 'react';
import { StyleSheet, TextInput, View } from 'react-native';

import type { components } from '@/api/schema';
import { ApiError } from '@/api/client';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { useEditResponse, usePostResponse } from '@/features/reviews/queries';

type ListedReview = components['schemas']['ListedReview'];

export function OwnerReviewCard({ review, targetId }: { review: ListedReview; targetId: string }) {
  const theme = useTheme();
  const postResponse = usePostResponse(targetId);
  const editResponse = useEditResponse(targetId);
  const existing = review.business_response;

  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(existing?.body ?? '');

  const rating = review.overall_rating ?? 0;
  const mutation = existing ? editResponse : postResponse;

  function startEditing() {
    setDraft(existing?.body ?? '');
    setEditing(true);
  }

  function save() {
    if (!draft.trim() || !review.id) return;
    const onDone = () => setEditing(false);
    if (existing?.id) {
      editResponse.mutate({ responseId: existing.id, body: draft.trim() }, { onSuccess: onDone });
    } else {
      postResponse.mutate({ reviewId: review.id, body: draft.trim() }, { onSuccess: onDone });
    }
  }

  return (
    <ThemedView type="backgroundElement" style={styles.card}>
      <View style={styles.header}>
        <ThemedText type="smallBold">{review.reviewer_name ?? 'Anonymous'}</ThemedText>
        <ThemedText type="small" themeColor="textSecondary" accessibilityLabel={`Rated ${rating} out of 5`}>
          {'★'.repeat(rating)}
          {'☆'.repeat(Math.max(0, 5 - rating))}
        </ThemedText>
      </View>

      {review.title ? <ThemedText type="smallBold">{review.title}</ThemedText> : null}
      <ThemedText type="small">{review.body}</ThemedText>

      {existing && !editing ? (
        <ThemedView type="backgroundSelected" style={styles.response}>
          <ThemedText type="smallBold">Your response</ThemedText>
          <ThemedText type="small">{existing.body}</ThemedText>
        </ThemedView>
      ) : null}

      {editing ? (
        <View style={styles.editor}>
          <TextInput
            value={draft}
            onChangeText={setDraft}
            placeholder="Write a public response…"
            placeholderTextColor={theme.textSecondary}
            multiline
            maxLength={2000}
            style={[styles.input, { color: theme.text, backgroundColor: theme.background }]}
          />
          {mutation.error ? (
            <ThemedText type="small" style={styles.error}>
              {mutation.error instanceof ApiError ? mutation.error.message : "Couldn't save your response."}
            </ThemedText>
          ) : null}
          <View style={styles.actions}>
            <Button title="Cancel" variant="secondary" onPress={() => setEditing(false)} />
            <Button title="Save" onPress={save} loading={mutation.isPending} disabled={!draft.trim()} />
          </View>
        </View>
      ) : (
        <Button title={existing ? 'Edit response' : 'Respond'} variant="secondary" onPress={startEditing} />
      )}
    </ThemedView>
  );
}

const styles = StyleSheet.create({
  card: {
    borderRadius: Spacing.two,
    padding: Spacing.three,
    gap: Spacing.two,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  response: {
    borderRadius: Spacing.two,
    padding: Spacing.two,
    gap: Spacing.half,
  },
  editor: {
    gap: Spacing.two,
  },
  input: {
    minHeight: 88,
    borderRadius: Spacing.two,
    padding: Spacing.three,
    fontSize: 16,
    textAlignVertical: 'top',
  },
  actions: {
    flexDirection: 'row',
    gap: Spacing.two,
  },
  error: {
    color: '#D64545',
  },
});
