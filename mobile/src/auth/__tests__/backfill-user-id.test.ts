/* eslint-disable @typescript-eslint/no-require-imports -- openapi-fetch captures globalThis.fetch at createClient(), so modules load after the mock */
const mockStore = new Map<string, string>();
jest.mock('expo-secure-store', () => ({
  getItemAsync: jest.fn(async (key: string) => mockStore.get(key) ?? null),
  setItemAsync: jest.fn(async (key: string, value: string) => void mockStore.set(key, value)),
  deleteItemAsync: jest.fn(async (key: string) => void mockStore.delete(key)),
}));
jest.mock('@/api/env', () => ({ apiBaseUrl: 'http://api.test' }));
jest.mock('@/features/push/register', () => ({ unregisterCurrentDevice: async () => undefined }));
jest.mock('@/lib/query-client', () => ({ queryClient: { clear: jest.fn() } }));

let mockFetch: (req: Request) => Response;
globalThis.fetch = jest.fn(async (input: RequestInfo | URL, init?: RequestInit) =>
  mockFetch(input instanceof Request ? input : new Request(input, init))
) as typeof fetch;

const { backfillCurrentUserId } = require('../context') as typeof import('../context');

beforeEach(() => {
  mockStore.clear();
  mockStore.set('adera.access_token', 'access');
  mockStore.set('adera.refresh_token', 'refresh');
});

it('stores the user id for a session created before it was saved', async () => {
  mockFetch = () =>
    new Response(JSON.stringify({ data: { id: 'user-42' } }), { status: 200, headers: { 'Content-Type': 'application/json' } });
  await backfillCurrentUserId();
  expect(mockStore.get('adera.user_id')).toBe('user-42');
});

it('leaves it for the next launch when offline', async () => {
  mockFetch = () => {
    throw new TypeError('Network request failed');
  };
  await expect(backfillCurrentUserId()).resolves.toBeUndefined();
  expect(mockStore.has('adera.user_id')).toBe(false);
});
