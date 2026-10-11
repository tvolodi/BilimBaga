import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement, type ReactNode } from 'react'
import {
  roleErrorKey,
  ROLE_NAME_PATTERN,
  NON_ASSIGNABLE_PERMISSIONS,
  useRolesAdmin,
  usePermissionsCatalogue,
  useCreateRole,
  useUpdateRole,
  useDeleteRole,
} from './roles'

afterEach(() => vi.unstubAllGlobals())

describe('roleErrorKey', () => {
  it.each(['ROLE_NAME_TAKEN', 'ROLE_SYSTEM_IMMUTABLE', 'VALIDATION_ERROR', 'NOT_FOUND', 'FORBIDDEN'])(
    'maps %s to its own key',
    (code) => {
      expect(roleErrorKey({ code, message: 'x' })).toEqual({ key: `roles.errors.${code}` })
    },
  )

  it('extracts the user count for ROLE_IN_USE', () => {
    expect(roleErrorKey({ code: 'ROLE_IN_USE', message: 'role is assigned to 99 user(s)', details: { count: 12 } })).toEqual({
      key: 'roles.errors.ROLE_IN_USE',
      values: { count: 12 },
    })
  })

  it('defaults ROLE_IN_USE count to 0 when details are missing', () => {
    expect(roleErrorKey({ code: 'ROLE_IN_USE', message: 'assigned to 5 users' }).values).toEqual({ count: 0 })
  })

  it('falls back to a generic key for unknown codes and non-errors', () => {
    expect(roleErrorKey({ code: 'WAT', message: 'x' }).key).toBe('roles.errors.generic')
    expect(roleErrorKey(new Error('network')).key).toBe('roles.errors.generic')
    expect(roleErrorKey(null).key).toBe('roles.errors.generic')
  })
})

describe('role constants mirror backend rules', () => {
  it('name pattern', () => {
    expect(ROLE_NAME_PATTERN.test('qa_reviewer')).toBe(true)
    expect(ROLE_NAME_PATTERN.test('QA Reviewer')).toBe(false)
    expect(ROLE_NAME_PATTERN.test('ab')).toBe(false)
    expect(ROLE_NAME_PATTERN.test('1abc')).toBe(false)
  })
  it('non-assignable permissions', () => {
    expect([...NON_ASSIGNABLE_PERMISSIONS].sort()).toEqual(['roles:manage', 'roles:read', 'tenant:manage'])
  })
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

function noContentFetch() {
  return vi.fn().mockResolvedValue({ ok: true, status: 204 })
}

function firstCall(fetchMock: ReturnType<typeof vi.fn>) {
  const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
  return { url, init, headers: (init?.headers ?? {}) as Record<string, string> }
}

describe('role queries', () => {
  it('useRolesAdmin lists the roles', async () => {
    const fetchMock = okFetch([{ id: 'r-1', name: 'qa_reviewer' }])
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()

    const { result } = renderHook(() => useRolesAdmin(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(firstCall(fetchMock).url).toBe('/api/v1/roles')
    expect(result.current.data).toEqual([{ id: 'r-1', name: 'qa_reviewer' }])
  })

  it('usePermissionsCatalogue lists the permission catalogue', async () => {
    const fetchMock = okFetch([{ id: 'p-1', resource: 'exams', action: 'read' }])
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()

    const { result } = renderHook(() => usePermissionsCatalogue(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(firstCall(fetchMock).url).toBe('/api/v1/roles/permissions')
  })
})

describe('role mutations', () => {
  it('useCreateRole posts the role and refreshes both role caches', async () => {
    const fetchMock = okFetch({ id: 'r-2', name: 'auditor' })
    vi.stubGlobal('fetch', fetchMock)
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')

    const { result } = renderHook(() => useCreateRole(), { wrapper })
    const body = { name: 'auditor', description: 'reads audit log', permissions: ['p-1'] }
    await act(async () => {
      await result.current.mutateAsync(body)
    })

    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/roles')
    expect(init.method).toBe('POST')
    expect(headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual(body)
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['roles'] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['users', 'roles'] })
  })

  it('useUpdateRole puts the description and permissions to the role path, without the id', async () => {
    const fetchMock = okFetch({ id: 'r-2', name: 'auditor' })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()

    const { result } = renderHook(() => useUpdateRole(), { wrapper })
    await act(async () => {
      await result.current.mutateAsync({ id: 'r-2', description: 'new text', permissions: [] })
    })

    const { url, init } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/roles/r-2')
    expect(init.method).toBe('PUT')
    expect(JSON.parse(init.body as string)).toEqual({ description: 'new text', permissions: [] })
  })

  it('useDeleteRole deletes the role and accepts a 204 response', async () => {
    const fetchMock = noContentFetch()
    vi.stubGlobal('fetch', fetchMock)
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')

    const { result } = renderHook(() => useDeleteRole(), { wrapper })
    await act(async () => {
      await result.current.mutateAsync('r-2')
    })

    const { url, init } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/roles/r-2')
    expect(init.method).toBe('DELETE')
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['roles'] })
  })
})
