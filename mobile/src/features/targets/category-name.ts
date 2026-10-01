import type { components } from '@/api/schema';
import i18n from '@/lib/i18n';

/** The category's name in the app language — the API sends per-language names alongside the default. */
export function categoryName(category: components['schemas']['Category'] | undefined): string | undefined {
  if (!category) return undefined;
  return category.name_translations?.[i18n.language] || category.name;
}
