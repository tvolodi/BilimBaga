import { describe, it, expect, vi, beforeAll, afterAll, afterEach } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import { useReactivateUser } from './users'

// FR-BB18 AC-13: client hook for POST /api/v1/users/{id}/reactivate.
const calls: { method: string; path: string }[] = []

const server = setupServer(
  http.post('/api/v1/users/:id/reactivate', ({ request }) => {
    calls.push({ method: request.method, path: new URL(request.url).pathname })
    return HttpResponse.json({ data: {}, error: null })
  }),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => {
  server.resetHandlers()
  calls.length = 0
})
afterAll(() => server.close())

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const Wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  )
  return { qc, wrapper: Wrapper }
}

describe('useReactivateUser', () => {
  it('POSTs to the reactivate endpoint of the given user', async () => {
    const { wrapper } = setup()
    const { result } = renderHook(() => useReactivateUser('u-7'), { wrapper })

    await act(async () => {
      await result.current.mutateAsync()
    })

    expect(calls).toEqual([{ method: 'POST', path: '/api/v1/users/u-7/reactivate' }])
  })

  it('invalidates the users queries after a successful reactivation', async () => {
    const { qc, wrapper } = setup()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useReactivateUser('u-7'), { wrapper })

    await act(async () => {
      await result.current.mutateAsync()
    })

    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['users'] })
  })

  it('rejects with the error message when the server refuses the request', async () => {
    server.use(
      http.post('/api/v1/users/:id/reactivate', () =>
        HttpResponse.json(
          { data: null, error: { code: 'USER_ALREADY_ACTIVE', message: 'user is already active' } },
          { status: 409 },
        ),
      ),
    )
    const { wrapper } = setup()
    const { result } = renderHook(() => useReactivateUser('u-7'), { wrapper })

    await act(async () => {
      await expect(result.current.mutateAsync()).rejects.toThrow('user is already active')
    })
  })
})
