import { Share } from 'react-native';

import i18n from '@/lib/i18n';

// The public web pages (internal/pages): anyone can open these, app or not,
// and Android App Links (app.json intentFilters) open them in the app when
// it's installed.
export const WEB_BASE_URL = (process.env.EXPO_PUBLIC_WEB_BASE_URL ?? 'https://adera.amanuel.work').replace(/\/$/, '');

// Below this many reviews the app and web pages hide the aggregate
// (confidence "none", internal/ratings); a share mustn't show it either.
const MIN_REVIEWS_FOR_RATING = 3;
const QUOTE_LENGTH = 140;

export function buildTargetLink(slugOrId: string): string {
  return `${WEB_BASE_URL}/t/${encodeURIComponent(slugOrId)}`;
}

export function buildReviewLink(reviewId: string): string {
  return `${WEB_BASE_URL}/r/${encodeURIComponent(reviewId)}`;
}

function quoteOf(text: string): string {
  const trimmed = text.trim();
  return trimmed.length > QUOTE_LENGTH ? `${trimmed.slice(0, QUOTE_LENGTH).trimEnd()}…` : trimmed;
}

export function buildTargetShareText(
  target: { name?: string; average_rating?: number | null; review_count?: number },
  link: string
): string {
  const name = target.name ?? i18n.t('share.thisPlace');
  const showRating = target.average_rating != null && (target.review_count ?? 0) >= MIN_REVIEWS_FOR_RATING;
  const text = showRating
    ? i18n.t('share.targetRated', { name, rating: target.average_rating!.toFixed(1) })
    : i18n.t('share.target', { name });
  return `${text}\n${link}`;
}

export function buildReviewShareText(
  review: { title?: string; body?: string; overall_rating?: number },
  targetName: string,
  link: string
): string {
  const quote = quoteOf(review.title || review.body || '');
  const text =
    review.overall_rating != null
      ? i18n.t('share.reviewRated', { quote, rating: review.overall_rating, name: targetName })
      : i18n.t('share.review', { quote, name: targetName });
  return `${text}\n${link}`;
}

export async function shareTarget(target: {
  id?: string;
  slug?: string;
  name?: string;
  average_rating?: number | null;
  review_count?: number;
}): Promise<void> {
  const key = target.slug ?? target.id;
  if (!key) return;
  await Share.share({ message: buildTargetShareText(target, buildTargetLink(key)) });
}

export async function shareReview(
  review: { id?: string; title?: string; body?: string; overall_rating?: number },
  targetName: string
): Promise<void> {
  if (!review.id) return;
  await Share.share({ message: buildReviewShareText(review, targetName, buildReviewLink(review.id)) });
}
