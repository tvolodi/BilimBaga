import { renderHook, act, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { describe, it, expect, beforeAll, afterEach, afterAll } from 'vitest'
import { type ReactNode } from 'react'
import { useLogin, useChangePassword, useLogout } from './auth'

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

describe('useChangePassword (ISS-171)', () => {
  it('stores the access token returned by change-password in the auth cache', async () => {
    server.use(
      http.post('/api/v1/auth/change-password', ({ request }) => {
        expect(request.headers.get('Authorization')).toBe('Bearer old-tok')
        return HttpResponse.json({
          data: { message: 'password changed', access_token: 'new-tok', token_type: 'Bearer', expires_in: 900 },
          error: null,
        })
      }),
    )
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    queryClient.setQueryData(['auth', 'accessToken'], 'old-tok')
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )
    const { result } = renderHook(() => useChangePassword(), { wrapper })
    act(() => {
      result.current.mutate({ current_password: 'Old1aaaa', new_password: 'New1aaaa' })
    })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(queryClient.getQueryData(['auth', 'accessToken'])).toBe('new-tok')
  })

  it('keeps the old token and surfaces the error when the change fails', async () => {
    server.use(
      http.post('/api/v1/auth/change-password', () =>
        HttpResponse.json(
          { data: null, error: { code: 'INVALID_CREDENTIALS', message: 'current password is incorrect' } },
          { status: 400 },
        ),
      ),
    )
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    queryClient.setQueryData(['auth', 'accessToken'], 'old-tok')
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )
    const { result } = renderHook(() => useChangePassword(), { wrapper })
    act(() => {
      result.current.mutate({ current_password: 'bad', new_password: 'New1aaaa' })
    })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.code).toBe('INVALID_CREDENTIALS')
    expect(queryClient.getQueryData(['auth', 'accessToken'])).toBe('old-tok')
  })
})

// FR-BB116 AC-7: a previous session's cached profile (and its preferred_locale) must not outlive
// the session, otherwise the next user's sync would apply the previous user's language.
describe('session change drops the cached profile (FR-BB116)', () => {
  function clientWithStaleProfile() {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    })
    queryClient.setQueryData(['users', 'me'], { id: 'user-a', preferred_locale: 'ru' })
    queryClient.setQueryData(['users', 'roles', 'user-a'], [])
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )
    return { queryClient, wrapper }
  }

  it('useLogin removes the previous user profile when the next user signs in', async () => {
    server.use(
      http.post('/api/v1/auth/login', () =>
        HttpResponse.json({
          data: { access_token: 'tok-b', token_type: 'Bearer', expires_in: 3600, user: MOCK_USER },
          error: null,
        }),
      ),
    )
    const { queryClient, wrapper } = clientWithStaleProfile()
    const { result } = renderHook(() => useLogin(), { wrapper })

    act(() => {
      result.current.mutate({ email: 'test@example.com', password: 'secret' })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(queryClient.getQueryData(['users', 'me'])).toBeUndefined()
    expect(queryClient.getQueryData(['users', 'roles', 'user-a'])).toBeUndefined()
  })

  it('useLogout removes the profile of the signed-out user', async () => {
    server.use(http.post('/api/v1/auth/logout', () => new HttpResponse(null, { status: 204 })))
    const { queryClient, wrapper } = clientWithStaleProfile()
    const { result } = renderHook(() => useLogout(), { wrapper })

    act(() => {
      result.current.mutate()
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(queryClient.getQueryData(['users', 'me'])).toBeUndefined()
  })
})
