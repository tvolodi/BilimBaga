import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { QueryClient } from '@tanstack/react-query'
import { downloadFile, downloadErrorKey } from './download'
import { downloadAdminCertificate, exportEmployeeRecord } from './employees'
import { exportAuditLog } from './audit'
import { downloadDashboardPdf, downloadExamCsv } from './reports'

function blobResponse(status = 200): Response {
  return new Response('x', { status })
}
function jsonResponse(body: unknown, status: number): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function makeQc(token: string | null = 'tok-1') {
  const qc = new QueryClient()
  qc.setQueryData(['auth', 'accessToken'], token)
  return qc
}

describe('downloadFile', () => {
  let fetchMock: ReturnType<typeof vi.fn>
  let revoke: ReturnType<typeof vi.fn>
  let clicked: HTMLAnchorElement[]

  beforeEach(() => {
    fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    revoke = vi.fn()
    URL.createObjectURL = vi.fn(() => 'blob:mock')
    URL.revokeObjectURL = revoke as unknown as typeof URL.revokeObjectURL
    clicked = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      clicked.push(this)
    })
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('sends the Bearer token, downloads via blob URL and revokes it', async () => {
    fetchMock.mockResolvedValueOnce(blobResponse())
    await downloadFile(makeQc(), '/api/v1/x', 'file.pdf')
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/x')
    expect(init.headers.Authorization).toBe('Bearer tok-1')
    expect(clicked).toHaveLength(1)
    expect(clicked[0].download).toBe('file.pdf')
    expect(revoke).toHaveBeenCalledWith('blob:mock')
  })

  it('raises the password-change flag on 403 PASSWORD_CHANGE_REQUIRED (ISS-160)', async () => {
    const qc = makeQc()
    fetchMock.mockResolvedValueOnce(jsonResponse({ data: null, error: { code: 'PASSWORD_CHANGE_REQUIRED' } }, 403))
    await expect(downloadFile(qc, '/api/v1/x', 'f.pdf')).rejects.toMatchObject({ code: 'PASSWORD_CHANGE_REQUIRED', status: 403 })
    expect(qc.getQueryData(['auth', 'passwordChangeRequired'])).toBe(true)
  })

  it('omits Authorization when there is no token', async () => {
    fetchMock.mockResolvedValueOnce(blobResponse())
    await downloadFile(makeQc(null), '/api/v1/x', 'f.pdf')
    expect(fetchMock.mock.calls[0][1].headers.Authorization).toBeUndefined()
  })

  it('refreshes the token on 401 and retries once with the new token', async () => {
    const qc = makeQc('old')
    fetchMock
      .mockResolvedValueOnce(jsonResponse({ data: null, error: { code: 'ERR_UNAUTHORIZED' } }, 401))
      .mockResolvedValueOnce(jsonResponse({ data: { access_token: 'new' }, error: null }, 200))
      .mockResolvedValueOnce(blobResponse())
    await downloadFile(qc, '/api/v1/x', 'f.pdf')
    expect(fetchMock.mock.calls[1][0]).toBe('/api/v1/auth/refresh')
    expect(fetchMock.mock.calls[2][1].headers.Authorization).toBe('Bearer new')
    expect(qc.getQueryData(['auth', 'accessToken'])).toBe('new')
    expect(clicked).toHaveLength(1)
  })

  it('throws ERR_UNAUTHORIZED when refresh fails', async () => {
    fetchMock
      .mockResolvedValueOnce(jsonResponse({}, 401))
      .mockResolvedValueOnce(jsonResponse({}, 401))
    await expect(downloadFile(makeQc(), '/api/v1/x', 'f.pdf')).rejects.toMatchObject({
      code: 'ERR_UNAUTHORIZED',
    })
    expect(clicked).toHaveLength(0)
    expect(URL.createObjectURL).not.toHaveBeenCalled()
  })

  it('does not retry a second time when the retried request is 401 again', async () => {
    fetchMock
      .mockResolvedValueOnce(jsonResponse({}, 401))
      .mockResolvedValueOnce(jsonResponse({ data: { access_token: 'new' }, error: null }, 200))
      .mockResolvedValueOnce(jsonResponse({}, 401))
    await expect(downloadFile(makeQc('old'), '/api/v1/x', 'f.pdf')).rejects.toMatchObject({
      code: 'ERR_UNAUTHORIZED',
    })
    expect(fetchMock).toHaveBeenCalledTimes(3)
    expect(clicked).toHaveLength(0)
  })

  it('throws the server error code on non-OK responses', async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse({ error: { code: 'ERR_FORBIDDEN' } }, 403))
    await expect(downloadFile(makeQc(), '/api/v1/x', 'f.pdf')).rejects.toMatchObject({
      code: 'ERR_FORBIDDEN',
      status: 403,
    })
  })

  it('maps errors to i18n keys', () => {
    expect(downloadErrorKey({ code: 'ERR_UNAUTHORIZED' })).toBe('download.session_expired')
    expect(downloadErrorKey({ code: 'SESSION_NOT_PASSED' })).toBe('download.session_not_passed')
    expect(downloadErrorKey({ code: 'EXAM_NOT_CERTIFIABLE' })).toBe('download.exam_not_certifiable')
    expect(downloadErrorKey({ code: 'ERR_OTHER' })).toBe('download.failed')
    expect(downloadErrorKey(new Error('boom'))).toBe('download.failed')
  })
})

describe('download wrappers send Bearer to the right endpoints', () => {
  let fetchMock: ReturnType<typeof vi.fn>
  beforeEach(() => {
    fetchMock = vi.fn().mockImplementation(() => Promise.resolve(blobResponse()))
    vi.stubGlobal('fetch', fetchMock)
    URL.createObjectURL = vi.fn(() => 'blob:mock')
    URL.revokeObjectURL = vi.fn()
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it.each([
    ['admin certificate', (qc: QueryClient) => downloadAdminCertificate(qc, 's1', 'CODE'), '/api/v1/admin/sessions/s1/certificate'],
    ['dashboard pdf', (qc: QueryClient) => downloadDashboardPdf(qc, '2026-01-01', '2026-01-31'), '/api/v1/admin/dashboard/export?from=2026-01-01&to=2026-01-31'],
    ['employee record csv', (qc: QueryClient) => exportEmployeeRecord(qc, 'u1'), '/api/v1/admin/users/u1/record/export'],
    ['audit log csv', (qc: QueryClient) => exportAuditLog(qc, { from: '2026-01-01', actions: ['user.login'] }), '/api/v1/audit/export?from=2026-01-01&action=user.login'],
    ['exam csv', (qc: QueryClient) => downloadExamCsv(qc, 'e1', 'My Exam'), '/api/v1/admin/exams/e1/results/export'],
  ])('%s', async (_n, call, expectedUrl) => {
    await call(makeQc())
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe(expectedUrl)
    expect(init.headers.Authorization).toBe('Bearer tok-1')
  })
})
