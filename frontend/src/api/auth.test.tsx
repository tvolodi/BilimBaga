import { renderHook, act, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { describe, it, expect, beforeAll, afterEach, afterAll } from 'vitest'
import { type ReactNode } from 'react'
import { useLogin } from './auth'

const server = setupServer()

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )
}

const MOCK_USER = {
  id: 'user-1',
  full_name: 'Test User',
  email: 'test@example.com',
  role: 'super_admin',
  force_password_change: false,
}

describe('useLogin', () => {
  it('stores access token and user on success', async () => {
    server.use(
      http.post('/api/v1/auth/login', () =>
        HttpResponse.json({
          data: {
            access_token: 'tok-123',
            token_type: 'Bearer',
            expires_in: 3600,
            user: MOCK_USER,
          },
          error: null,
        }),
      ),
    )

    const wrapper = createWrapper()
    const { result } = renderHook(() => useLogin(), { wrapper })

    act(() => {
      result.current.mutate({ email: 'test@example.com', password: 'secret' })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.access_token).toBe('tok-123')
    expect(result.current.data?.user.role).toBe('super_admin')
  })

  it('throws ApiError with code INVALID_CREDENTIALS on 401', async () => {
    server.use(
      http.post('/api/v1/auth/login', () =>
        HttpResponse.json(
          { data: null, error: { code: 'INVALID_CREDENTIALS', message: 'invalid email or password' } },
          { status: 401 },
        ),
      ),
    )

    const wrapper = createWrapper()
    const { result } = renderHook(() => useLogin(), { wrapper })

    await act(async () => {
      try {
        await result.current.mutateAsync({ email: 'bad@example.com', password: 'wrong' })
      } catch {
        // expected
      }
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.code).toBe('INVALID_CREDENTIALS')
  })

  it('throws ApiError with code ACCOUNT_LOCKED on 423', async () => {
    server.use(
      http.post('/api/v1/auth/login', () =>
        HttpResponse.json(
          {
            data: null,
            error: {
              code: 'ACCOUNT_LOCKED',
              message: 'account locked, try again after 2099-01-01T12:00:00Z',
            },
          },
          { status: 423 },
        ),
      ),
    )

    const wrapper = createWrapper()
    const { result } = renderHook(() => useLogin(), { wrapper })

    await act(async () => {
      try {
        await result.current.mutateAsync({ email: 'locked@example.com', password: 'pass' })
      } catch {
        // expected
      }
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.code).toBe('ACCOUNT_LOCKED')
  })

  it('throws on network error (fetch failure)', async () => {
    server.use(
      http.post('/api/v1/auth/login', () => HttpResponse.error()),
    )

    const wrapper = createWrapper()
    const { result } = renderHook(() => useLogin(), { wrapper })

    await act(async () => {
      try {
        await result.current.mutateAsync({ email: 'test@example.com', password: 'pass' })
      } catch {
        // expected
      }
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})
