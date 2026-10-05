/* eslint-disable @typescript-eslint/no-require-imports -- each test loads fresh modules against a fresh i18n state */
import am from '../locales/am.json';
import en from '../locales/en.json';

jest.mock('@/api/env', () => ({ apiBaseUrl: 'http://api.test' }));
jest.mock('@react-native-async-storage/async-storage', () =>
  require('@react-native-async-storage/async-storage/jest/async-storage-mock')
);
jest.mock('expo-localization', () => ({ getLocales: () => [{ languageCode: 'am' }] }));

beforeEach(() => jest.resetModules());

it('describes push events in English before i18n has initialized (headless background task)', () => {
  const { describeEventType } = require('@/features/activity/event-copy');
  expect(describeEventType('response.created')).toBe(en.events.response);
  expect(describeEventType('something.new')).toBe(en.events.default);
});

it('describes events in the app language once i18n is ready', async () => {
  const { initI18n } = require('../i18n');
  await initI18n();
  const { describeEventType } = require('@/features/activity/event-copy');
  expect(describeEventType('claim.approved')).toBe(am.events.claim);
});

it('never shows English server text to an Amharic user', async () => {
  const { initI18n } = require('../i18n');
  await initI18n();
  const { ApiError } = require('@/api/client');
  const { friendlyAuthError } = require('@/auth/friendly-error');
  const validation = new ApiError(422, { error: { code: 'validation_failed', message: 'x', details: { email: 'must be valid' } } });
  const unknown = new ApiError(500, { error: { code: 'internal_error', message: 'database exploded' } });
  expect(friendlyAuthError(validation)).toBe(am.errors.validation);
  expect(friendlyAuthError(unknown)).toBe(am.errors.generic);
});

it('labels every moderation status and review sort order in both languages', () => {
  const statuses = ['pending', 'published', 'under_review', 'rejected', 'hidden', 'removed'];
  const sorts = ['newest', 'highest', 'lowest', 'most_helpful'];
  for (const dict of [en, am]) {
    for (const s of statuses) expect(dict.myReviews.status).toHaveProperty(s);
    for (const s of sorts) expect(dict.target.sort).toHaveProperty(s);
  }
});

it('shows category names in the app language, falling back to the default name', async () => {
  const { initI18n } = require('../i18n');
  await initI18n();
  const { categoryName } = require('@/features/targets/category-name');
  const category = { name: 'Restaurants & Cafés', name_translations: { am: 'ምግብ ቤቶች እና ካፌዎች', en: 'Restaurants & Cafés' } };
  expect(categoryName(category)).toBe('ምግብ ቤቶች እና ካፌዎች');
  expect(categoryName({ name: 'Untranslated' })).toBe('Untranslated');
});

it('labels every target type offered in search filters', () => {
  const { TARGET_TYPES } = require('@/features/search/types');
  for (const dict of [en, am]) for (const type of TARGET_TYPES) expect(dict.search.type).toHaveProperty(type);
});

it('uses event-specific wording where the prefix would mislead', () => {
  const { describeEventType } = require('@/features/activity/event-copy');
  expect(describeEventType('target.approve')).toBe(en.events.targetApproved);
  expect(describeEventType('target.hide')).toBe(en.events.target);
});

it('tells owners about a new review with its rating, even before i18n loads', () => {
  const { describeEventType } = require('@/features/activity/event-copy');
  expect(describeEventType('review.received', { rating: '2' })).toBe(
    en.events.reviewReceived.replace('{{rating}}', '2')
  );
});
