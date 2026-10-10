import type { QueryClient } from '@tanstack/react-query'
import { endRevokedSession, refreshAccessTokenOnce, TOKEN_REVOKED } from '@/lib/sessionRevoked'

interface ApiEnvelope<T, M> {
  data: T
  meta?: M
  error: null | { code: string; message: string; fields?: Array<{ field: string; message: string }>; details?: unknown }
}

export interface ApiError extends Error {
  code: string
  /** HTTP status of the failed response. */
  status?: number
  /** Field-level validation errors from the envelope, when the backend sends them. */
  fields?: Array<{ field: string; message: string }>
  details?: unknown
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
  const envelope = await withRevocationRetry(qc, () => apiFetchOnce<T, never>(qc, url, options))
  return envelope?.data as T
}

/** Authenticated fetch for a paginated list: the rows and the envelope's meta (same rules as apiFetch). */
export async function apiFetchPaginated<T, M>(
  qc: QueryClient,
  url: string,
  options?: RequestInit,
): Promise<{ data: T; meta: M }> {
  const envelope = await withRevocationRetry(qc, () => apiFetchOnce<T, M>(qc, url, options))
  if (!envelope) throw new Error('Request failed: 204')
  return { data: envelope.data, meta: envelope.meta as M }
}

/**
 * Runs one authenticated request. A TOKEN_REVOKED rejection refreshes the token once (shared with
 * concurrent callers) and retries once. The session ends when the refresh fails, yields the same
 * token, or the retry is revoked again.
 */
export async function withRevocationRetry<R>(qc: QueryClient, attempt: () => Promise<R>): Promise<R> {
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  try {
    return await attempt()
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
      return await attempt()
    } catch (retryErr) {
      if ((retryErr as { code?: unknown } | null)?.code === TOKEN_REVOKED) endRevokedSession(qc)
      throw retryErr
    }
  }
}

async function apiFetchOnce<T, M>(
  qc: QueryClient,
  url: string,
  options?: RequestInit,
): Promise<ApiEnvelope<T, M> | undefined> {
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  const authHeader: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {}
  const res = await fetch(url, {
    ...options,
    credentials: 'include',
    headers: { ...authHeader, ...options?.headers },
  })
  // 204 No Content (e.g. DELETE endpoints) has no body to parse.
  if (res.status === 204) {
    return undefined
  }
  // A proxy or gateway error (502, HTML body) must surface as a normal Error, not a JSON SyntaxError.
  let body: ApiEnvelope<T, M> | null = null
  try {
    body = (await res.json()) as ApiEnvelope<T, M>
  } catch {
    body = null
  }
  if (body?.error) {
    const err = new Error(body.error.message) as ApiError
    err.code = body.error.code ?? 'ERR_UNKNOWN'
    err.status = res.status
    err.fields = body.error.fields
    err.details = body.error.details
    throw err
  }
  if (!res.ok || !body) {
    const err = new Error(`Request failed: ${res.status}`) as ApiError
    err.code = 'ERR_HTTP'
    err.status = res.status
    throw err
  }
  return body
}
