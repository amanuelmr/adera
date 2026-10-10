import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';

import { apiClient, unwrap } from '@/api/client';
import type { components } from '@/api/schema';
import { unregisterCurrentDevice } from '@/features/push/register';
import { clearQueryCache } from '@/lib/query-client';
import { onForcedSignOut } from './events';
import { clearTokens, getAccessToken, getCurrentUserId, getRefreshToken, setCurrentUserId, setTokens } from './storage';
// Registers the auth middleware (token attachment + refresh-on-401) on
// `apiClient` as a side effect of import — see middleware.ts.
import './middleware';

export type RegisterInput = {
  displayName: string;
  email?: string;
  phone?: string;
  password: string;
  preferredLanguage?: components['schemas']['Language'];
};

type AuthStatus = 'loading' | 'signedIn' | 'signedOut';

type AuthContextValue = {
  status: AuthStatus;
  register: (input: RegisterInput) => Promise<void>;
  login: (identifier: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  logoutAll: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export async function backfillCurrentUserId(): Promise<void> {
  try {
    const { data } = unwrap(await apiClient.GET('/api/v1/users/me'));
    if (data.id) await setCurrentUserId(data.id);
  } catch {
    // Offline or failing: retried on the next launch.
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>('loading');

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const [accessToken, refreshToken, userId] = await Promise.all([
        getAccessToken(),
        getRefreshToken(),
        getCurrentUserId(),
      ]);
      const signedIn = !!(accessToken && refreshToken);
      if (!cancelled) setStatus(signedIn ? 'signedIn' : 'signedOut');
      // Sessions from before the user id was stored alongside the tokens
      // can't queue offline work (it's tagged per account) until it's known.
      if (signedIn && !userId) await backfillCurrentUserId();
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // Cached queries include per-account data (profile, own reviews,
  // viewer_voted, notifications) — dropped on every account change so the
  // next person on this device never sees the previous one's.
  useEffect(
    () =>
      onForcedSignOut(() => {
        void clearQueryCache();
        setStatus('signedOut');
      }),
    []
  );

  const register = useCallback(async (input: RegisterInput) => {
    const result = await apiClient.POST('/api/v1/auth/register', {
      body: {
        display_name: input.displayName,
        email: input.email,
        phone: input.phone,
        password: input.password,
        preferred_language: input.preferredLanguage,
      },
    });
    const { data } = unwrap(result);
    if (!data.tokens?.access_token || !data.tokens.refresh_token || !data.user?.id) {
      throw new Error('Registration succeeded but no session was returned');
    }
    await setTokens({ accessToken: data.tokens.access_token, refreshToken: data.tokens.refresh_token });
    await setCurrentUserId(data.user.id);
    void clearQueryCache();
    setStatus('signedIn');
  }, []);

  const login = useCallback(async (identifier: string, password: string) => {
    const result = await apiClient.POST('/api/v1/auth/login', { body: { identifier, password } });
    const { data } = unwrap(result);
    if (!data.tokens?.access_token || !data.tokens.refresh_token || !data.user?.id) {
      throw new Error('Login succeeded but no session was returned');
    }
    await setTokens({ accessToken: data.tokens.access_token, refreshToken: data.tokens.refresh_token });
    await setCurrentUserId(data.user.id);
    void clearQueryCache();
    setStatus('signedIn');
  }, []);

  const signOutLocally = useCallback(async () => {
    await clearTokens();
    void clearQueryCache();
    setStatus('signedOut');
  }, []);

  const logout = useCallback(async () => {
    // Before the tokens that authorize it are gone. Best-effort throughout:
    // an unreachable server shouldn't trap the user in a session they asked
    // to leave — the refresh token becomes useless either way once local
    // tokens are cleared.
    await unregisterCurrentDevice().catch(() => undefined);
    await apiClient.POST('/api/v1/auth/logout').catch(() => undefined);
    await signOutLocally();
  }, [signOutLocally]);

  const logoutAll = useCallback(async () => {
    await unregisterCurrentDevice().catch(() => undefined);
    await apiClient.POST('/api/v1/auth/logout-all').catch(() => undefined);
    await signOutLocally();
  }, [signOutLocally]);

  const value = useMemo(
    () => ({ status, register, login, logout, logoutAll }),
    [status, register, login, logout, logoutAll]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext);
  if (!value) throw new Error('useAuth must be used within an AuthProvider');
  return value;
}
