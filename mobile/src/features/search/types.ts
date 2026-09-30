import type { components } from '@/api/schema';

// categoryName is intentionally not stored here — it's always derivable from
// `category` (an id) plus the categories list, so keeping it as separate
// state would just be a second source of truth to fall out of sync.
export type SearchFilters = {
  category?: string;
  type?: components['schemas']['TargetType'];
  minRating?: number;
  verified?: boolean;
};

// Labels live in the locale files under search.type.*.
export const TARGET_TYPES: components['schemas']['TargetType'][] = [
  'business',
  'location',
  'online_seller',
  'product',
  'service',
  'repair_provider',
  'restaurant',
  'cafe',
];

export const MIN_RATING_OPTIONS = [3, 4, 4.5] as const;
