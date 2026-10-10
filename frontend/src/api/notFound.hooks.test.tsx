import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useExam } from './exams'
import { useExamAnalytics } from './analytics'
import { useExamHistory } from './sessions'
import { isNotFoundError } from '@/lib/apiRetry'

// #487: the API answers an unknown exam with HTTP 404 and the code ERR_NOT_FOUND. Each exam hook must hand the page
// an error that isNotFoundError recognises, so the page shows the not-found state. These tests use the real hooks and
// the real request path (fetch is stubbed with the real response), not hand-built error objects.

const notFoundFetch = vi.fn(async () => ({
  ok: false,
  status: 404,
  json: async () => ({ data: null, error: { code: 'ERR_NOT_FOUND', message: 'exam not found' } }),
}))

function renderWithClient<T>(hook: () => T) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['auth', 'accessToken'], 'token-1')
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  )
  return renderHook(hook, { wrapper })
}

afterEach(() => {
  notFoundFetch.mockClear()
  vi.unstubAllGlobals()
})

describe('exam hooks surface a not-found answer as a not-found error (#487)', () => {
  it('useExam (the wizard edit page) recognises ERR_NOT_FOUND', async () => {
    vi.stubGlobal('fetch', notFoundFetch)
    const { result } = renderWithClient(() => useExam('missing-exam'))
    // The hook retries anything that is not a not-found answer; on a broken check that takes seconds.
    await waitFor(() => expect(result.current.isError).toBe(true), { timeout: 15_000 })
    expect(isNotFoundError(result.current.error)).toBe(true)
  })

  it('useExamAnalytics (the analytics page) recognises ERR_NOT_FOUND', async () => {
    vi.stubGlobal('fetch', notFoundFetch)
    const { result } = renderWithClient(() => useExamAnalytics('missing-exam'))
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(isNotFoundError(result.current.error)).toBe(true)
  })

  it('useExamHistory recognises ERR_NOT_FOUND', async () => {
    vi.stubGlobal('fetch', notFoundFetch)
    const { result } = renderWithClient(() => useExamHistory('missing-exam'))
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(isNotFoundError(result.current.error)).toBe(true)
  })
})
