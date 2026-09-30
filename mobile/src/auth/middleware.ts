import type { Middleware } from 'openapi-fetch';

import { apiClient } from '@/api/client';
import { unregisterCurrentDevice } from '@/features/push/register';
import { emitForcedSignOut } from './events';
import { clearTokens, getAccessToken, getRefreshToken, setTokens } from './storage';

/**
 * Endpoints excluded from token attachment and refresh-on-401: attaching a
 * stale token to login/register is pointless, and refreshing in response to
 * a 401 *from* /auth/refresh itself would either loop or mask "this refresh
 * token is dead" as some other failure.
 */
const AUTH_EXEMPT_SCHEMA_PATHS = new Set(['/api/v1/auth/login', '/api/v1/auth/register', '/api/v1/auth/refresh']);

// Device unregistration runs *inside* a failed refresh (performRefresh) and
// during logout. If its 401 triggered a refresh, it would await the very
// refresh that is awaiting it — a deadlock. It's best-effort anyway.
function skipsRefreshOn401(schemaPath: string, method: string): boolean {
  return schemaPath === '/api/v1/users/me/devices' && method === 'DELETE';
}

/**
 * Single-flight guard: the backend's refresh rotation is single-use with
 * family revocation (internal/auth/service.go), so two concurrent refresh
 * calls would revoke every session and force-logout the user. Every caller
 * during a refresh awaits this same promise instead of firing its own call.
 */
type RefreshOutcome = { kind: 'refreshed'; accessToken: string } | { kind: 'dead' } | { kind: 'transient' };

let refreshInFlight: Promise<RefreshOutcome> | null = null;

async function refreshAccessToken(): Promise<RefreshOutcome> {
  if (!refreshInFlight) {
    refreshInFlight = performRefresh().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

async function performRefresh(): Promise<RefreshOutcome> {
  const refreshToken = await getRefreshToken();
  if (!refreshToken) return { kind: 'dead' };

  const result = await apiClient.POST('/api/v1/auth/refresh', { body: { refresh_token: refreshToken } });
  // Only a 401 means the refresh token itself is dead. A 429 or 5xx says
  // nothing about the session — signing out on those turns a backend blip
  // into a forced logout.
  if (result.error && result.response.status !== 401) return { kind: 'transient' };

  const newAccessToken = result.data?.data?.access_token;
  const newRefreshToken = result.data?.data?.refresh_token;
  if (result.error || !newAccessToken || !newRefreshToken) {
    // Best-effort, before the tokens that could authorize it are gone: the
    // access token that triggered this refresh has already 401'd once, so
    // there's no guarantee this succeeds either, but attempting it here is
    // the only chance there ever is — once clearTokens() below runs, no
    // authenticated call can be made again for this session.
    await unregisterCurrentDevice();
    await clearTokens();
    return { kind: 'dead' };
  }

  await setTokens({ accessToken: newAccessToken, refreshToken: newRefreshToken });
  return { kind: 'refreshed', accessToken: newAccessToken };
}

// Requests are cloned here (before fetch consumes the body) so a 401 retry
// can replay them; a Request's body can only be read once.
const pristineRequests = new Map<string, Request>();

export const authMiddleware: Middleware = {
  async onRequest({ request, schemaPath, id }) {
    if (AUTH_EXEMPT_SCHEMA_PATHS.has(schemaPath)) return undefined;

    pristineRequests.set(id, request.clone());
    const token = await getAccessToken();
    if (token) {
      request.headers.set('Authorization', `Bearer ${token}`);
    }
    return request;
  },

  async onResponse({ request, response, schemaPath, id }) {
    const pristine = pristineRequests.get(id);
    pristineRequests.delete(id);

    if (
      response.status !== 401 ||
      AUTH_EXEMPT_SCHEMA_PATHS.has(schemaPath) ||
      skipsRefreshOn401(schemaPath, request.method) ||
      !pristine
    ) {
      return undefined;
    }

    const outcome = await refreshAccessToken();
    if (outcome.kind === 'dead') {
      emitForcedSignOut();
      return undefined;
    }
    if (outcome.kind === 'transient') return undefined;

    const headers = new Headers(pristine.headers);
    headers.set('Authorization', `Bearer ${outcome.accessToken}`);
    return fetch(new Request(pristine, { headers }));
  },

  // onResponse never fires for a network failure (offline, timeout) — common
  // on the flaky connections this app targets — so without this the cloned
  // request (potentially a full review submission body) leaks forever.
  async onError({ id }) {
    pristineRequests.delete(id);
    return undefined;
  },
};

// Registered here (a side effect of importing this module) rather than in
// client.ts, which would need to import this file back to call `.use()` —
// this module already depends on client.ts for `apiClient`.
apiClient.use(authMiddleware);
