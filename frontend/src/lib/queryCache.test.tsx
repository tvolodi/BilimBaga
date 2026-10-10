import { describe, it, expect, vi, afterEach } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider, type QueryClient as Client } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useLogin, useLogout } from '@/api/auth'
import { endRevokedSession } from './sessionRevoked'
import { clearNonPublicQueries } from './queryCache'

// #436: every per-user query family is cleared at session end, at logout and at login. The public
// families (branding, certificate check) stay, and so does the auth state, which the session writes itself.

const PER_USER_FAMILIES = [
  'exams', 'questions', 'users', 'categories', 'tags', 'departments', 'portal', 'grading-queue',
  'grading-session', 'roles', 'session-next-question', 'session-result', 'dashboard', 'my-results',
  'exams-list-reports', 'exam-history', 'exam-analytics', 'employee-record', 'employee-progress',
  'audit-log', 'ai-insights',
]
const PUBLIC_FAMILIES = ['tenant', 'verify']

function seedEveryFamily(qc: Client) {
  for (const family of [...PER_USER_FAMILIES, ...PUBLIC_FAMILIES]) {
    qc.setQueryData([family, 'seed'], { family })
  }
  qc.setQueryData(['users', 'me'], { id: 'old-user', preferred_locale: 'ru' })
  qc.setQueryData(['auth', 'accessToken'], 'old-token')
  qc.setQueryData(['auth', 'currentUser'], { id: 'old-user' })
}

function cachedFamilies(qc: Client): Set<unknown> {
  return new Set(qc.getQueryCache().findAll().map((query) => query.queryKey[0]))
}

function expectPerUserCleared(qc: Client) {
  const cached = cachedFamilies(qc)
  for (const family of PER_USER_FAMILIES) {
    expect(cached.has(family), `${family} survived`).toBe(false)
  }
}

function expectPublicKept(qc: Client) {
  const cached = cachedFamilies(qc)
  for (const family of PUBLIC_FAMILIES) {
    expect(cached.has(family), `${family} was cleared`).toBe(true)
  }
}

function clientWith(): Client {
  return new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
}

function wrapperFor(qc: Client) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('clearNonPublicQueries (#436)', () => {
  it('removes every per-user family and keeps the public families and the auth state', () => {
    const qc = clientWith()
    seedEveryFamily(qc)

    clearNonPublicQueries(qc)

    expectPerUserCleared(qc)
    expectPublicKept(qc)
    expect(qc.getQueryData(['auth', 'accessToken'])).toBe('old-token')
  })
})

describe('session end (#436)', () => {
  it('clears every per-user family and raises the notice', () => {
    const qc = clientWith()
    seedEveryFamily(qc)

    endRevokedSession(qc)

    expectPerUserCleared(qc)
    expectPublicKept(qc)
    expect(qc.getQueryData(['auth', 'accessToken'])).toBeNull()
    expect(qc.getQueryData(['auth', 'sessionRevoked'])).toBe(true)
  })
})

describe('logout (#436)', () => {
  it('clears every per-user family', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => ({ ok: true, status: 200, json: async () => ({ data: null, error: null }) })))
    const qc = clientWith()
    seedEveryFamily(qc)
    const { result } = renderHook(() => useLogout(), { wrapper: wrapperFor(qc) })

    await act(async () => {
      await result.current.mutateAsync()
    })

    expectPerUserCleared(qc)
    expectPublicKept(qc)
    expect(qc.getQueryData(['auth', 'currentUser'])).toBeNull()
  })
})

describe('login (#436)', () => {
  it('clears every per-user family before storing the new user and token', async () => {
    const user = { id: 'new-user', email: 'new@example.com', full_name: 'New', role_name: 'employee', force_password_change: false }
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        ok: true,
        status: 200,
        json: async () => ({ data: { access_token: 'new-token', user }, error: null }),
      })),
    )
    const qc = clientWith()
    seedEveryFamily(qc)
    const { result } = renderHook(() => useLogin(), { wrapper: wrapperFor(qc) })

    await act(async () => {
      await result.current.mutateAsync({ email: 'new@example.com', password: 'secret' })
    })

    expectPerUserCleared(qc)
    expectPublicKept(qc)
    expect(qc.getQueryData(['users', 'me'])).toBeUndefined()
    expect(qc.getQueryData(['auth', 'accessToken'])).toBe('new-token')
    expect(qc.getQueryData(['auth', 'currentUser'])).toEqual(user)
  })
})
