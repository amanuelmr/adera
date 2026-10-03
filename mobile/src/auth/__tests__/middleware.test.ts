/* eslint-disable @typescript-eslint/no-require-imports -- openapi-fetch captures globalThis.fetch at createClient(), so modules must load after the fetch mock below */
const mockStore = new Map<string, string>();

jest.mock('expo-secure-store', () => ({
  getItemAsync: jest.fn(async (key: string) => mockStore.get(key) ?? null),
  setItemAsync: jest.fn(async (key: string, value: string) => void mockStore.set(key, value)),
  deleteItemAsync: jest.fn(async (key: string) => void mockStore.delete(key)),
}));

jest.mock('@/api/env', () => ({ apiBaseUrl: 'http://api.test' }));

// Mirrors the real unregisterCurrentDevice: an authenticated DELETE through
// the same apiClient/middleware, which is what made the deadlock possible.
jest.mock('@/features/push/register', () => ({
  unregisterCurrentDevice: async () => {
    await require('@/api/client').apiClient.POST('/api/v1/users/me/devices/unregister', { body: { token: 'fcm-token' } }).catch(() => undefined);
  },
}));

type Route = (req: Request) => Response | Promise<Response>;

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
}

const unauthorized = () => json(401, { error: { code: 'unauthorized', message: 'invalid or expired token' } });

let route: Route;
const calls: { method: string; path: string; auth: string | null }[] = [];

globalThis.fetch = jest.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
  const req = input instanceof Request ? input : new Request(input, init);
  calls.push({ method: req.method, path: new URL(req.url).pathname, auth: req.headers.get('Authorization') });
  return route(req);
}) as typeof fetch;

// Not isolated per test, so the mocked unregister shares the exact apiClient
// instance the middleware is attached to, as in the app.
require('../middleware');
const { apiClient } = require('@/api/client') as typeof import('@/api/client');
const { onForcedSignOut } = require('../events') as typeof import('../events');

function load() {
  return { apiClient, onForcedSignOut };
}

let unsubscribers: (() => void)[] = [];
afterEach(() => {
  unsubscribers.forEach((u) => u());
  unsubscribers = [];
});

function listen() {
  const signOut = jest.fn();
  unsubscribers.push(onForcedSignOut(signOut));
  return signOut;
}

async function withTimeout<T>(promise: Promise<T>, ms = 2000): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  try {
    return await Promise.race([
      promise,
      new Promise<T>((_, reject) => {
        timer = setTimeout(() => reject(new Error('timed out — deadlock?')), ms);
      }),
    ]);
  } finally {
    clearTimeout(timer);
  }
}

beforeEach(() => {
  mockStore.clear();
  mockStore.set('adera.access_token', 'old-access');
  mockStore.set('adera.refresh_token', 'old-refresh');
  calls.length = 0;
});

it('signs out without deadlocking when the refresh token is dead', async () => {
  route = () => unauthorized();
  const { apiClient } = load();
  const signOut = listen();

  const result = await withTimeout(apiClient.GET('/api/v1/users/me'));

  expect(result.response.status).toBe(401);
  expect(signOut).toHaveBeenCalledTimes(1);
  expect(mockStore.has('adera.refresh_token')).toBe(false);
  expect(calls.filter((c) => c.path === '/api/v1/auth/refresh')).toHaveLength(1);
});

it('keeps the session when refresh fails transiently (5xx / 429)', async () => {
  route = (req) =>
    new URL(req.url).pathname === '/api/v1/auth/refresh'
      ? json(503, { error: { code: 'internal_error', message: 'unavailable' } })
      : unauthorized();
  const { apiClient } = load();
  const signOut = listen();

  const result = await withTimeout(apiClient.GET('/api/v1/users/me'));

  expect(result.response.status).toBe(401);
  expect(signOut).not.toHaveBeenCalled();
  expect(mockStore.get('adera.refresh_token')).toBe('old-refresh');
});

it('refreshes once for concurrent 401s and retries each with the new token', async () => {
  route = (req) => {
    const path = new URL(req.url).pathname;
    if (path === '/api/v1/auth/refresh') {
      return json(200, { data: { access_token: 'new-access', refresh_token: 'new-refresh' } });
    }
    return req.headers.get('Authorization') === 'Bearer new-access' ? json(200, { data: {} }) : unauthorized();
  };
  const { apiClient } = load();

  const results = await withTimeout(
    Promise.all([apiClient.GET('/api/v1/users/me'), apiClient.GET('/api/v1/users/me/reviews')])
  );

  expect(results.map((r) => r.response.status)).toEqual([200, 200]);
  expect(calls.filter((c) => c.path === '/api/v1/auth/refresh')).toHaveLength(1);
  expect(mockStore.get('adera.refresh_token')).toBe('new-refresh');
});
