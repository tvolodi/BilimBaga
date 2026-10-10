import { describe, it, expect } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider, type QueryClient as QC } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useAccessToken } from './useAccessToken'

function wrapperFor(qc: QC) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  }
}

describe('useAccessToken (ISS-249)', () => {
  it('returns the token held under ["auth", "accessToken"]', () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    qc.setQueryData(['auth', 'accessToken'], 'tok-1')
    const { result } = renderHook(() => useAccessToken(), { wrapper: wrapperFor(qc) })
    expect(result.current.token).toBe('tok-1')
    expect(result.current.status).toBe('success')
  })

  it('re-renders with a null token when the token is cleared after mount', async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    qc.setQueryData(['auth', 'accessToken'], 'tok-1')
    const { result } = renderHook(() => useAccessToken(), { wrapper: wrapperFor(qc) })

    act(() => qc.setQueryData(['auth', 'accessToken'], null))

    await waitFor(() => expect(result.current.token).toBeNull())
  })

  it('stays pending and never fetches when the token was never set', () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { result } = renderHook(() => useAccessToken(), { wrapper: wrapperFor(qc) })
    expect(result.current.token).toBeUndefined()
    expect(result.current.status).toBe('pending')
    expect(qc.getQueryState(['auth', 'accessToken'])?.fetchStatus).toBe('idle')
  })
})
