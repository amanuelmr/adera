/* eslint-disable @typescript-eslint/no-require-imports -- fresh module state per test */
import { INITIAL_REVIEW_FORM } from '../types';

const mockDisk = new Map<string, string>();
jest.mock('@react-native-async-storage/async-storage', () => ({
  getItem: async (key: string) => mockDisk.get(key) ?? null,
  setItem: async (key: string, value: string) => void mockDisk.set(key, value),
  removeItem: async (key: string) => void mockDisk.delete(key),
}));
const mockDeleted: string[] = [];
jest.mock('../photos', () => ({ deletePersistedPhoto: (uri: string) => mockDeleted.push(uri) }));

const drafts = require('../drafts') as typeof import('../drafts');

const filled = { ...INITIAL_REVIEW_FORM, overallRating: 4, body: 'Great macchiato', photos: [{ uri: 'file:///p1.jpg' }] };

beforeEach(() => {
  mockDisk.clear();
  mockDeleted.length = 0;
});

it('saves and restores a draft per account and place', async () => {
  await drafts.saveDraft('user-1', 'place-1', filled, 2, 1000);
  expect(await drafts.loadDraft('user-1', 'place-1', 2000)).toEqual({ form: filled, step: 2, savedAt: 1000 });
  expect(await drafts.loadDraft('user-2', 'place-1', 2000)).toBeUndefined();
  expect(await drafts.loadDraft('user-1', 'place-2', 2000)).toBeUndefined();
});

it('does not keep an empty form', async () => {
  await drafts.saveDraft('user-1', 'place-1', filled, 2);
  await drafts.saveDraft('user-1', 'place-1', INITIAL_REVIEW_FORM, 0);
  expect(await drafts.loadDraft('user-1', 'place-1')).toBeUndefined();
});

it('clearing after submit keeps the photos (the upload owns them now)', async () => {
  await drafts.saveDraft('user-1', 'place-1', filled, 2);
  await drafts.clearDraft('user-1', 'place-1');
  expect(await drafts.loadDraft('user-1', 'place-1')).toBeUndefined();
  expect(mockDeleted).toEqual([]);
});

it('starting over deletes the draft and its photos', async () => {
  await drafts.saveDraft('user-1', 'place-1', filled, 2);
  const draft = await drafts.loadDraft('user-1', 'place-1');
  await drafts.discardDraft('user-1', 'place-1', draft);
  expect(await drafts.loadDraft('user-1', 'place-1')).toBeUndefined();
  expect(mockDeleted).toEqual(['file:///p1.jpg']);
});

it('drops drafts older than two weeks, with their photos', async () => {
  await drafts.saveDraft('user-1', 'place-1', filled, 2, 0);
  expect(await drafts.loadDraft('user-1', 'place-1', drafts.DRAFT_MAX_AGE_MS + 1)).toBeUndefined();
  expect(mockDeleted).toEqual(['file:///p1.jpg']);
});

it('ignores an unreadable draft instead of blocking the form', async () => {
  mockDisk.set('adera.review-draft.v1:user-1:place-1', '{broken');
  expect(await drafts.loadDraft('user-1', 'place-1')).toBeUndefined();
});
