import type { QueryClient } from '@tanstack/react-query'
import { clearE2eToken, readE2eToken, writeE2eToken } from '@/lib/e2eTokenSeed'

/** Backend 401 code: the role/department/status (or password) behind the JWT changed (ISS-240). */
export const TOKEN_REVOKED = 'TOKEN_REVOKED'

/** Query key holding the "session ended because the token was revoked" notice for the login page. */
export const SESSION_REVOKED_KEY = ['auth', 'sessionRevoked'] as const

/**
 * End the session client-side: drop the token and user (RequireAuth then redirects to /login)
 * and raise the notice LoginPage shows. Idempotent.
 */
export function endRevokedSession(qc: QueryClient): void {
  clearE2eToken()
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
