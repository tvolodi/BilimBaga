import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement, type ReactNode } from 'react'
import { downloadFile } from '@/api/download'
import { useAuditLog, useAuditFilterOptions, exportAuditLog, type AuditFilters } from './audit'

vi.mock('@/api/download', () => ({ downloadFile: vi.fn(async () => undefined) }))

afterEach(() => vi.unstubAllGlobals())

beforeEach(() => {
  vi.mocked(downloadFile).mockClear()
})

function makeWrapper(token: string | null = 'tok') {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  qc.setQueryData(['auth', 'accessToken'], token)
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children)
  return { qc, wrapper }
}

function okFetch(data: unknown) {
  return vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => ({ data, error: null }) })
}

const filters: AuditFilters = {
  from: '2026-01-01',
  to: '2026-01-31',
  actor: 'user-1',
  actions: ['user.login', 'exam.create'],
  entityType: 'exam',
}

describe('useAuditLog', () => {
  it('requests a page with every filter in the query string', async () => {
    const fetchMock = okFetch({ items: [], meta: { page: 2, per_page: 50, total: 0 } })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper('tok-9')

    const { result } = renderHook(() => useAuditLog(filters, 2), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe(
      '/api/v1/audit?from=2026-01-01&to=2026-01-31&actor=user-1&action=user.login&action=exam.create&entity_type=exam&page=2&per_page=50',
    )
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer tok-9')
  })

  it('requests the first page with no filters when none are set', async () => {
    const fetchMock = okFetch({ items: [], meta: { page: 1, per_page: 50, total: 0 } })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()

    const { result } = renderHook(() => useAuditLog({}, 1), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect((fetchMock.mock.calls[0] as [string])[0]).toBe('/api/v1/audit?page=1&per_page=50')
  })

  it('skips an empty action list', async () => {
    const fetchMock = okFetch({ items: [], meta: { page: 1, per_page: 50, total: 0 } })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()

    const { result } = renderHook(() => useAuditLog({ actions: [] }, 1), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect((fetchMock.mock.calls[0] as [string])[0]).toBe('/api/v1/audit?page=1&per_page=50')
  })
})

describe('exportAuditLog', () => {
  it('downloads the CSV export with the same filters as the list', async () => {
    const { qc } = makeWrapper()

    await exportAuditLog(qc, filters)

    expect(downloadFile).toHaveBeenCalledWith(
      qc,
      '/api/v1/audit/export?from=2026-01-01&to=2026-01-31&actor=user-1&action=user.login&action=exam.create&entity_type=exam',
      'audit-export.csv',
    )
  })

  it('passes a download failure to the caller', async () => {
    vi.mocked(downloadFile).mockRejectedValueOnce(new Error('export too large'))
    const { qc } = makeWrapper()

    await expect(exportAuditLog(qc, { from: '2026-01-01' })).rejects.toThrow('export too large')
  })
})

describe('useAuditFilterOptions', () => {
  it('requests the tenant options from /audit/actions with the access token, kept for five minutes', async () => {
    const fetchMock = okFetch({ actions: ['exam.create'], entity_types: ['exam'] })
    vi.stubGlobal('fetch', fetchMock)
    const { qc, wrapper } = makeWrapper('tok-3')

    const { result } = renderHook(() => useAuditFilterOptions(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/audit/actions')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer tok-3')
    expect(result.current.data).toEqual({ actions: ['exam.create'], entity_types: ['exam'] })
    expect(qc.getQueryCache().find({ queryKey: ['audit-filter-options'] })?.options).toMatchObject({ staleTime: 5 * 60 * 1000 })
  })

  it('reports a failed request, so the filter bar can show its inline error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('network down')))
    const { wrapper } = makeWrapper()

    const { result } = renderHook(() => useAuditFilterOptions(), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})
