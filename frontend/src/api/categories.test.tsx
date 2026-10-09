import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import {
  useCategories,
  useCreateCategory,
  useUpdateCategory,
  useDeleteCategory,
  flattenCategories,
  getDescendantIds,
  type CategoryNode,
} from './categories'

afterEach(() => vi.unstubAllGlobals())

function makeWrapper(token: string | null = 'tok') {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  qc.setQueryData(['auth', 'accessToken'], token)
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  )
  return { qc, wrapper }
}

function okFetch(data: unknown) {
  return vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => ({ data, error: null }) })
}

function errorFetch(code: string, message: string, details?: Record<string, unknown>, status = 400) {
  return vi.fn().mockResolvedValue({
    ok: false,
    status,
    json: async () => ({ data: null, error: { code, message, details } }),
  })
}

const makeNode = (overrides: Partial<CategoryNode>): CategoryNode => ({
  id: 'cat-default',
  name: 'Default',
  parent_id: null,
  track: null,
  sort_order: 0,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  children: [],
  ...overrides,
})

const sampleNode = makeNode({ id: 'cat-1', name: 'Science' })

describe('useCategories', () => {
  it('GETs the tree with the bearer token and returns the data', async () => {
    const fetchMock = okFetch([sampleNode])
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useCategories(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual([sampleNode])
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/categories')
    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer tok')
  })

  it('omits the Authorization header when no token is cached', async () => {
    const fetchMock = okFetch([])
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper(null)
    const { result } = renderHook(() => useCategories(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect((init.headers as Record<string, string>).Authorization).toBeUndefined()
  })

  it('surfaces an API error body with its code', async () => {
    vi.stubGlobal('fetch', errorFetch('FORBIDDEN', 'nope', undefined, 403))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useCategories(), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toMatchObject({ code: 'FORBIDDEN', message: 'nope' })
  })

  it('reports ERR_HTTP when a non-ok response has no error body', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: false, status: 500, json: async () => ({ data: null, error: null }) }),
    )
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useCategories(), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toMatchObject({ code: 'ERR_HTTP', message: 'Request failed: 500' })
  })
})

describe('useCreateCategory', () => {
  it('POSTs the JSON body with the content type header and returns the created node', async () => {
    const fetchMock = okFetch(sampleNode)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useCreateCategory(), { wrapper })
    const created = await result.current.mutateAsync({
      name: 'Science',
      parent_id: 'cat-0',
      track: 'safety',
      sort_order: 3,
    })
    expect(created).toEqual(sampleNode)
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/categories')
    expect(init.method).toBe('POST')
    expect((init.headers as Record<string, string>)['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual({
      name: 'Science',
      parent_id: 'cat-0',
      track: 'safety',
      sort_order: 3,
    })
  })

  it('invalidates the categories query after a successful create', async () => {
    vi.stubGlobal('fetch', okFetch(sampleNode))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useCreateCategory(), { wrapper })
    await result.current.mutateAsync({ name: 'Science', sort_order: 0 })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['categories'] })
  })

  it('rejects with the API error code and details', async () => {
    vi.stubGlobal('fetch', errorFetch('CATEGORY_CYCLE', 'cycle', { path: 'parent_id' }))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useCreateCategory(), { wrapper })
    await expect(result.current.mutateAsync({ name: 'X', sort_order: 0 })).rejects.toMatchObject({
      code: 'CATEGORY_CYCLE',
      message: 'cycle',
      details: { path: 'parent_id' },
    })
    expect(invalidate).not.toHaveBeenCalled()
  })
})

describe('useUpdateCategory', () => {
  it('PUTs the partial body to the category URL and returns the updated node', async () => {
    const updated = { ...sampleNode, name: 'Physics' }
    const fetchMock = okFetch(updated)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useUpdateCategory(), { wrapper })
    const out = await result.current.mutateAsync({ id: 'cat-1', body: { name: 'Physics' } })
    expect(out).toEqual(updated)
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/categories/cat-1')
    expect(init.method).toBe('PUT')
    expect(JSON.parse(init.body as string)).toEqual({ name: 'Physics' })
  })

  it('sends clear_parent when moving a node to the top level', async () => {
    const fetchMock = okFetch(sampleNode)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useUpdateCategory(), { wrapper })
    await result.current.mutateAsync({ id: 'cat-1', body: { clear_parent: true } })
    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect(JSON.parse(init.body as string)).toEqual({ clear_parent: true })
  })

  it('invalidates the categories query after a successful update', async () => {
    vi.stubGlobal('fetch', okFetch(sampleNode))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useUpdateCategory(), { wrapper })
    await result.current.mutateAsync({ id: 'cat-1', body: { sort_order: 2 } })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['categories'] })
  })

  it('rejects with the API error code on failure', async () => {
    vi.stubGlobal('fetch', errorFetch('PARENT_NOT_FOUND', 'gone', undefined, 404))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useUpdateCategory(), { wrapper })
    await expect(
      result.current.mutateAsync({ id: 'cat-1', body: { parent_id: 'missing' } }),
    ).rejects.toMatchObject({ code: 'PARENT_NOT_FOUND' })
  })
})

describe('useDeleteCategory', () => {
  it('DELETEs the category URL and resolves with undefined on 204', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204 })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useDeleteCategory(), { wrapper })
    const out = await result.current.mutateAsync('cat-1')
    expect(out).toBeUndefined()
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/categories/cat-1')
    expect(init.method).toBe('DELETE')
  })

  it('invalidates the categories query after a successful delete', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 204 }))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useDeleteCategory(), { wrapper })
    await result.current.mutateAsync('cat-1')
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['categories'] })
  })

  it('rejects with CATEGORY_IN_USE and its usage count on failure', async () => {
    vi.stubGlobal(
      'fetch',
      errorFetch('CATEGORY_IN_USE', 'in use', { in_use_count: 4 }, 409),
    )
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useDeleteCategory(), { wrapper })
    await expect(result.current.mutateAsync('cat-1')).rejects.toMatchObject({
      code: 'CATEGORY_IN_USE',
      details: { in_use_count: 4 },
    })
    expect(invalidate).not.toHaveBeenCalled()
  })
})

describe('flattenCategories', () => {
  it('returns an empty list for an empty tree', () => {
    expect(flattenCategories([])).toEqual([])
  })

  it('flattens nested nodes depth-first, parents before children', () => {
    const grandchild = makeNode({ id: 'g', name: 'G', parent_id: 'c' })
    const child = makeNode({ id: 'c', name: 'C', parent_id: 'a', children: [grandchild] })
    const sibling = makeNode({ id: 'b', name: 'B' })
    const root = makeNode({ id: 'a', name: 'A', children: [child] })
    expect(flattenCategories([root, sibling]).map((n) => n.id)).toEqual(['a', 'c', 'g', 'b'])
  })

  it('tolerates nodes that have no children array', () => {
    const bare = { ...makeNode({ id: 'x' }), children: undefined } as unknown as CategoryNode
    expect(flattenCategories([bare]).map((n) => n.id)).toEqual(['x'])
  })
})

describe('getDescendantIds', () => {
  it('returns an empty list for a leaf node', () => {
    expect(getDescendantIds(makeNode({ id: 'leaf' }))).toEqual([])
  })

  it('returns every descendant id at any depth, excluding the node itself', () => {
    const grandchild = makeNode({ id: 'g' })
    const child = makeNode({ id: 'c', children: [grandchild] })
    const root = makeNode({ id: 'a', children: [child, makeNode({ id: 'd' })] })
    expect(getDescendantIds(root)).toEqual(['c', 'g', 'd'])
  })

  it('tolerates a node whose children array is missing', () => {
    const bare = { ...makeNode({ id: 'x' }), children: undefined } as unknown as CategoryNode
    expect(getDescendantIds(bare)).toEqual([])
  })
})
