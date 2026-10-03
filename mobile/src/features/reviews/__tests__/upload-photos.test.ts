/* eslint-disable @typescript-eslint/no-require-imports -- ApiError must come from the same module registry as the code under test */
jest.mock('@/api/env', () => ({ apiBaseUrl: 'http://api.test' }));
jest.mock('@/lib/query-client', () => ({ queryClient: { invalidateQueries: jest.fn() } }));
const mockMissing = new Set<string>();
jest.mock('expo-file-system', () => ({
  File: class {
    mockUri: string;
    constructor(mockUri: string) {
      this.mockUri = mockUri;
    }
    get exists() {
      return !mockMissing.has(this.mockUri);
    }
  },
}));
const mockDeleted: string[] = [];
jest.mock('../photos', () => ({ deletePersistedPhoto: (uri: string) => mockDeleted.push(uri) }));

const { ApiError, apiClient } = require('@/api/client') as typeof import('@/api/client');
const { uploadPhotos } = require('../queries') as typeof import('../queries');

const presign = jest.spyOn(apiClient, 'POST');

beforeEach(() => {
  presign.mockReset();
  mockDeleted.length = 0;
  mockMissing.clear();
});

it('sorts failures: refused photos are deleted and counted, network failures kept for retry', async () => {
  presign
    .mockRejectedValueOnce(new ApiError(409, { error: { code: 'conflict', message: 'at most 5 photos per review' } } as never))
    .mockRejectedValueOnce(new TypeError('Network request failed'));

  const result = await uploadPhotos('review-1', ['file:///refused.jpg', 'file:///offline.jpg']);

  expect(result).toEqual({ retryable: ['file:///offline.jpg'], rejected: 1 });
  expect(mockDeleted).toEqual(['file:///refused.jpg']);
});

it('skips photo files that no longer exist instead of retrying them forever', async () => {
  mockMissing.add('file:///gone.jpg');
  const result = await uploadPhotos('review-1', ['file:///gone.jpg']);
  expect(result).toEqual({ retryable: [], rejected: 0 });
  expect(presign).not.toHaveBeenCalled();
});
