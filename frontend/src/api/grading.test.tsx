import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useGradingQueue, useGradingSession, useSubmitGrade } from './grading'

afterEach(() => vi.unstubAllGlobals())

function makeWrapper(token: string | null = 'tok') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  qc.setQueryData(['auth', 'accessToken'], token)
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  )
  return { qc, wrapper }
}

function okFetch(data: unknown) {
  return vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data, error: null }) })
}

describe('grading api hooks', () => {
  it('useGradingQueue requests the page with auth and optional exam filter', async () => {
    const fetchMock = okFetch({ items: [], meta: { page: 2, per_page: 20, total: 0 } })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useGradingQueue(2, 'exam-1'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/admin/grading?page=2&per_page=20&exam_id=exam-1')
    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer tok')
  })

  it('useGradingQueue omits exam_id and Authorization when absent', async () => {
    const fetchMock = okFetch({ items: [], meta: { page: 1, per_page: 20, total: 0 } })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper(null)
    const { result } = renderHook(() => useGradingQueue(1), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/admin/grading?page=1&per_page=20')
    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect((init.headers as Record<string, string>).Authorization).toBeUndefined()
  })

  it('surfaces API error bodies as errors', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        json: async () => ({ data: null, error: { code: 'FORBIDDEN', message: 'nope' } }),
      }),
    )
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useGradingSession('s1'), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toBeInstanceOf(Error)
  })

  it('throws on non-ok responses without an error body', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: false, status: 500, json: async () => ({ data: null, error: null }) }),
    )
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useGradingSession('s1'), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect((result.current.error as Error).message).toContain('500')
  })

  it('useGradingSession loads session detail', async () => {
    const detail = { session_id: 's1', employee_name: 'A', exam_title: 'E', submitted_at: null, questions: [] }
    const fetchMock = okFetch(detail)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useGradingSession('s1'), { wrapper })
    await waitFor(() => expect(result.current.data).toEqual(detail))
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/admin/grading/s1')
  })

  it('useSubmitGrade posts the grade and invalidates session and queue when all graded', async () => {
    const fetchMock = okFetch({
      question_id: 'q1', grading_status: 'graded', score_pct: 80, session_status: 'graded', all_graded: true,
    })
    vi.stubGlobal('fetch', fetchMock)
    const { qc, wrapper } = makeWrapper()
    const spy = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useSubmitGrade('s1'), { wrapper })
    result.current.mutate({ questionId: 'q1', scorePct: 80, feedback: 'good' })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/admin/grading/s1/answers/q1')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toEqual({ score_pct: 80, feedback: 'good' })
    expect(spy).toHaveBeenCalledWith({ queryKey: ['grading-session', 's1'] })
    expect(spy).toHaveBeenCalledWith({ queryKey: ['grading-queue'] })
  })

  it('useSubmitGrade does not invalidate the queue while answers remain', async () => {
    vi.stubGlobal(
      'fetch',
      okFetch({ question_id: 'q1', grading_status: 'graded', score_pct: 50, session_status: 'submitted', all_graded: false }),
    )
    const { qc, wrapper } = makeWrapper()
    const spy = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useSubmitGrade('s1'), { wrapper })
    result.current.mutate({ questionId: 'q1', scorePct: 50, feedback: '' })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(spy).not.toHaveBeenCalledWith({ queryKey: ['grading-queue'] })
  })
})
