/* eslint-disable @typescript-eslint/no-require-imports, import/first -- mocks must be declared before the module under test loads */
jest.mock('@react-native-async-storage/async-storage', () =>
  require('@react-native-async-storage/async-storage/jest/async-storage-mock')
);
const mockClearQueue = jest.fn(async () => undefined);
jest.mock('@/features/reviews/offline-queue', () => ({ clearQueue: () => mockClearQueue() }));
jest.mock('@/features/reviews/photos', () => ({ PERSISTED_PHOTO_PREFIX: 'review-photo-', deletePersistedPhoto: jest.fn() }));
const mockFiles = [
  { name: 'review-photo-1.jpg', delete: jest.fn() },
  { name: 'review-photo-2.jpg', delete: jest.fn() },
  { name: 'something-else.db', delete: jest.fn() },
];
jest.mock('expo-file-system', () => ({
  Paths: { document: 'file:///doc/' },
  Directory: class {
    list() {
      return mockFiles;
    }
  },
}));

import AsyncStorage from '@react-native-async-storage/async-storage';

import { wipeLocalAccountData } from '../local-data';

it('removes drafts, the offline queue and saved review photos, and nothing else', async () => {
  await AsyncStorage.multiSet([
    ['adera.review-draft.v1:user-1:t1', '{}'],
    ['adera.review-draft.v1:user-2:t9', '{}'],
    ['adera.settings', '{"dataSaver":true}'],
    ['adera.language', 'am'],
  ]);

  await wipeLocalAccountData();

  expect(mockClearQueue).toHaveBeenCalled();
  expect(await AsyncStorage.getAllKeys()).toEqual(['adera.settings', 'adera.language']);
  expect(mockFiles[0].delete).toHaveBeenCalled();
  expect(mockFiles[1].delete).toHaveBeenCalled();
  expect(mockFiles[2].delete).not.toHaveBeenCalled();
});

it('keeps going when one part fails', async () => {
  mockClearQueue.mockRejectedValueOnce(new Error('storage full'));
  await AsyncStorage.setItem('adera.review-draft.v1:user-1:t1', '{}');
  await expect(wipeLocalAccountData()).resolves.toBeUndefined();
  expect(await AsyncStorage.getItem('adera.review-draft.v1:user-1:t1')).toBeNull();
});
