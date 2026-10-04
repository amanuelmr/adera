/* eslint-disable @typescript-eslint/no-require-imports -- modules are re-required per test so the queue's module-level state starts fresh */
jest.mock('@react-native-async-storage/async-storage', () =>
  require('@react-native-async-storage/async-storage/jest/async-storage-mock')
);
jest.mock('@/api/env', () => ({ apiBaseUrl: 'http://api.test' }));
jest.mock('@/auth/storage', () => ({ getCurrentUserId: jest.fn(async () => 'user-1') }));
jest.mock('../photos', () => ({ deletePersistedPhoto: jest.fn() }));
let mockOnline = true;
jest.mock('@/lib/network-status', () => ({ isConnected: async () => mockOnline }));
jest.mock('../queries', () => ({
  createReview: jest.fn(),
  uploadPhotos: jest.fn(async () => ({ retryable: [], rejected: 0 })),
  invalidateReviewCaches: jest.fn(),
}));

const form = { photos: [] } as never;

let mockedCreateReview: jest.MockedFunction<typeof import('../queries')['createReview']>;
let apiError: (status: number, code?: string) => Error;
let AsyncStorage: typeof import('@react-native-async-storage/async-storage').default;

// The queue keeps module-level state, so every test loads a fresh copy — and
// takes ApiError and the mocks from that same registry, so instanceof holds.
function loadQueue(): typeof import('../offline-queue') {
  return require('../offline-queue');
}

beforeEach(async () => {
  mockOnline = true;
  jest.resetModules();
  const storage = require('@react-native-async-storage/async-storage');
  AsyncStorage = storage.default ?? storage;
  await AsyncStorage.clear();
  const { ApiError } = require('@/api/client') as typeof import('@/api/client');
  apiError = (status, code = 'internal_error') =>
    new ApiError(status, { error: { code, message: `failed with ${status}` } } as never);
  mockedCreateReview = require('../queries').createReview;
});

it.each([500, 503, 429, 408])('keeps a queued review for retry after a %i', async (status) => {
  const queue = loadQueue();
  await queue.enqueueReviewSubmission({ targetId: 't1', idempotencyKey: 'k1', form });
  mockedCreateReview.mockRejectedValueOnce(apiError(status));

  await queue.processQueue();

  const [job] = queue.getQueueSnapshot();
  expect(job).toBeDefined();
  expect(job.failure).toBeUndefined();
});

it('keeps a queued review for retry after a network failure', async () => {
  const queue = loadQueue();
  await queue.enqueueReviewSubmission({ targetId: 't1', idempotencyKey: 'k1', form });
  mockedCreateReview.mockRejectedValueOnce(new TypeError('Network request failed'));

  await queue.processQueue();

  expect(queue.getQueueSnapshot()).toHaveLength(1);
});

it('keeps a rejected review visible as failed instead of dropping it, and never retries it', async () => {
  const queue = loadQueue();
  await queue.enqueueReviewSubmission({ targetId: 't1', idempotencyKey: 'k1', form });
  mockedCreateReview.mockRejectedValueOnce(apiError(409, 'cooldown_active'));

  await queue.processQueue();
  await queue.processQueue();

  const [job] = queue.getQueueSnapshot();
  expect(job.failure).toEqual({ message: 'failed with 409', code: 'cooldown_active' });
  expect(mockedCreateReview).toHaveBeenCalledTimes(1);

  await queue.dismissJob(job.id);
  expect(queue.getQueueSnapshot()).toHaveLength(0);
});

it('stops the run on a 401 without touching later jobs', async () => {
  const queue = loadQueue();
  await queue.enqueueReviewSubmission({ targetId: 't1', idempotencyKey: 'k1', form });
  await queue.enqueueReviewSubmission({ targetId: 't2', idempotencyKey: 'k2', form });
  mockedCreateReview.mockRejectedValueOnce(apiError(401, 'unauthorized'));

  await queue.processQueue();

  expect(mockedCreateReview).toHaveBeenCalledTimes(1);
  expect(queue.getQueueSnapshot()).toHaveLength(2);
});

it('removes a job once it succeeds', async () => {
  const queue = loadQueue();
  await queue.enqueueReviewSubmission({ targetId: 't1', idempotencyKey: 'k1', form });
  mockedCreateReview.mockResolvedValueOnce('review-1');

  await queue.processQueue();

  expect(queue.getQueueSnapshot()).toHaveLength(0);
  expect(require('../queries').invalidateReviewCaches).toHaveBeenCalledWith('t1', 'review-1');
});

it('does not lose a job enqueued while the queue is still loading from storage', async () => {
  await AsyncStorage.setItem(
    'adera.offline-queue.v1',
    JSON.stringify([{ type: 'helpful-vote', id: 'old', userId: 'user-1', reviewId: 'r0', voted: true }])
  );
  const queue = loadQueue();

  await Promise.all([
    queue.enqueueHelpfulVote('r1', true),
    queue.enqueueReviewSubmission({ targetId: 't1', idempotencyKey: 'k1', form }),
  ]);

  expect(queue.getQueueSnapshot().map((job) => job.type).sort()).toEqual([
    'helpful-vote',
    'helpful-vote',
    'review-submission',
  ]);
});

it('gives up after repeated online failures and can be retried by hand', async () => {
  const queue = loadQueue();
  await queue.enqueueReviewSubmission({ targetId: 't1', idempotencyKey: 'k1', form });
  mockedCreateReview.mockRejectedValue(apiError(503));

  for (let i = 0; i < queue.MAX_ATTEMPTS; i++) await queue.processQueue();

  const [job] = queue.getQueueSnapshot();
  expect(job.failure?.code).toBe(queue.GAVE_UP);
  expect(mockedCreateReview).toHaveBeenCalledTimes(queue.MAX_ATTEMPTS);
  await queue.processQueue();
  expect(mockedCreateReview).toHaveBeenCalledTimes(queue.MAX_ATTEMPTS); // no longer retried on its own

  mockedCreateReview.mockResolvedValueOnce('review-1');
  await queue.retryJob(job.id);
  expect(queue.getQueueSnapshot()).toHaveLength(0);
});

it("doesn't run, or spend attempts, while offline", async () => {
  const queue = loadQueue();
  await queue.enqueueReviewSubmission({ targetId: 't1', idempotencyKey: 'k1', form });
  mockOnline = false;

  for (let i = 0; i < queue.MAX_ATTEMPTS + 5; i++) await queue.processQueue();

  expect(mockedCreateReview).not.toHaveBeenCalled();
  expect(queue.getQueueSnapshot()[0].failure).toBeUndefined();
});
