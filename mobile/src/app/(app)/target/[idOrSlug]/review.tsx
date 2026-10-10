import { useMutation } from '@tanstack/react-query';
import { router, Stack, useLocalSearchParams } from 'expo-router';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ActivityIndicator, ScrollView, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ApiError } from '@/api/client';
import { getCurrentUserId } from '@/auth/storage';
import type { components } from '@/api/schema';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { clearDraft, discardDraft, loadDraft, saveDraft, type ReviewDraft } from '@/features/reviews/drafts';
import { generateIdempotencyKey } from '@/features/reviews/idempotency';
import { enqueueReviewSubmission } from '@/features/reviews/offline-queue';
import { invalidateReviewCaches, submitReview, submitReviewEdit, useCriteria, useReview } from '@/features/reviews/queries';
import { isConnected } from '@/lib/network-status';
import { INITIAL_REVIEW_FORM, type ReviewFormState } from '@/features/reviews/types';
import { canProceedFromStep, REVIEW_STEP_COUNT } from '@/features/reviews/form-steps';
import { CriteriaStep } from '@/features/reviews/steps/criteria-step';
import { ContextStep } from '@/features/reviews/steps/context-step';
import { DisclosureStep } from '@/features/reviews/steps/disclosure-step';
import { RatingStep } from '@/features/reviews/steps/rating-step';
import { MAX_PHOTOS, TextPhotoStep } from '@/features/reviews/steps/text-photo-step';
import { recoverPendingPhoto } from '@/features/reviews/photos';
import { formatRelative } from '@/lib/format';
import { shareReview } from '@/features/share/share';
import { useTarget } from '@/features/targets/queries';


// Existing media is left untouched (photos here are only newly added ones —
// see submitReviewEdit); discovery_source/expectation_match come back from
// the API as plain strings rather than their enum types (a spec looseness,
// not a validation gap — only enum members are ever actually stored).
function withPhoto(form: ReviewFormState, uri: string | undefined): ReviewFormState {
  if (!uri || form.photos.length >= MAX_PHOTOS || form.photos.some((p) => p.uri === uri)) return form;
  return { ...form, photos: [...form.photos, { uri }] };
}

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
  // The version the form's contents came from — not whatever version the
  // cached review holds at submit time. A background refetch can bring in a
  // newer version (an edit from another device); sending that with this
  // older content would silently overwrite it instead of getting a 412.
  const [hydratedVersion, setHydratedVersion] = useState<number>();
  if (isEditing && existingReview.data && hydratedFrom !== reviewId) {
    setHydratedFrom(reviewId);
    setHydratedVersion(existingReview.data.version);
    setForm(reviewToFormState(existingReview.data));
  }

  // Drafts (new reviews only — an edit reloads from the server). 'checking'
  // until the saved draft and any photo Android delivered after killing the
  // app are loaded; 'prompt' asks whether to continue a found draft.
  const targetId = target.data?.id;
  const [userId, setUserId] = useState<string | null>();
  const [draftState, setDraftState] = useState<'checking' | 'prompt' | 'ready'>(isEditing ? 'ready' : 'checking');
  const [foundDraft, setFoundDraft] = useState<ReviewDraft>();
  const [recoveredPhoto, setRecoveredPhoto] = useState<string>();

  useEffect(() => {
    if (isEditing || !targetId) return;
    let cancelled = false;
    (async () => {
      const uid = await getCurrentUserId();
      const [draft, recovered] = await Promise.all([
        uid ? loadDraft(uid, targetId) : undefined,
        recoverPendingPhoto().catch(() => undefined),
      ]);
      if (cancelled) return;
      setUserId(uid);
      setRecoveredPhoto(recovered);
      if (draft) {
        setFoundDraft(draft);
        setDraftState('prompt');
      } else {
        if (recovered) setForm((prev) => withPhoto(prev, recovered));
        setDraftState('ready');
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [isEditing, targetId]);

  function resumeDraft() {
    if (!foundDraft) return;
    setForm(withPhoto(foundDraft.form, recoveredPhoto));
    setStep(Math.min(foundDraft.step, REVIEW_STEP_COUNT - 1));
    setDraftState('ready');
  }

  function startOver() {
    if (userId && targetId) discardDraft(userId, targetId, foundDraft).catch(() => undefined);
    setForm(withPhoto(INITIAL_REVIEW_FORM, recoveredPhoto));
    setStep(0);
    setDraftState('ready');
  }

  function update<K extends keyof ReviewFormState>(key: K, value: ReviewFormState[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  const submit = useMutation({
    mutationFn: async (): Promise<{
      queued: boolean;
      retryingPhotoCount: number;
      rejectedPhotoCount: number;
      reviewId?: string;
    }> => {
      const targetId = target.data!.id!;

      if (isEditing) {
        // Full-replace PUT with no Idempotency-Key support — editing stays
        // online-only rather than growing the offline queue a third job type.
        if (!(await isConnected())) throw new Error('offline');
        const result = await submitReviewEdit(reviewId!, targetId, form, hydratedVersion!);
        if (result.photos.retryable.length > 0) {
          // The edit is saved; only new photos need another attempt. A job
          // with a reviewId skips creation and just uploads them.
          await enqueueReviewSubmission({
            targetId,
            form,
            idempotencyKey,
            reviewId,
            pendingPhotoUris: result.photos.retryable,
          });
        }
        return {
          queued: false,
          retryingPhotoCount: result.photos.retryable.length,
          rejectedPhotoCount: result.photos.rejected,
          reviewId,
        };
      }

      if (!(await isConnected())) {
        await enqueueReviewSubmission({ targetId, form, idempotencyKey });
        return { queued: true, retryingPhotoCount: 0, rejectedPhotoCount: 0 };
      }

      try {
        const result = await submitReview(targetId, form, idempotencyKey);
        if (result.photos.retryable.length > 0) {
          // The review itself is live; only the photos need another attempt.
          await enqueueReviewSubmission({
            targetId,
            form,
            idempotencyKey,
            reviewId: result.reviewId,
            pendingPhotoUris: result.photos.retryable,
          });
        }
        return {
          queued: false,
          retryingPhotoCount: result.photos.retryable.length,
          rejectedPhotoCount: result.photos.rejected,
          reviewId: result.reviewId,
        };
      } catch (err) {
        if (err instanceof ApiError) throw err; // a real rejection — nothing offline retry can fix
        // Network dropped mid-attempt — the review may or may not have been
        // created; queuing a fresh attempt is safe either way because the
        // Idempotency-Key makes a duplicate POST /reviews a no-op.
        await enqueueReviewSubmission({ targetId, form, idempotencyKey });
        return { queued: true, retryingPhotoCount: 0, rejectedPhotoCount: 0 };
      }
    },
    onSuccess: (result) => {
      if (!result.queued) invalidateReviewCaches(target.data!.id!, result.reviewId);
      // Submitted or queued: the draft has served its purpose. Its photos
      // now belong to the upload, so they're kept.
      if (!isEditing && userId && targetId) clearDraft(userId, targetId).catch(() => undefined);
    },
  });

  // Autosave, debounced. Paused while submitting and after success so a late
  // save can't resurrect a draft that was just cleared.
  useEffect(() => {
    if (isEditing || draftState !== 'ready' || !userId || !targetId || submit.isPending || submit.isSuccess) return;
    const timer = setTimeout(() => saveDraft(userId, targetId, form, step).catch(() => undefined), 500);
    return () => clearTimeout(timer);
  }, [isEditing, draftState, userId, targetId, form, step, submit.isPending, submit.isSuccess]);

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
          {t(isEditing ? 'review.loadReviewFailed' : 'review.loadTargetFailed')}
        </ThemedText>
      </ThemedView>
    );
  }

  if (draftState === 'checking') {
    return (
      <ThemedView style={styles.centered}>
        <ActivityIndicator />
      </ThemedView>
    );
  }

  if (draftState === 'prompt' && foundDraft) {
    return (
      <ThemedView style={styles.centered}>
        <ThemedText type="subtitle">{t('review.draftTitle')}</ThemedText>
        <ThemedText type="small" themeColor="textSecondary" style={styles.confirmationBody}>
          {t('review.draftSaved', { when: formatRelative(new Date(foundDraft.savedAt)) })}
        </ThemedText>
        <View style={styles.confirmationActions}>
          <Button title={t('review.draftResume')} onPress={resumeDraft} />
          <Button title={t('review.draftStartOver')} variant="secondary" onPress={startOver} />
        </View>
      </ThemedView>
    );
  }

  if (submit.isSuccess) {
    return (
      <ThemedView style={styles.centered}>
        <ThemedText type="subtitle">
          {t(submit.data.queued ? 'review.queuedTitle' : isEditing ? 'review.updatedTitle' : 'review.thanksTitle')}
        </ThemedText>
        <ThemedText type="small" themeColor="textSecondary" style={styles.confirmationBody}>
          {submit.data.queued ? t('review.queuedBody') : t('review.liveBody', { name: target.data.name })}
          {submit.data.retryingPhotoCount > 0
            ? ` ${t('review.photosRetrying', { count: submit.data.retryingPhotoCount })}`
            : ''}
          {submit.data.rejectedPhotoCount > 0
            ? ` ${t('review.photosRejected', { count: submit.data.rejectedPhotoCount })}`
            : ''}
        </ThemedText>
        {submit.data.reviewId ? (
          <ThemedText type="small" themeColor="textSecondary" style={styles.confirmationBody}>
            {t('evidence.addProofPrompt')}
          </ThemedText>
        ) : null}
        <View style={styles.confirmationActions}>
          {submit.data.reviewId && !submit.data.queued ? (
            // Telegram sharing is the main way people find places here; the
            // link opens the public review page (or the app, if installed).
            <Button
              title={t('review.shareYours')}
              onPress={() =>
                shareReview(
                  { id: submit.data.reviewId, title: form.title, body: form.body, overall_rating: form.overallRating },
                  target.data.name ?? ''
                ).catch(() => undefined)
              }
            />
          ) : null}
          {submit.data.reviewId ? (
            <Button
              title={t('evidence.addButton')}
              variant="secondary"
              onPress={() => router.replace(`/review/${submit.data.reviewId}/evidence`)}
            />
          ) : null}
          <Button title={t('common.backToTarget')} onPress={() =>
              // The form was opened from the place's screen; going back avoids
              // stacking a second copy of it (its data was just refreshed).
              router.canGoBack() ? router.back() : router.replace(`/target/${idOrSlug}`)
            } />
        </View>
      </ThemedView>
    );
  }

  const missingRequiredCriteria = (criteria.data ?? []).filter(
    (criterion) => criterion.required && criterion.code && !form.criterionScores[criterion.code]
  );
  const canProceed = canProceedFromStep(step, form, {
    pending: criteria.isPending,
    missingRequired: missingRequiredCriteria.length,
  });

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
      <Stack.Screen options={{ title: `${isEditing ? t('nav.editReview') : t('nav.writeReview')} · ${step + 1}/${REVIEW_STEP_COUNT}` }} />
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <View style={styles.dots}>
          {Array.from({ length: REVIEW_STEP_COUNT }).map((_, index) => (
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
              rating={form.overallRating}
              title={form.title}
              body={form.body}
              photos={form.photos}
              onChangeTitle={(title) => update('title', title)}
              onChangeBody={(body) => update('body', body)}
              onChangePhotos={(photos) => update('photos', photos)}
            />
          ) : (
            <View style={styles.detailsStep}>
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
              <DisclosureStep
                incentiveType={form.incentiveType}
                materialConnection={form.materialConnection}
                disclosureDetails={form.disclosureDetails}
                onChangeIncentiveType={(value) => update('incentiveType', value)}
                onChangeMaterialConnection={(value) => update('materialConnection', value)}
                onChangeDisclosureDetails={(value) => update('disclosureDetails', value)}
              />
            </View>
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
          {step < REVIEW_STEP_COUNT - 1 ? (
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
  detailsStep: {
    gap: Spacing.five,
  },
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
  confirmationActions: {
    gap: Spacing.two,
    alignSelf: 'stretch',
    paddingHorizontal: Spacing.four,
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
