import { describe, it, expect, vi, afterEach } from 'vitest'
import { QueryClient } from '@tanstack/react-query'
import { apiFetch, apiFetchPaginated } from './apiFetch'
import { SESSION_REVOKED_KEY } from '@/lib/sessionRevoked'

function qcWithToken(token = 't0k3n') {
  const qc = new QueryClient()
  qc.setQueryData(['auth', 'accessToken'], token)
  return qc
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('apiFetch', () => {
  it('returns undefined for a 204 No Content response without parsing a body (DELETE /roles/{id})', async () => {
    const json = vi.fn().mockRejectedValue(new SyntaxError('Unexpected end of JSON input'))
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 204, json }))
    const result = await apiFetch<void>(qcWithToken(), '/api/v1/roles/abc', { method: 'DELETE' })
    expect(result).toBeUndefined()
    expect(json).not.toHaveBeenCalled()
  })

  it('sends the Bearer token', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204, json: vi.fn() })
    vi.stubGlobal('fetch', fetchMock)
    await apiFetch<void>(qcWithToken('abc'), '/x', { method: 'DELETE' })
    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer abc')
  })

  it('returns data for a normal 200 envelope', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => ({ data: { id: 1 }, error: null }) }),
    )
    await expect(apiFetch<{ id: number }>(qcWithToken(), '/x')).resolves.toEqual({ id: 1 })
  })

  it('throws an ApiError carrying code and details for an error envelope', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 409,
        json: async () => ({ data: null, error: { code: 'ROLE_IN_USE', message: 'in use', details: { count: 3 } } }),
      }),
    )
    await expect(apiFetch(qcWithToken(), '/x', { method: 'DELETE' })).rejects.toMatchObject({
      code: 'ROLE_IN_USE',
      details: { count: 3 },
    })
  })

  it('reports a gateway HTML body as ERR_HTTP, not a JSON SyntaxError', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 502,
        json: async () => {
          throw new SyntaxError('Unexpected token < in JSON')
        },
      }),
    )
    await expect(apiFetch(qcWithToken(), '/x')).rejects.toMatchObject({
      code: 'ERR_HTTP',
      message: 'Request failed: 502',
      status: 502,
    })
  })

  it('falls back to ERR_UNKNOWN for an error envelope without a code', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: false, status: 500, json: async () => ({ data: null, error: { message: 'boom' } }) }),
    )
    await expect(apiFetch(qcWithToken(), '/x')).rejects.toMatchObject({ code: 'ERR_UNKNOWN', message: 'boom' })
  })

  it('carries the HTTP status and field errors from the envelope', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 422,
        json: async () => ({
          data: null,
          error: { code: 'VALIDATION', message: 'bad', fields: [{ field: 'email', message: 'taken' }] },
        }),
      }),
    )
    await expect(apiFetch(qcWithToken(), '/x')).rejects.toMatchObject({
      status: 422,
      fields: [{ field: 'email', message: 'taken' }],
    })
  })

  it('apiFetchPaginated returns the rows and the envelope meta', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ data: [{ id: 1 }], meta: { page: 2, per_page: 20, total: 41 }, error: null }),
      }),
    )
    await expect(apiFetchPaginated<{ id: number }[], { page: number; per_page: number; total: number }>(qcWithToken(), '/x')).resolves.toEqual({
      data: [{ id: 1 }],
      meta: { page: 2, per_page: 20, total: 41 },
    })
  })
})

describe('apiFetch TOKEN_REVOKED handling (ISS-249)', () => {
  const revoked = {
    ok: false,
    status: 401,
    json: async () => ({ data: null, error: { code: 'TOKEN_REVOKED', message: 'account changed' } }),
  }
  const okRes = { ok: true, status: 200, json: async () => ({ data: { id: 7 }, error: null }) }
  const refreshRes = (token: string) => ({
    ok: true,
    status: 200,
    json: async () => ({ data: { access_token: token }, error: null }),
  })

  function route(handlers: Record<string, ((n: number) => unknown)[]>) {
    const counts: Record<string, number> = {}
    return vi.fn(async (url: string) => {
      const n = (counts[url] = (counts[url] ?? 0) + 1)
      const list = handlers[url]
      return list[Math.min(n, list.length) - 1](n)
    })
  }

  it('refreshes once and retries with the fresh token', async () => {
    const fetchMock = route({
      '/x': [() => revoked, () => okRes],
      '/api/v1/auth/refresh': [() => refreshRes('fresh')],
    })
    vi.stubGlobal('fetch', fetchMock)
    const qc = qcWithToken('old')
    await expect(apiFetch<{ id: number }>(qc, '/x')).resolves.toEqual({ id: 7 })
    expect(qc.getQueryData(['auth', 'accessToken'])).toBe('fresh')
    expect(fetchMock).toHaveBeenCalledTimes(3)
    const retryInit = fetchMock.mock.calls[2] as unknown as [string, RequestInit]
    expect((retryInit[1].headers as Record<string, string>).Authorization).toBe('Bearer fresh')
    expect(qc.getQueryData(SESSION_REVOKED_KEY)).toBeUndefined()
  })

  it('ends the session when the refresh fails', async () => {
    const fetchMock = route({
      '/x': [() => revoked],
      '/api/v1/auth/refresh': [() => ({ ok: false, status: 401, json: async () => ({}) })],
    })
    vi.stubGlobal('fetch', fetchMock)
    const qc = qcWithToken('old')
    qc.setQueryData(['auth', 'currentUser'], { id: 'u' })
    await expect(apiFetch(qc, '/x')).rejects.toMatchObject({ code: 'TOKEN_REVOKED' })
    expect(qc.getQueryData(['auth', 'accessToken'])).toBeNull()
    expect(qc.getQueryData(['auth', 'currentUser'])).toBeNull()
    expect(qc.getQueryData(SESSION_REVOKED_KEY)).toBe(true)
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('ends the session when refresh returns the same stale token', async () => {
    const fetchMock = route({
      '/x': [() => revoked],
      '/api/v1/auth/refresh': [() => refreshRes('old')],
    })
    vi.stubGlobal('fetch', fetchMock)
    const qc = qcWithToken('old')
    await expect(apiFetch(qc, '/x')).rejects.toMatchObject({ code: 'TOKEN_REVOKED' })
    expect(qc.getQueryData(SESSION_REVOKED_KEY)).toBe(true)
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('does not loop: a second TOKEN_REVOKED after retry logs out, with one refresh only', async () => {
    const fetchMock = route({
      '/x': [() => revoked],
      '/api/v1/auth/refresh': [() => refreshRes('fresh')],
    })
    vi.stubGlobal('fetch', fetchMock)
    const qc = qcWithToken('old')
    await expect(apiFetch(qc, '/x')).rejects.toMatchObject({ code: 'TOKEN_REVOKED' })
    expect(fetchMock).toHaveBeenCalledTimes(3)
    expect(qc.getQueryData(['auth', 'accessToken'])).toBeNull()
    expect(qc.getQueryData(SESSION_REVOKED_KEY)).toBe(true)
  })

  it('does not refresh for other error codes', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({ data: null, error: { code: 'TOKEN_EXPIRED', message: 'x' } }),
    })
    vi.stubGlobal('fetch', fetchMock)
    await expect(apiFetch(qcWithToken(), '/x')).rejects.toMatchObject({ code: 'TOKEN_EXPIRED' })
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })
})
