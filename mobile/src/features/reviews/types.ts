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

// Display labels live in the locale files (review.discovery.*, etc.), keyed
// by these values — see the step components.
export const DISCOVERY_SOURCES: components['schemas']['DiscoverySource'][] = [
  'tiktok',
  'instagram',
  'youtube',
  'facebook',
  'telegram',
  'friend',
  'google_maps',
  'walk_in',
  'other',
];

export const EXPECTATION_MATCHES: components['schemas']['ExpectationMatch'][] = [
  'better',
  'as_expected',
  'worse',
  'very_different',
];

export const INCENTIVE_TYPES: components['schemas']['IncentiveType'][] = [
  'none',
  'discount',
  'free_product_or_service',
  'payment',
  'contest_entry',
  'loyalty_points',
  'other',
];

export const MATERIAL_CONNECTIONS: components['schemas']['MaterialConnection'][] = [
  'none',
  'current_employee',
  'former_employee',
  'owner_or_executive',
  'family_or_friend',
  'business_partner',
  'other',
];
