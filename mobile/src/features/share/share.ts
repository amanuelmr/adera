import { Share } from 'react-native';

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
  const rating = target.average_rating != null ? `${target.average_rating.toFixed(1)}★ on Adera` : 'on Adera';
  return `${target.name ?? 'This place'} — ${rating}\n${link}`;
}

export function buildReviewShareText(
  review: { title?: string; body?: string; overall_rating?: number },
  targetName: string,
  link: string
): string {
  const quote = (review.title || review.body || '').slice(0, 140);
  const rating = review.overall_rating != null ? `${review.overall_rating}★ ` : '';
  return `"${quote}" — ${rating}review of ${targetName} on Adera\n${link}`;
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
