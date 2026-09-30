import { Share } from 'react-native';

import i18n from '@/lib/i18n';

/**
 * The only real acquisition surface (a public https:// permalink) is Phase 3
 * web-layer work — this deep link is useful only between two people who
 * already have the app installed. Swapping in a real URL later only means
 * changing this one function, not any call site.
 */
export function buildShareLink(targetId: string): string {
  return `adera://target/${targetId}`;
}

export function buildTargetShareText(target: { name?: string; average_rating?: number | null }, link: string): string {
  const name = target.name ?? i18n.t('share.thisPlace');
  const text =
    target.average_rating != null
      ? i18n.t('share.targetRated', { name, rating: target.average_rating.toFixed(1) })
      : i18n.t('share.target', { name });
  return `${text}\n${link}`;
}

export function buildReviewShareText(
  review: { title?: string; body?: string; overall_rating?: number },
  targetName: string,
  link: string
): string {
  const quote = (review.title || review.body || '').slice(0, 140);
  const text =
    review.overall_rating != null
      ? i18n.t('share.reviewRated', { quote, rating: review.overall_rating, name: targetName })
      : i18n.t('share.review', { quote, name: targetName });
  return `${text}\n${link}`;
}

export async function shareTarget(target: { id?: string; name?: string; average_rating?: number | null }): Promise<void> {
  if (!target.id) return;
  const link = buildShareLink(target.id);
  await Share.share({ message: buildTargetShareText(target, link) });
}

export async function shareReview(
  review: { target_id?: string; title?: string; body?: string; overall_rating?: number },
  targetName: string
): Promise<void> {
  if (!review.target_id) return;
  const link = buildShareLink(review.target_id);
  await Share.share({ message: buildReviewShareText(review, targetName, link) });
}
