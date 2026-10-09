import { describe, it, expect, vi, afterEach } from 'vitest'
import { QueryClient } from '@tanstack/react-query'
import { apiFetch } from './apiFetch'

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
})
