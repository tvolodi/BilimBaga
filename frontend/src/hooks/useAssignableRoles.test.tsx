import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import type { ReactNode } from 'react'
import { useRoles, rolesQueryKey } from '@/api/users'

let calls = 0
const server = setupServer(
  http.get('/api/v1/users/roles', () => {
    calls++
    return HttpResponse.json({ data: [], error: null })
  }),
)
beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => { server.resetHandlers(); calls = 0 })
afterAll(() => server.close())

function setup(claims: Record<string, unknown> | null) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  if (claims) qc.setQueryData(['auth', 'accessToken'], `h.${btoa(JSON.stringify(claims))}.s`)
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  )
  return { qc, ...renderHook(() => useRoles(), { wrapper }) }
}

describe('useRoles query key', () => {
  it('is scoped to the current user id', async () => {
    const { qc, result } = setup({ role: 'super_admin', sub: 'user-a' })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(qc.getQueryCache().find({ queryKey: rolesQueryKey('user-a') })).toBeDefined()
    expect(qc.getQueryCache().find({ queryKey: ['users', 'roles'], exact: true })).toBeUndefined()
    // prefix invalidation still matches
    expect(qc.getQueryCache().findAll({ queryKey: ['users', 'roles'] })).toHaveLength(1)
  })

  it('stays disabled without a user id', async () => {
    const { result } = setup({ role: 'super_admin' })
    await new Promise((r) => setTimeout(r, 50))
    expect(result.current.fetchStatus).toBe('idle')
    expect(calls).toBe(0)
  })
})
