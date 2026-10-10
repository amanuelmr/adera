import type { ReviewFormState } from './types';
import { experienceDateError, parsePrice } from './validation';

// Rating → criteria → text & photos → details (context + disclosure).
// Context and disclosure share the last step: disclosure stays on screen
// (it is trust-critical), but neither needs a page of its own.
export const REVIEW_STEP_COUNT = 4;
export const MIN_BODY_LENGTH = 20;

export function canProceedFromStep(
  step: number,
  form: ReviewFormState,
  criteria: { pending: boolean; missingRequired: number }
): boolean {
  switch (step) {
    case 0:
      return form.overallRating != null;
    case 1:
      return !criteria.pending && criteria.missingRequired === 0;
    case 2:
      return form.body.trim().length >= MIN_BODY_LENGTH;
    case 3: {
      const detailsRequired = form.incentiveType === 'other' || form.materialConnection === 'other';
      return (
        !experienceDateError(form.experienceDate) &&
        !parsePrice(form.pricePaid).error &&
        (!detailsRequired || form.disclosureDetails.trim().length > 0)
      );
    }
    default:
      return false;
  }
}

/** A local calendar date as YYYY-MM-DD (the format experience_date takes). */
export function toIsoDate(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

export function daysAgo(days: number, today: Date = new Date()): string {
  return toIsoDate(new Date(today.getFullYear(), today.getMonth(), today.getDate() - days));
}

export type DateChoice = 'today' | 'yesterday' | 'pick';

/** Which date chip a stored experience_date corresponds to, if any. */
export function dateChoiceFor(value: string, today: Date = new Date()): DateChoice | undefined {
  const trimmed = value.trim();
  if (!trimmed) return undefined;
  if (trimmed === daysAgo(0, today)) return 'today';
  if (trimmed === daysAgo(1, today)) return 'yesterday';
  return 'pick';
}

export type WritingPrompt = 'low' | 'mid' | 'high';

/** Prompts under the text box nudge toward specifics that fit the rating. */
export function writingPromptFor(rating: number | undefined): WritingPrompt | undefined {
  if (rating == null) return undefined;
  if (rating <= 2) return 'low';
  if (rating === 3) return 'mid';
  return 'high';
}
