import type { components } from '@/api/schema';

export type PickedPhoto = {
  /** Local file URI after client-side compression, before upload. */
  uri: string;
};

// would_recommend and return_likelihood exist on the API but aren't part of
// this screen set (docs/mobile-plan.md §9 Step 4 doesn't call for them) —
// left uncollected rather than sent as guessed defaults.
export type ReviewFormState = {
  overallRating?: number;
  criterionScores: Record<string, number>;
  title: string;
  body: string;
  photos: PickedPhoto[];
  experienceDate: string;
  discoverySource?: components['schemas']['DiscoverySource'];
  expectationMatch?: components['schemas']['ExpectationMatch'];
  pricePaid: string;
  incentiveType: components['schemas']['IncentiveType'];
  materialConnection: components['schemas']['MaterialConnection'];
  disclosureDetails: string;
};

export const INITIAL_REVIEW_FORM: ReviewFormState = {
  criterionScores: {},
  title: '',
  body: '',
  photos: [],
  experienceDate: '',
  pricePaid: '',
  incentiveType: 'none',
  materialConnection: 'none',
  disclosureDetails: '',
};

// Mirrors internal/reviews/models.go's SocialSources — expectation_match is
// only meaningful (and only accepted by the API) for these sources.
export const SOCIAL_DISCOVERY_SOURCES = new Set<components['schemas']['DiscoverySource']>([
  'tiktok',
  'instagram',
  'youtube',
  'facebook',
  'telegram',
]);

export const DISCOVERY_SOURCE_LABELS: Record<components['schemas']['DiscoverySource'], string> = {
  tiktok: 'TikTok',
  instagram: 'Instagram',
  youtube: 'YouTube',
  facebook: 'Facebook',
  telegram: 'Telegram',
  friend: 'A friend',
  google_maps: 'Google Maps',
  walk_in: 'Walked in',
  other: 'Other',
};

export const EXPECTATION_MATCH_LABELS: Record<components['schemas']['ExpectationMatch'], string> = {
  better: 'Better than expected',
  as_expected: 'As expected',
  worse: 'Worse than expected',
  very_different: 'Very different',
};

export const INCENTIVE_TYPE_LABELS: Record<components['schemas']['IncentiveType'], string> = {
  none: 'None',
  discount: 'Discount',
  free_product_or_service: 'Free product/service',
  payment: 'Payment',
  contest_entry: 'Contest entry',
  loyalty_points: 'Loyalty points',
  other: 'Other',
};

export const MATERIAL_CONNECTION_LABELS: Record<components['schemas']['MaterialConnection'], string> = {
  none: 'None',
  current_employee: 'Current employee',
  former_employee: 'Former employee',
  owner_or_executive: 'Owner/executive',
  family_or_friend: 'Family or friend of owner',
  business_partner: 'Business partner',
  other: 'Other',
};

export const RATING_LABELS: Record<number, string> = {
  1: 'Poor',
  2: 'Fair',
  3: 'Good',
  4: 'Very good',
  5: 'Excellent',
};
