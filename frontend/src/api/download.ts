import type { QueryClient } from '@tanstack/react-query'
import { markPasswordChangeRequired, PASSWORD_CHANGE_REQUIRED } from '@/lib/passwordChangeRequired'
import { endRevokedSession, refreshAccessTokenOnce, TOKEN_REVOKED } from '@/lib/sessionRevoked'

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

const DOWNLOAD_ERROR_KEYS: Record<string, string> = {
  ERR_UNAUTHORIZED: 'download.session_expired',
  SESSION_NOT_PASSED: 'download.session_not_passed',
  EXAM_NOT_CERTIFIABLE: 'download.exam_not_certifiable',
}

/** i18n key describing a download failure, for display to the user. */
export function downloadErrorKey(err: unknown): string {
  const code = (err as Partial<DownloadError> | null)?.code
  return code && Object.prototype.hasOwnProperty.call(DOWNLOAD_ERROR_KEYS, code) ? DOWNLOAD_ERROR_KEYS[code] : 'download.failed'
}

function authInit(qc: QueryClient, options?: RequestInit): RequestInit {
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return {
    ...options,
    credentials: 'include',
    headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}), ...options?.headers },
  }
}

/** True when a 401 response says the token was revoked (ISS-249). Reads a clone, so the body stays usable. */
async function isRevoked(res: Response): Promise<boolean> {
  const body = (await res.clone().json().catch(() => null)) as { error?: { code?: string } } | null
  return body?.error?.code === TOKEN_REVOKED
}

/**
 * Authenticated file download: fetch (Bearer header) -> blob -> object URL -> anchor click.
 * A blob cannot go through the shared apiFetch (it parses JSON), so this keeps its own request, but
 * uses the same token rules: on 401 the token is refreshed once and the request retried; a revoked
 * session that cannot be refreshed, or whose retry is revoked again, ends. The object URL is always
 * revoked. Throws DownloadError (code ERR_UNAUTHORIZED / server code / ERR_DOWNLOAD).
 */
export async function downloadFile(
  qc: QueryClient,
  url: string,
  fallbackFilename: string,
): Promise<void> {
  const sent = qc.getQueryData<string | null>(['auth', 'accessToken'])
  let res = await fetch(url, authInit(qc))
  if (res.status === 401) {
    const revoked = await isRevoked(res)
    // A concurrent request may already have refreshed the token since this one started.
    const current = qc.getQueryData<string | null>(['auth', 'accessToken'])
    const fresh = current && current !== sent ? current : await refreshAccessTokenOnce(qc)
    if (fresh && fresh !== sent) {
      res = await fetch(url, authInit(qc))
    } else if (revoked) {
      endRevokedSession(qc)
    }
    if (res.status === 401) {
      if (await isRevoked(res)) endRevokedSession(qc)
      throw downloadError('ERR_UNAUTHORIZED', 401)
    }
  }
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: { code?: string } } | null
    if (res.status === 403 && body?.error?.code === PASSWORD_CHANGE_REQUIRED) markPasswordChangeRequired(qc)
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
