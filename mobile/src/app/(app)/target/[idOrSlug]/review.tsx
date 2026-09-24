import { useMutation } from '@tanstack/react-query';
import { router, Stack, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ActivityIndicator, ScrollView, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ApiError } from '@/api/client';
import type { components } from '@/api/schema';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { generateIdempotencyKey } from '@/features/reviews/idempotency';
import { enqueueReviewSubmission } from '@/features/reviews/offline-queue';
import { submitReview, submitReviewEdit, useCriteria, useReview } from '@/features/reviews/queries';
import { isConnected } from '@/lib/network-status';
import { INITIAL_REVIEW_FORM, type ReviewFormState } from '@/features/reviews/types';
import { CriteriaStep } from '@/features/reviews/steps/criteria-step';
import { ContextStep } from '@/features/reviews/steps/context-step';
import { DisclosureStep } from '@/features/reviews/steps/disclosure-step';
import { RatingStep } from '@/features/reviews/steps/rating-step';
import { TextPhotoStep } from '@/features/reviews/steps/text-photo-step';
import { useTarget } from '@/features/targets/queries';

const STEP_COUNT = 5;
const MIN_BODY_LENGTH = 20;

// Existing media is left untouched (photos here are only newly added ones —
// see submitReviewEdit); discovery_source/expectation_match come back from
// the API as plain strings rather than their enum types (a spec looseness,
// not a validation gap — only enum members are ever actually stored).
function reviewToFormState(review: components['schemas']['Review']): ReviewFormState {
  return {
    overallRating: review.overall_rating,
    criterionScores: review.criterion_scores ?? {},
    title: review.title ?? '',
    body: review.body ?? '',
    photos: [],
    experienceDate: review.experience_date?.slice(0, 10) ?? '',
    discoverySource: review.discovery_source as components['schemas']['DiscoverySource'] | undefined,
    expectationMatch: review.expectation_match as components['schemas']['ExpectationMatch'] | undefined,
    pricePaid: review.price_paid != null ? String(review.price_paid) : '',
    incentiveType: review.incentive_type ?? 'none',
    materialConnection: review.material_connection ?? 'none',
    disclosureDetails: review.disclosure_details ?? '',
  };
}

export default function WriteReviewScreen() {
  const { idOrSlug, reviewId } = useLocalSearchParams<{ idOrSlug: string; reviewId?: string }>();
  const isEditing = !!reviewId;
  const target = useTarget(idOrSlug);
  const existingReview = useReview(reviewId);
  const criteria = useCriteria(target.data?.category_id);
  const theme = useTheme();
  const { t } = useTranslation();

  const [step, setStep] = useState(0);
  const [form, setForm] = useState<ReviewFormState>(INITIAL_REVIEW_FORM);
  const [idempotencyKey] = useState(generateIdempotencyKey);

  // Hydrates the form from the fetched review exactly once it arrives —
  // adjusting state during render (not in an effect) so there's no extra
  // render pass showing the blank form first. See filters-sheet.tsx for the
  // same pattern.
  const [hydratedFrom, setHydratedFrom] = useState<string>();
  if (isEditing && existingReview.data && hydratedFrom !== reviewId) {
    setHydratedFrom(reviewId);
    setForm(reviewToFormState(existingReview.data));
  }

  function update<K extends keyof ReviewFormState>(key: K, value: ReviewFormState[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  const submit = useMutation({
    mutationFn: async (): Promise<{ queued: boolean; failedPhotoCount: number }> => {
      const targetId = target.data!.id!;

      if (isEditing) {
        // Full-replace PUT with no Idempotency-Key support — editing stays
        // online-only rather than growing the offline queue a third job type.
        if (!(await isConnected())) throw new Error('offline');
        const result = await submitReviewEdit(reviewId!, targetId, form, existingReview.data!.version!);
        return { queued: false, failedPhotoCount: result.retryablePhotoUris.length };
      }

      if (!(await isConnected())) {
        await enqueueReviewSubmission({ targetId, form, idempotencyKey });
        return { queued: true, failedPhotoCount: 0 };
      }

      try {
        const result = await submitReview(targetId, form, idempotencyKey);
        if (result.retryablePhotoUris.length > 0) {
          // The review itself is live; only the photos need another attempt.
          await enqueueReviewSubmission({
            targetId,
            form,
            idempotencyKey,
            reviewId: result.reviewId,
            pendingPhotoUris: result.retryablePhotoUris,
          });
        }
        return { queued: false, failedPhotoCount: result.retryablePhotoUris.length };
      } catch (err) {
        if (err instanceof ApiError) throw err; // a real rejection — nothing offline retry can fix
        // Network dropped mid-attempt — the review may or may not have been
        // created; queuing a fresh attempt is safe either way because the
        // Idempotency-Key makes a duplicate POST /reviews a no-op.
        await enqueueReviewSubmission({ targetId, form, idempotencyKey });
        return { queued: true, failedPhotoCount: 0 };
      }
    },
  });

  if (target.isPending || (isEditing && existingReview.isPending)) {
    return (
      <ThemedView style={styles.centered}>
        <ActivityIndicator />
      </ThemedView>
    );
  }

  if (target.isError || !target.data?.id || (isEditing && (existingReview.isError || !existingReview.data))) {
    return (
      <ThemedView style={styles.centered}>
        <ThemedText type="small" themeColor="textSecondary">
          Couldn&apos;t load this {isEditing ? 'review' : 'target'}. Check your connection and try again.
        </ThemedText>
      </ThemedView>
    );
  }

  if (submit.isSuccess) {
    return (
      <ThemedView style={styles.centered}>
        <ThemedText type="subtitle">
          {submit.data.queued ? 'Saved — sending soon' : isEditing ? 'Review updated' : 'Thanks for your review!'}
        </ThemedText>
        <ThemedText type="small" themeColor="textSecondary" style={styles.confirmationBody}>
          {submit.data.queued
            ? "You're offline right now. We'll send this the moment you're back online."
            : `It's live on ${target.data.name}'s page.`}
          {submit.data.failedPhotoCount > 0
            ? ` ${submit.data.failedPhotoCount} photo${submit.data.failedPhotoCount === 1 ? '' : 's'} couldn't be uploaded and will retry automatically.`
            : ''}
        </ThemedText>
        <Button title={t('common.backToTarget')} onPress={() => router.replace(`/target/${idOrSlug}`)} />
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
      ? t('errors.reviewCooldown')
      : submit.error instanceof ApiError && submit.error.code === 'rate_limited'
        ? t('errors.reviewRateLimited')
        : submit.error instanceof ApiError && submit.error.code === 'precondition_failed'
          ? t('errors.reviewStale')
          : submit.error instanceof ApiError && submit.error.code === 'validation_failed'
            ? (Object.values(submit.error.details ?? {})[0] ?? submit.error.message)
            : submit.error instanceof Error && submit.error.message === 'offline'
              ? t('errors.reviewOffline')
              : t('errors.reviewGeneric')
    : undefined;

  return (
    <ThemedView style={styles.container}>
      <Stack.Screen options={{ title: `${isEditing ? t('nav.editReview') : t('nav.writeReview')} · ${step + 1}/${STEP_COUNT}` }} />
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
          {step > 0 ? (
            <Button title={t('common.back')} variant="secondary" onPress={() => setStep((s) => s - 1)} />
          ) : (
            <View style={styles.spacer} />
          )}
          {step < STEP_COUNT - 1 ? (
            <Button title={t('common.next')} onPress={() => setStep((s) => s + 1)} disabled={!canProceed} />
          ) : (
            <Button
              title={isEditing ? t('common.saveChanges') : t('common.submit')}
              onPress={() => submit.mutate()}
              loading={submit.isPending}
              disabled={!canProceed}
            />
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
