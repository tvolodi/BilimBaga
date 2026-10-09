import type { QueryClient } from '@tanstack/react-query'

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
 */
export async function apiFetch<T>(
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
