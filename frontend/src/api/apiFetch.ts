import type { QueryClient } from '@tanstack/react-query'
import { endRevokedSession, refreshAccessTokenOnce, TOKEN_REVOKED } from '@/lib/sessionRevoked'

interface ApiResponse<T> {
  data: T
  error: null | { code: string; message: string; details?: Record<string, unknown> }
}

export interface ApiError extends Error {
  code: string
  details?: Record<string, unknown>
}

/**
 * Authenticated fetch for all API modules.
 *
 * Reads the JWT from the React Query cache (key: ['auth', 'accessToken']) and
 * attaches it as a Bearer token. Every API module MUST use this function instead
 * of rolling its own fetch wrapper — that is the only way to guarantee the
 * Authorization header is always sent.
 *
 * ISS-249: a 401 TOKEN_REVOKED means the JWT claims (role/department/status) no longer match
 * the DB. The token is refreshed once and the request retried once; if the refresh fails, or
 * yields the same token, or the retry is revoked again, the session ends (token cleared,
 * RequireAuth redirects to /login, LoginPage shows a localized notice). Never loops.
 */
export async function apiFetch<T>(
  qc: QueryClient,
  url: string,
  options?: RequestInit,
): Promise<T> {
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  try {
    return await apiFetchOnce<T>(qc, url, options)
  } catch (err) {
    if ((err as { code?: unknown } | null)?.code !== TOKEN_REVOKED) throw err
    // A concurrent request may already have refreshed the token since this one started.
    const current = qc.getQueryData<string | null>(['auth', 'accessToken'])
    const fresh = current && current !== token ? current : await refreshAccessTokenOnce(qc)
    if (!fresh || fresh === token) {
      endRevokedSession(qc)
      throw err
    }
    try {
      return await apiFetchOnce<T>(qc, url, options)
    } catch (retryErr) {
      if ((retryErr as { code?: unknown } | null)?.code === TOKEN_REVOKED) endRevokedSession(qc)
      throw retryErr
    }
  }
}

async function apiFetchOnce<T>(
  qc: QueryClient,
  url: string,
  options?: RequestInit,
): Promise<T> {
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  const authHeader: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {}
  const res = await fetch(url, {
    ...options,
    headers: { ...authHeader, ...options?.headers },
  })
  // 204 No Content (e.g. DELETE endpoints) has no body to parse.
  if (res.status === 204) {
    return undefined as T
  }
  const body: ApiResponse<T> = await res.json()
  if (body.error) {
    const err = new Error(body.error.message) as ApiError
    err.code = body.error.code
    err.details = body.error.details
    throw err
  }
  if (!res.ok) {
    const err = new Error(`Request failed: ${res.status}`) as ApiError
    err.code = 'ERR_HTTP'
    throw err
  }
  return body.data
}
