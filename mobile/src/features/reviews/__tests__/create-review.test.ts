/* eslint-disable @typescript-eslint/no-require-imports -- the API client must come from the same module registry as the code under test */
jest.mock('@/api/env', () => ({ apiBaseUrl: 'http://api.test' }));
jest.mock('@/lib/query-client', () => ({ queryClient: { invalidateQueries: jest.fn() } }));
jest.mock('expo-file-system', () => ({ File: class {} }));
jest.mock('../photos', () => ({ deletePersistedPhoto: jest.fn() }));
let mockInstallId: string | undefined;
jest.mock('@/lib/install-id', () => ({ getInstallId: async () => mockInstallId }));

const { apiClient } = require('@/api/client') as typeof import('@/api/client');
const { createReview } = require('../queries') as typeof import('../queries');
const { INITIAL_REVIEW_FORM } = require('../types') as typeof import('../types');

const post = jest.spyOn(apiClient, 'POST');
const form = { ...INITIAL_REVIEW_FORM, overallRating: 4, body: 'Long enough to tell people something.' };

function requestHeaders() {
  return (post.mock.calls[0] as unknown as [string, { headers?: Record<string, string> }])[1].headers;
}

beforeEach(() => {
  post.mockReset();
  post.mockResolvedValue({ data: { data: { id: 'review-1' } }, response: new Response() } as never);
});

it('sends the install ID with a new review', async () => {
  mockInstallId = '6f1c2b9e-8d43-4a57-9b0e-2f6c1d3a4b5c';
  await expect(createReview('target-1', form, 'key-1')).resolves.toBe('review-1');
  expect(requestHeaders()).toEqual({
    'Idempotency-Key': 'key-1',
    'X-Install-ID': '6f1c2b9e-8d43-4a57-9b0e-2f6c1d3a4b5c',
  });
});

it('still submits when there is no install ID', async () => {
  mockInstallId = undefined;
  await createReview('target-1', form, 'key-1');
  expect(requestHeaders()).toEqual({ 'Idempotency-Key': 'key-1' });
});
