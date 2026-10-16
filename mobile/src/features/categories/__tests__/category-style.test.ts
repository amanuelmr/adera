import { categoryIcon } from '../category-style';

it('gives each seeded category its own icon', () => {
  const icons = ['restaurant_cafe', 'beauty_salon', 'electronics_repair', 'online_seller'].map(categoryIcon);
  expect(new Set(icons).size).toBe(4);
  expect(icons).not.toContain('store');
});

it('falls back to a shop icon for categories it does not know', () => {
  expect(categoryIcon('pharmacy')).toBe('store');
  expect(categoryIcon(undefined)).toBe('store');
});
