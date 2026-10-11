import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement, type ReactNode } from 'react'
import {
  useSession,
  useSaveAnswer,
  useSubmitSession,
  useReportEvent,
  useSessionResult,
  useNextAdaptiveQuestion,
  useMyResults,
  useExamHistory,
} from './sessions'

afterEach(() => vi.unstubAllGlobals())

function makeWrapper(token: string | null = 'tok') {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  qc.setQueryData(['auth', 'accessToken'], token)
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children)
  return { qc, wrapper }
}

function okFetch(data: unknown) {
  return vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => ({ data, error: null }) })
}

function errorFetch(error: { code: string; message: string }, status = 404) {
  return vi.fn().mockResolvedValue({ ok: false, status, json: async () => ({ data: null, error }) })
}

function firstCall(fetchMock: ReturnType<typeof vi.fn>) {
  const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
  return { url, init, headers: (init?.headers ?? {}) as Record<string, string> }
}

describe('session queries', () => {
  it('useSession loads the resume payload with the bearer token', async () => {
    const fetchMock = okFetch({ session_id: 's-1', status: 'in_progress' })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper('tok-1')

    const { result } = renderHook(() => useSession('s-1'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const { url, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/portal/sessions/s-1')
    expect(headers.Authorization).toBe('Bearer tok-1')
    expect(result.current.data).toEqual({ session_id: 's-1', status: 'in_progress' })
  })

  it('useSession surfaces the envelope error code and status', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'NOT_FOUND', message: 'no such session' }, 404))
    const { wrapper } = makeWrapper()

    const { result } = renderHook(() => useSession('missing'), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(result.current.error).toMatchObject({ message: 'no such session', code: 'NOT_FOUND', status: 404 })
  })

  it('useSessionResult loads the result of one session', async () => {
    const fetchMock = okFetch({ session_id: 's-2', passed: true })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()

    const { result } = renderHook(() => useSessionResult('s-2'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(firstCall(fetchMock).url).toBe('/api/v1/portal/sessions/s-2/result')
    expect(result.current.data).toEqual({ session_id: 's-2', passed: true })
  })

  it('useNextAdaptiveQuestion does not fetch while disabled', () => {
    const fetchMock = okFetch({ question: null, questions_answered: 0, done: true })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()

    renderHook(() => useNextAdaptiveQuestion('s-3', false), { wrapper })

    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('useNextAdaptiveQuestion fetches the next question when enabled', async () => {
    const fetchMock = okFetch({ question: null, questions_answered: 4, done: true })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()

    const { result } = renderHook(() => useNextAdaptiveQuestion('s-3', true), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(firstCall(fetchMock).url).toBe('/api/v1/portal/sessions/s-3/next-question')
    expect(result.current.data?.done).toBe(true)
  })

  it('useMyResults requests the page with the sort and direction', async () => {
    const fetchMock = okFetch({ sessions: [], meta: { page: 2, per_page: 20, total: 0 } })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()

    const { result } = renderHook(() => useMyResults(2, 'score', 'asc'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(firstCall(fetchMock).url).toBe('/api/v1/portal/results?page=2&sort=score&dir=asc&per_page=20')
  })

  it('useExamHistory asks for the latest session of one exam', async () => {
    const fetchMock = okFetch({ exam_id: 'e-1', exam_title: 'Fire', sessions: [], meta: { page: 1, per_page: 1, total: 0 } })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()

    const { result } = renderHook(() => useExamHistory('e-1'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(firstCall(fetchMock).url).toBe('/api/v1/portal/exams/e-1/history?page=1&per_page=1')
  })

  it('useExamHistory does not fetch without an exam id', () => {
    const fetchMock = okFetch({})
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()

    renderHook(() => useExamHistory(''), { wrapper })

    expect(fetchMock).not.toHaveBeenCalled()
  })
})

describe('session mutations', () => {
  it('useSaveAnswer sends the answer as JSON to the question path', async () => {
    const fetchMock = okFetch({ question_id: 'q-1', saved_at: 'now', remaining_seconds: 600 })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()

    const { result } = renderHook(() => useSaveAnswer('s-4'), { wrapper })
    const answer = { selected_option_ids: ['o-1'], text_answer: null, time_spent_seconds: 12 }
    await act(async () => {
      await result.current.mutateAsync({ questionId: 'q-1', answer })
    })

    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/portal/sessions/s-4/answers/q-1')
    expect(init.method).toBe('PUT')
    expect(headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual(answer)
  })

  it('useSubmitSession posts to the submit path', async () => {
    const fetchMock = okFetch({ session_id: 's-5', status: 'submitted', submitted_at: 'now', score_pct: 80, passed: true })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()

    const { result } = renderHook(() => useSubmitSession('s-5'), { wrapper })
    await act(async () => {
      await result.current.mutateAsync()
    })

    const { url, init } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/portal/sessions/s-5/submit')
    expect(init.method).toBe('POST')
  })

  it('useReportEvent posts the event type as JSON', async () => {
    const fetchMock = okFetch({ warn: true, event_count: 1 })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()

    const { result } = renderHook(() => useReportEvent('s-6'), { wrapper })
    await act(async () => {
      await result.current.mutateAsync({ type: 'tab_switch' })
    })

    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/portal/sessions/s-6/events')
    expect(init.method).toBe('POST')
    expect(headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual({ type: 'tab_switch' })
  })

  it('useSaveAnswer rejects with the server error', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'SESSION_EXPIRED', message: 'session expired' }, 409))
    const { wrapper } = makeWrapper()

    const { result } = renderHook(() => useSaveAnswer('s-7'), { wrapper })
    await act(async () => {
      await expect(
        result.current.mutateAsync({
          questionId: 'q-1',
          answer: { selected_option_ids: [], text_answer: 'x', time_spent_seconds: 1 },
        }),
      ).rejects.toMatchObject({ code: 'SESSION_EXPIRED', status: 409 })
    })
  })
})
