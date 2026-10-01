/* eslint-disable @typescript-eslint/no-require-imports -- openapi-fetch captures globalThis.fetch at createClient(), so modules must load after the fetch mock below */
jest.mock('@/api/env', () => ({ apiBaseUrl: 'http://api.test' }));
jest.mock('expo-constants', () => ({ expoConfig: { version: '1.0.0' } }));

// A request that never answers, like a stalled mobile connection — it only
// settles when aborted.
globalThis.fetch = jest.fn(
  (input: RequestInfo | URL, init?: RequestInit) =>
    new Promise<Response>((_, reject) => {
      const signal = input instanceof Request ? input.signal : init?.signal;
      signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')));
    })
) as typeof fetch;

const { fetchVersionGate, VERSION_CHECK_TIMEOUT_MS } = require('../queries') as typeof import('../queries');

beforeEach(() => jest.useFakeTimers());
afterEach(() => jest.useRealTimers());

it('gives up on a stalled version check instead of holding the splash screen', async () => {
  const result = fetchVersionGate();
  const settled = expect(result).rejects.toThrow();

  jest.advanceTimersByTime(VERSION_CHECK_TIMEOUT_MS);

  await settled;
});
