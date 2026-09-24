import { useMutation } from '@tanstack/react-query';
import { router, Stack, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { ActivityIndicator, ScrollView, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ApiError } from '@/api/client';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { generateIdempotencyKey } from '@/features/reviews/idempotency';
import { submitReview, useCriteria } from '@/features/reviews/queries';
import { INITIAL_REVIEW_FORM, type ReviewFormState } from '@/features/reviews/types';
import { CriteriaStep } from '@/features/reviews/steps/criteria-step';
import { ContextStep } from '@/features/reviews/steps/context-step';
import { DisclosureStep } from '@/features/reviews/steps/disclosure-step';
import { RatingStep } from '@/features/reviews/steps/rating-step';
import { TextPhotoStep } from '@/features/reviews/steps/text-photo-step';
import { useTarget } from '@/features/targets/queries';

const STEP_COUNT = 5;
const MIN_BODY_LENGTH = 20;

export default function WriteReviewScreen() {
  const { idOrSlug } = useLocalSearchParams<{ idOrSlug: string }>();
  const target = useTarget(idOrSlug);
  const criteria = useCriteria(target.data?.category_id);
  const theme = useTheme();

  const [step, setStep] = useState(0);
  const [form, setForm] = useState<ReviewFormState>(INITIAL_REVIEW_FORM);
  const [idempotencyKey] = useState(generateIdempotencyKey);

  function update<K extends keyof ReviewFormState>(key: K, value: ReviewFormState[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  const submit = useMutation({
    mutationFn: () => submitReview(target.data!.id!, form, idempotencyKey),
  });

  if (target.isPending) {
    return (
      <ThemedView style={styles.centered}>
        <ActivityIndicator />
      </ThemedView>
    );
  }

  if (target.isError || !target.data?.id) {
    return (
      <ThemedView style={styles.centered}>
        <ThemedText type="small" themeColor="textSecondary">
          Couldn&apos;t load this target. Check your connection and try again.
        </ThemedText>
      </ThemedView>
    );
  }

  if (submit.isSuccess) {
    return (
      <ThemedView style={styles.centered}>
        <ThemedText type="subtitle">Thanks for your review!</ThemedText>
        <ThemedText type="small" themeColor="textSecondary" style={styles.confirmationBody}>
          It&apos;s live on {target.data.name}&apos;s page.
          {submit.data.failedPhotoCount > 0
            ? ` ${submit.data.failedPhotoCount} photo${submit.data.failedPhotoCount === 1 ? '' : 's'} couldn't be uploaded — you can add them later.`
            : ''}
        </ThemedText>
        <Button title="Back to target" onPress={() => router.replace(`/target/${idOrSlug}`)} />
      </ThemedView>
    );
  }

  const missingRequiredCriteria = (criteria.data ?? []).filter(
    (criterion) => criterion.required && criterion.code && !form.criterionScores[criterion.code]
  );
  const detailsRequired = form.incentiveType === 'other' || form.materialConnection === 'other';

  const canProceed = [
    form.overallRating != null,
    !criteria.isPending && missingRequiredCriteria.length === 0,
    form.body.trim().length >= MIN_BODY_LENGTH,
    true,
    !detailsRequired || form.disclosureDetails.trim().length > 0,
  ][step];

  const submitError = submit.error
    ? submit.error instanceof ApiError && submit.error.code === 'cooldown_active'
      ? "You've already reviewed this in the last 30 days."
      : submit.error instanceof ApiError && submit.error.code === 'rate_limited'
        ? "You've submitted a few reviews already today — try again later."
        : submit.error instanceof ApiError && submit.error.code === 'validation_failed'
          ? (Object.values(submit.error.details ?? {})[0] ?? submit.error.message)
          : "Couldn't submit your review. Check your connection and try again."
    : undefined;

  return (
    <ThemedView style={styles.container}>
      <Stack.Screen options={{ title: `Write a review · ${step + 1}/${STEP_COUNT}` }} />
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <View style={styles.dots}>
          {Array.from({ length: STEP_COUNT }).map((_, index) => (
            <View
              key={index}
              style={[styles.dot, { backgroundColor: index <= step ? theme.text : theme.backgroundElement }]}
            />
          ))}
        </View>

        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
          {step === 0 ? (
            <RatingStep value={form.overallRating} onChange={(rating) => update('overallRating', rating)} />
          ) : step === 1 ? (
            <CriteriaStep
              categoryId={target.data.category_id}
              scores={form.criterionScores}
              onChange={(code, rating) => update('criterionScores', { ...form.criterionScores, [code]: rating })}
            />
          ) : step === 2 ? (
            <TextPhotoStep
              title={form.title}
              body={form.body}
              photos={form.photos}
              onChangeTitle={(title) => update('title', title)}
              onChangeBody={(body) => update('body', body)}
              onChangePhotos={(photos) => update('photos', photos)}
            />
          ) : step === 3 ? (
            <ContextStep
              experienceDate={form.experienceDate}
              discoverySource={form.discoverySource}
              expectationMatch={form.expectationMatch}
              pricePaid={form.pricePaid}
              onChangeExperienceDate={(value) => update('experienceDate', value)}
              onChangeDiscoverySource={(value) => update('discoverySource', value)}
              onChangeExpectationMatch={(value) => update('expectationMatch', value)}
              onChangePricePaid={(value) => update('pricePaid', value)}
            />
          ) : (
            <DisclosureStep
              incentiveType={form.incentiveType}
              materialConnection={form.materialConnection}
              disclosureDetails={form.disclosureDetails}
              onChangeIncentiveType={(value) => update('incentiveType', value)}
              onChangeMaterialConnection={(value) => update('materialConnection', value)}
              onChangeDisclosureDetails={(value) => update('disclosureDetails', value)}
            />
          )}
        </ScrollView>

        {submitError ? (
          <ThemedText type="small" style={styles.error}>
            {submitError}
          </ThemedText>
        ) : null}

        <View style={styles.actions}>
          {step > 0 ? <Button title="Back" variant="secondary" onPress={() => setStep((s) => s - 1)} /> : <View style={styles.spacer} />}
          {step < STEP_COUNT - 1 ? (
            <Button title="Next" onPress={() => setStep((s) => s + 1)} disabled={!canProceed} />
          ) : (
            <Button title="Submit" onPress={() => submit.mutate()} loading={submit.isPending} disabled={!canProceed} />
          )}
        </View>
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
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: Spacing.three,
    padding: Spacing.four,
  },
  confirmationBody: {
    textAlign: 'center',
  },
  dots: {
    flexDirection: 'row',
    justifyContent: 'center',
    gap: Spacing.two,
    paddingVertical: Spacing.three,
  },
  dot: {
    width: 8,
    height: 8,
    borderRadius: 4,
  },
  content: {
    flexGrow: 1,
    paddingHorizontal: Spacing.four,
  },
  actions: {
    flexDirection: 'row',
    gap: Spacing.two,
    padding: Spacing.four,
  },
  spacer: {
    flex: 1,
  },
  error: {
    color: '#D64545',
    paddingHorizontal: Spacing.four,
    paddingBottom: Spacing.two,
  },
});
