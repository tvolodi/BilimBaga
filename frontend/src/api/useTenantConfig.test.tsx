import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import { useTenantConfig, type TenantConfig } from './useTenantConfig'

const mockConfig: TenantConfig = {
  app_name: 'BilimBaga',
  primary_color: '#0ea5e9',
  accent_color: '#f59e0b',
  default_locale: 'kk',
  available_locales: ['kk', 'ru', 'en'],
}

const server = setupServer(
  http.get('/api/v1/tenant/config', () => {
    return HttpResponse.json({ data: mockConfig, error: null })
  }),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )
}

describe('useTenantConfig', () => {
  it('fetches and returns tenant config data', async () => {
    const { result } = renderHook(() => useTenantConfig(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data).toEqual(mockConfig)
  })

  it('exposes all five public config keys', async () => {
    const { result } = renderHook(() => useTenantConfig(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const data = result.current.data!
    expect(data.app_name).toBe('BilimBaga')
    expect(data.primary_color).toBe('#0ea5e9')
    expect(data.accent_color).toBe('#f59e0b')
    expect(data.default_locale).toBe('kk')
    expect(data.available_locales).toEqual(['kk', 'ru', 'en'])
  })

  it('enters error state when API returns an error', async () => {
    server.use(
      http.get('/api/v1/tenant/config', () => {
        return HttpResponse.json(
          { data: null, error: { code: 'INTERNAL_ERROR', message: 'server error' } },
          { status: 200 },
        )
      }),
    )

    const { result } = renderHook(() => useTenantConfig(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe('server error')
  })
})
