import type { QueryClient } from '@tanstack/react-query'

export interface DownloadError extends Error {
  code: string
  status?: number
}

function downloadError(code: string, status?: number): DownloadError {
  const err = new Error(code) as DownloadError
  err.code = code
  err.status = status
  return err
}

/** i18n key describing a download failure, for display to the user. */
export function downloadErrorKey(err: unknown): string {
  const code = (err as Partial<DownloadError> | null)?.code
  return code === 'ERR_UNAUTHORIZED' ? 'download.session_expired' : 'download.failed'
}

function authInit(qc: QueryClient, options?: RequestInit): RequestInit {
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return {
    ...options,
    credentials: 'include',
    headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}), ...options?.headers },
  }
}

/** Refresh the access token once (same endpoint as useRefreshToken); returns true on success. */
async function refreshAccessToken(qc: QueryClient): Promise<boolean> {
  try {
    const res = await fetch('/api/v1/auth/refresh', { method: 'POST', credentials: 'include' })
    if (!res.ok) return false
    const json = (await res.json()) as { data?: { access_token?: string } | null; error?: unknown }
    const token = json.data?.access_token
    if (json.error || !token) return false
    qc.setQueryData(['auth', 'accessToken'], token)
    return true
  } catch {
    return false
  }
}

/**
 * Authenticated file download: fetch (Bearer header) -> blob -> object URL -> anchor click.
 * On 401 the access token is refreshed once and the request retried. The object URL is
 * always revoked. Throws DownloadError (code ERR_UNAUTHORIZED / server code / ERR_DOWNLOAD).
 */
export async function downloadFile(
  qc: QueryClient,
  url: string,
  fallbackFilename: string,
): Promise<void> {
  let res = await fetch(url, authInit(qc))
  if (res.status === 401 && (await refreshAccessToken(qc))) {
    res = await fetch(url, authInit(qc))
  }
  if (!res.ok) {
    if (res.status === 401) throw downloadError('ERR_UNAUTHORIZED', 401)
    const body = (await res.json().catch(() => null)) as { error?: { code?: string } } | null
    throw downloadError(body?.error?.code ?? 'ERR_DOWNLOAD', res.status)
  }
  const blob = await res.blob()
  const objectUrl = URL.createObjectURL(blob)
  try {
    const a = document.createElement('a')
    a.href = objectUrl
    a.download = fallbackFilename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  } finally {
    URL.revokeObjectURL(objectUrl)
  }
}
