import type { IconName } from '@/components/icon';

// An icon for each category, by its stable code (migrations/0010 and
// admin-created ones). Categories the app doesn't know yet get a generic
// shop icon, so a new category never breaks a screen.
const CATEGORY_ICONS: Record<string, IconName> = {
  restaurant_cafe: 'utensils',
  beauty_salon: 'scissors',
  electronics_repair: 'wrench',
  online_seller: 'shopping-bag',
};

export function categoryIcon(code: string | undefined): IconName {
  return (code && CATEGORY_ICONS[code]) || 'store';
}
