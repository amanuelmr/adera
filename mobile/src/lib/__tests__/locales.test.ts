import am from '../locales/am.json';
import en from '../locales/en.json';
import { INCENTIVE_TYPE_LABELS, MATERIAL_CONNECTION_LABELS } from '@/features/reviews/types';

function keyPaths(value: unknown, prefix = ''): string[] {
  if (typeof value !== 'object' || value === null) return [prefix];
  return Object.entries(value).flatMap(([key, child]) => keyPaths(child, prefix ? `${prefix}.${key}` : key));
}

it('has the same keys in English and Amharic', () => {
  expect(keyPaths(am).sort()).toEqual(keyPaths(en).sort());
});

// Disclosure labels are trust-critical: a value the backend can return must
// never render as a raw key.
it.each([
  ['incentive', INCENTIVE_TYPE_LABELS],
  ['connection', MATERIAL_CONNECTION_LABELS],
] as const)('labels every non-none %s disclosure value', (group, values) => {
  for (const value of Object.keys(values).filter((v) => v !== 'none')) {
    expect(en.disclosure[group]).toHaveProperty(value);
    expect(am.disclosure[group]).toHaveProperty(value);
  }
});
