import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useGenerateQuestions } from './ai'

afterEach(() => vi.restoreAllMocks())

function setup(token: string | null) {
  const qc = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
  qc.setQueryData(['auth', 'accessToken'], token)
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  )
  return renderHook(() => useGenerateQuestions(), { wrapper })
}

describe('useGenerateQuestions', () => {
  it('sends the Bearer Authorization header from the auth cache', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: { questions: [] }, error: null }),
    })
    vi.stubGlobal('fetch', fetchMock)
    const { result } = setup('tok-123')
    result.current.mutate({ category_id: 'c', difficulty: 'easy', count: 1 })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer tok-123')
  })

  it('omits Authorization when no token is cached', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: { questions: [] }, error: null }),
    })
    vi.stubGlobal('fetch', fetchMock)
    const { result } = setup(null)
    result.current.mutate({ category_id: 'c', difficulty: 'easy', count: 1 })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect((init.headers as Record<string, string>).Authorization).toBeUndefined()
  })
})
