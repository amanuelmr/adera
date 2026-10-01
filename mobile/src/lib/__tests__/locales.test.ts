/// <reference types="node" />
import fs from 'fs';
import path from 'path';

import am from '../locales/am.json';
import en from '../locales/en.json';
import { INCENTIVE_TYPES, MATERIAL_CONNECTIONS } from '@/features/reviews/types';

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
  ['incentive', INCENTIVE_TYPES],
  ['connection', MATERIAL_CONNECTIONS],
] as const)('labels every %s disclosure value, in the form and on public cards', (group, values) => {
  for (const value of values) {
    expect(en.disclosure[`${group}Option`]).toHaveProperty(value);
    expect(am.disclosure[`${group}Option`]).toHaveProperty(value);
    if (value === 'none') continue;
    expect(en.disclosure[group]).toHaveProperty(value);
    expect(am.disclosure[group]).toHaveProperty(value);
  }
});

function sourceFiles(dir: string): string[] {
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return entry.name === '__tests__' ? [] : sourceFiles(full);
    return /\.tsx?$/.test(entry.name) && !entry.name.endsWith('.d.ts') ? [full] : [];
  });
}

function lookup(dict: unknown, key: string): unknown {
  return key.split('.').reduce<unknown>((node, part) => (node as Record<string, unknown> | undefined)?.[part], dict);
}

// A missing key renders as the raw key path on screen. Only literal keys are
// checked; dynamic ones (template strings) are covered by their own tests.
it('defines every literal t() key used in the app, in both languages', () => {
  const used = new Set<string>();
  for (const file of sourceFiles(path.join(__dirname, '..', '..'))) {
    for (const match of fs.readFileSync(file, 'utf8').matchAll(/\bt\(\s*['"]([\w.]+)['"]/g)) used.add(match[1]);
  }
  const missing = [...used].filter((key) =>
    [en, am].some(
      (dict) =>
        typeof lookup(dict, key) !== 'string' &&
        typeof lookup(dict, `${key}_one`) !== 'string' &&
        typeof lookup(dict, `${key}_other`) !== 'string'
    )
  );
  expect(used.size).toBeGreaterThan(50);
  expect(missing).toEqual([]);
});

it('labels every review-form option the API accepts', () => {
  const { DISCOVERY_SOURCES, EXPECTATION_MATCHES } = jest.requireActual('@/features/reviews/types');
  for (const dict of [en, am]) {
    for (const value of DISCOVERY_SOURCES) expect(dict.review.discovery).toHaveProperty(value);
    for (const value of EXPECTATION_MATCHES) expect(dict.review.expectation).toHaveProperty(value);
    for (const star of ['1', '2', '3', '4', '5']) expect(dict.review.rating).toHaveProperty(star);
  }
});

// An Amharic value identical to its English is almost always a string that
// was copied over and never translated. Brand names are the exception.
const SAME_IN_BOTH = new Set(['tiktok', 'instagram', 'youtube', 'facebook', 'telegram', 'google_maps'].map((k) => `review.discovery.${k}`));

it('translates every Amharic string rather than copying the English', () => {
  const untranslated = keyPaths(en).filter(
    (key) => !SAME_IN_BOTH.has(key) && lookup(am, key) === lookup(en, key)
  );
  expect(untranslated).toEqual([]);
});
