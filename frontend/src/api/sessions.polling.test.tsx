import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement, type ReactNode } from 'react'
import { useSessionResult, RESULT_POLL_INTERVAL_MS, RESULT_POLL_LIMIT_MS } from './sessions'

// FR-BB322 AC-6: the result of a waiting session is polled every 10 s, stops on a final status, stops after 15 min,
// and Refresh starts the 15 minutes again. Tests use fake timers, so no real time passes.

let status = 'grading_pending'
let fetchMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  status = 'grading_pending'
  vi.useFakeTimers()
  fetchMock = vi.fn().mockImplementation(async () => ({
    ok: true,
    status: 200,
    json: async () => ({ data: { session_id: 's-1', exam_id: 'e-1', status }, error: null }),
  }))
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

function wrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return ({ children }: { children: ReactNode }) => createElement(QueryClientProvider, { client: qc }, children)
}

async function advance(ms: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms)
  })
}

describe('useSessionResult polling (FR-BB322 AC-6)', () => {
  it('polls every 10 seconds while the result waits for grading', async () => {
    renderHook(() => useSessionResult('s-1'), { wrapper: wrapper() })
    await advance(0)
    expect(fetchMock).toHaveBeenCalledTimes(1)

    await advance(RESULT_POLL_INTERVAL_MS)
    expect(fetchMock).toHaveBeenCalledTimes(2)

    await advance(RESULT_POLL_INTERVAL_MS)
    expect(fetchMock).toHaveBeenCalledTimes(3)
  })

  it('stops polling as soon as the result is final', async () => {
    const { result } = renderHook(() => useSessionResult('s-1'), { wrapper: wrapper() })
    await advance(0)
    status = 'submitted'
    await advance(RESULT_POLL_INTERVAL_MS)
    const afterFinal = fetchMock.mock.calls.length

    await advance(RESULT_POLL_INTERVAL_MS * 3)
    expect(fetchMock.mock.calls.length).toBe(afterFinal)
    expect(result.current.data?.status).toBe('submitted')
    expect(result.current.pollingStopped).toBe(false)
  })

  it('stops after 15 minutes and shows that polling stopped', async () => {
    const { result } = renderHook(() => useSessionResult('s-1'), { wrapper: wrapper() })
    await advance(RESULT_POLL_LIMIT_MS + RESULT_POLL_INTERVAL_MS)
    expect(result.current.pollingStopped).toBe(true)

    const stoppedAt = fetchMock.mock.calls.length
    await advance(RESULT_POLL_INTERVAL_MS * 6)
    expect(fetchMock.mock.calls.length).toBe(stoppedAt)
  })

  it('Refresh fetches once and re-arms the 15 minutes', async () => {
    const { result } = renderHook(() => useSessionResult('s-1'), { wrapper: wrapper() })
    await advance(RESULT_POLL_LIMIT_MS + RESULT_POLL_INTERVAL_MS)
    expect(result.current.pollingStopped).toBe(true)
    const beforeRefresh = fetchMock.mock.calls.length

    await act(async () => {
      await result.current.refresh()
    })
    expect(fetchMock.mock.calls.length).toBe(beforeRefresh + 1)
    expect(result.current.pollingStopped).toBe(false)

    await advance(RESULT_POLL_INTERVAL_MS)
    expect(fetchMock.mock.calls.length).toBe(beforeRefresh + 2)
  })

  it('does not poll a final result at all', async () => {
    status = 'submitted'
    renderHook(() => useSessionResult('s-1'), { wrapper: wrapper() })
    await advance(0)
    await advance(RESULT_POLL_INTERVAL_MS * 3)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })
})
