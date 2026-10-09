import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import {
  useDepartments,
  useCreateDepartment,
  useUpdateDepartment,
  useDeleteDepartment,
  findDepartmentById,
  type Department,
} from './departments'

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

const makeDept = (overrides: Partial<Department>): Department => ({
  id: 'dept-default',
  name: 'Default',
  parent_id: null,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  children: [],
  ...overrides,
})

const sampleDept = makeDept({ id: 'dept-1', name: 'Engineering' })

describe('useDepartments', () => {
  it('GETs the department tree with the bearer token and returns the data', async () => {
    const fetchMock = okFetch([sampleDept])
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useDepartments(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual([sampleDept])
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/departments')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer tok')
  })

  it('omits the Authorization header when no token is cached', async () => {
    const fetchMock = okFetch([])
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper(null)
    const { result } = renderHook(() => useDepartments(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect((init.headers as Record<string, string>).Authorization).toBeUndefined()
  })

  it('surfaces an API error body with its code', async () => {
    vi.stubGlobal('fetch', errorFetch('FORBIDDEN', 'nope', undefined, 403))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useDepartments(), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toMatchObject({ code: 'FORBIDDEN', message: 'nope' })
  })

  it('reports ERR_HTTP when a non-ok response has no error body', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: false, status: 500, json: async () => ({ data: null, error: null }) }),
    )
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useDepartments(), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toMatchObject({ code: 'ERR_HTTP', message: 'Request failed: 500' })
  })
})

describe('useCreateDepartment', () => {
  it('POSTs the JSON body with the content type header and returns the created department', async () => {
    const created = makeDept({ id: 'dept-9', name: 'Sales', parent_id: 'dept-1' })
    const fetchMock = okFetch(created)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useCreateDepartment(), { wrapper })
    const out = await result.current.mutateAsync({ name: 'Sales', parent_id: 'dept-1' })
    expect(out).toEqual(created)
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/departments')
    expect(init.method).toBe('POST')
    expect((init.headers as Record<string, string>)['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual({ name: 'Sales', parent_id: 'dept-1' })
  })

  it('sends a top-level create with only the name when no parent is given', async () => {
    const fetchMock = okFetch(sampleDept)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useCreateDepartment(), { wrapper })
    await result.current.mutateAsync({ name: 'Engineering' })
    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect(JSON.parse(init.body as string)).toEqual({ name: 'Engineering' })
  })

  it('invalidates the departments query after a successful create', async () => {
    vi.stubGlobal('fetch', okFetch(sampleDept))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useCreateDepartment(), { wrapper })
    await result.current.mutateAsync({ name: 'Engineering' })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['departments'] })
  })

  it('rejects with the API error code and message and does not invalidate', async () => {
    vi.stubGlobal('fetch', errorFetch('DUPLICATE_NAME', 'duplicate', undefined, 409))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useCreateDepartment(), { wrapper })
    await expect(result.current.mutateAsync({ name: 'Engineering' })).rejects.toMatchObject({
      code: 'DUPLICATE_NAME',
      message: 'duplicate',
    })
    expect(invalidate).not.toHaveBeenCalled()
  })
})

describe('useUpdateDepartment', () => {
  it('PUTs the new name to the department URL and returns the updated department', async () => {
    const updated = { ...sampleDept, name: 'Platform' }
    const fetchMock = okFetch(updated)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useUpdateDepartment(), { wrapper })
    const out = await result.current.mutateAsync({ id: 'dept-1', name: 'Platform' })
    expect(out).toEqual(updated)
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/departments/dept-1')
    expect(init.method).toBe('PUT')
    expect((init.headers as Record<string, string>)['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual({ name: 'Platform' })
  })

  it('invalidates the departments query after a successful update', async () => {
    vi.stubGlobal('fetch', okFetch(sampleDept))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useUpdateDepartment(), { wrapper })
    await result.current.mutateAsync({ id: 'dept-1', name: 'Engineering' })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['departments'] })
  })

  it('rejects with the API error code on failure and does not invalidate', async () => {
    vi.stubGlobal('fetch', errorFetch('NOT_FOUND', 'gone', undefined, 404))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useUpdateDepartment(), { wrapper })
    await expect(result.current.mutateAsync({ id: 'dept-1', name: 'X' })).rejects.toMatchObject({
      code: 'NOT_FOUND',
    })
    expect(invalidate).not.toHaveBeenCalled()
  })
})

describe('useDeleteDepartment', () => {
  it('DELETEs the department URL and resolves with undefined on 204', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204 })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useDeleteDepartment(), { wrapper })
    const out = await result.current.mutateAsync('dept-1')
    expect(out).toBeUndefined()
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/departments/dept-1')
    expect(init.method).toBe('DELETE')
  })

  it('invalidates the departments query after a successful delete', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 204 }))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useDeleteDepartment(), { wrapper })
    await result.current.mutateAsync('dept-1')
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['departments'] })
  })

  it('rejects with the API error code when the department is not empty', async () => {
    vi.stubGlobal('fetch', errorFetch('DEPARTMENT_NOT_EMPTY', 'has users', { count: 3 }, 409))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useDeleteDepartment(), { wrapper })
    await expect(result.current.mutateAsync('dept-1')).rejects.toMatchObject({
      code: 'DEPARTMENT_NOT_EMPTY',
      message: 'has users',
      details: { count: 3 },
    })
    expect(invalidate).not.toHaveBeenCalled()
  })
})

describe('findDepartmentById', () => {
  const child = makeDept({ id: 'dept-2', name: 'Frontend', parent_id: 'dept-1' })
  const grandchild = makeDept({ id: 'dept-3', name: 'Web', parent_id: 'dept-2' })
  const tree = [makeDept({ id: 'dept-1', name: 'Engineering', children: [{ ...child, children: [grandchild] }] })]

  it('finds a top-level department', () => {
    expect(findDepartmentById(tree, 'dept-1')?.name).toBe('Engineering')
  })

  it('finds a nested child and a deeper grandchild', () => {
    expect(findDepartmentById(tree, 'dept-2')?.name).toBe('Frontend')
    expect(findDepartmentById(tree, 'dept-3')?.name).toBe('Web')
  })

  it('returns undefined when the id is not in the tree', () => {
    expect(findDepartmentById(tree, 'missing')).toBeUndefined()
  })

  it('returns undefined for an empty tree', () => {
    expect(findDepartmentById([], 'dept-1')).toBeUndefined()
  })

  it('tolerates a node whose children field is absent', () => {
    const sparse = [{ id: 'dept-x', name: 'Sparse' } as unknown as Department]
    expect(findDepartmentById(sparse, 'dept-x')?.name).toBe('Sparse')
    expect(findDepartmentById(sparse, 'dept-y')).toBeUndefined()
  })
})
