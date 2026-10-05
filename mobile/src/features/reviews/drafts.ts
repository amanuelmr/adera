import AsyncStorage from '@react-native-async-storage/async-storage';

import { deletePersistedPhoto } from './photos';
import type { ReviewFormState } from './types';

// An unfinished review, kept so it survives the app being closed or killed —
// on low-memory Android, opening the camera or gallery is exactly when the
// OS tends to kill the app. One draft per account and place.
export type ReviewDraft = { form: ReviewFormState; step: number; savedAt: number };

// Old drafts are dropped rather than resurrected weeks later.
export const DRAFT_MAX_AGE_MS = 14 * 24 * 60 * 60 * 1000;

function key(userId: string, targetId: string): string {
  return `adera.review-draft.v1:${userId}:${targetId}`;
}

/** Worth keeping: anything the user actually entered. */
export function hasContent(form: ReviewFormState): boolean {
  return (
    form.overallRating != null ||
    Object.keys(form.criterionScores).length > 0 ||
    form.title.trim() !== '' ||
    form.body.trim() !== '' ||
    form.photos.length > 0 ||
    form.experienceDate !== '' ||
    form.discoverySource != null ||
    form.pricePaid !== '' ||
    form.incentiveType !== 'none' ||
    form.materialConnection !== 'none' ||
    form.disclosureDetails !== ''
  );
}

export async function saveDraft(
  userId: string,
  targetId: string,
  form: ReviewFormState,
  step: number,
  now: number = Date.now()
): Promise<void> {
  if (!hasContent(form)) {
    await AsyncStorage.removeItem(key(userId, targetId));
    return;
  }
  const draft: ReviewDraft = { form, step, savedAt: now };
  await AsyncStorage.setItem(key(userId, targetId), JSON.stringify(draft));
}

export async function loadDraft(
  userId: string,
  targetId: string,
  now: number = Date.now()
): Promise<ReviewDraft | undefined> {
  try {
    const raw = await AsyncStorage.getItem(key(userId, targetId));
    if (!raw) return undefined;
    const draft = JSON.parse(raw) as ReviewDraft;
    if (now - draft.savedAt > DRAFT_MAX_AGE_MS) {
      await discardDraft(userId, targetId, draft);
      return undefined;
    }
    return draft;
  } catch {
    return undefined; // unreadable: start fresh rather than block the form
  }
}

/** The review was submitted or queued: its photos now belong to the upload. */
export async function clearDraft(userId: string, targetId: string): Promise<void> {
  await AsyncStorage.removeItem(key(userId, targetId));
}

/** The user chose to start over: the draft's photos go too. */
export async function discardDraft(userId: string, targetId: string, draft?: ReviewDraft): Promise<void> {
  draft?.form.photos.forEach((photo) => deletePersistedPhoto(photo.uri));
  await AsyncStorage.removeItem(key(userId, targetId));
}
