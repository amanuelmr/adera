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

export const TARGET_TYPE_LABELS: Record<components['schemas']['TargetType'], string> = {
  business: 'Business',
  location: 'Location',
  online_seller: 'Online seller',
  product: 'Product',
  service: 'Service',
  repair_provider: 'Repair provider',
  restaurant: 'Restaurant',
  cafe: 'Café',
};

export const MIN_RATING_OPTIONS = [3, 4, 4.5] as const;
