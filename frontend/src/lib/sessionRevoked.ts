import type { QueryClient } from '@tanstack/react-query'
import { clearE2eToken, readE2eToken, writeE2eToken } from '@/lib/e2eTokenSeed'
import { clearNonPublicQueries } from '@/lib/queryCache'

/** Backend 401 code: the role/department/status (or password) behind the JWT changed (ISS-240). */
export const TOKEN_REVOKED = 'TOKEN_REVOKED'

/** Query key holding the "session ended because the token was revoked" notice for the login page. */
export const SESSION_REVOKED_KEY = ['auth', 'sessionRevoked'] as const

/**
 * End the session client-side (AC-11): drop the token, the user and every per-user cache so none of it
 * outlives the session (#436), then raise the notice LoginPage shows. Guards subscribed to the token
 * redirect to /login and keep the requested path in location state, so sign-in can restore it. Idempotent.
 */
export function endRevokedSession(qc: QueryClient): void {
  clearE2eToken()
  clearNonPublicQueries(qc)
  qc.setQueryData(SESSION_REVOKED_KEY, true)
  qc.setQueryData(['auth', 'accessToken'], null)
  qc.setQueryData(['auth', 'currentUser'], null)
}

let inflight: Promise<string | null> | null = null

/**
 * Refresh the access token once (shared by concurrent callers). Resolves to the new token and
 * stores it in the cache, or null when the refresh failed.
 */
export function refreshAccessTokenOnce(qc: QueryClient): Promise<string | null> {
  if (inflight) return inflight
  inflight = (async () => {
    try {
      const res = await fetch('/api/v1/auth/refresh', { method: 'POST', credentials: 'include' })
      if (!res.ok) return null
      const json = (await res.json()) as { data?: { access_token?: string } | null; error?: unknown }
      const token = json.data?.access_token
      if (json.error || !token) return null
      if (readE2eToken()) writeE2eToken(token)
      qc.setQueryData(['auth', 'accessToken'], token)
      return token
    } catch {
      return null
    } finally {
      inflight = null
    }
  })()
  return inflight
}
