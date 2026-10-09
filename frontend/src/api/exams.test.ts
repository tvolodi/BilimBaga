import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement, type ReactNode } from 'react'
import {
  ExamApiError,
  useExams,
  useExam,
  useCreateExam,
  useUpdateExam,
  usePublishExam,
  useUnpublishExam,
  useArchiveExam,
  useAddSection,
  useAddRule,
  useUpdateRule,
  useDeleteRule,
  useSetRuleQuestions,
  useExamAssignments,
  useAddAssignment,
  useDeleteAssignment,
  useEligibleCounts,
  type CreateExamPayload,
  type ExamDetail,
  type ExamAssignment,
  type PublishValidationDetail,
  type QuestionRuleDetail,
  type QuestionRulePayload,
} from './exams'

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

function errorFetch(
  error: { code: string; message: string; fields?: Array<{ field: string; message: string }>; details?: unknown },
  status = 422,
) {
  return vi.fn().mockResolvedValue({ ok: false, status, json: async () => ({ data: null, error }) })
}

function noContentFetch(ok = true, status = 204) {
  return vi.fn().mockResolvedValue({ ok, status })
}

function firstCall(fetchMock: ReturnType<typeof vi.fn>) {
  const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
  return { url, init, headers: (init?.headers ?? {}) as Record<string, string> }
}

const sampleExam: ExamDetail = {
  id: 'exam-1',
  title: 'Safety basics',
  description: null,
  status: 'draft',
  time_limit_minutes: 30,
  passing_score_pct: 70,
  max_attempts: 2,
  available_from: null,
  available_until: null,
  shuffle_questions: true,
  shuffle_options: false,
  show_answers: 'after_completion',
  on_tab_switch: 'log',
  certificate_enabled: true,
  created_by: 'user-1',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-02T00:00:00Z',
  sections: [{ id: 'sec-1', title: 'Part A', sort_order: 0 }],
  rules: [],
}

const samplePayload: CreateExamPayload = {
  title: 'Safety basics',
  time_limit_minutes: 30,
  passing_score_pct: 70,
  max_attempts: 2,
  shuffle_questions: true,
  shuffle_options: false,
  show_answers: 'after_completion',
  on_tab_switch: 'log',
  certificate_enabled: true,
}

const sampleRule: QuestionRuleDetail = {
  id: 'rule-9',
  section_id: 'sec-1',
  mode: 'random',
  category_id: 'cat-1',
  tag_ids: ['tag-1'],
  difficulty: 'easy',
  count: 5,
  sort_order: 0,
}

const ruleBody: QuestionRulePayload = {
  section_id: 'sec-1',
  mode: 'random',
  category_id: 'cat-1',
  tag_ids: ['tag-1'],
  difficulty: 'easy',
  count: 5,
  sort_order: 0,
}

const sampleAssignment: ExamAssignment = {
  id: 'asg-1',
  exam_id: 'exam-1',
  assignee_type: 'department',
  assignee_id: 'dep-1',
  deadline: null,
  assigned_by: 'user-1',
  assigned_at: '2026-01-03T00:00:00Z',
}

describe('useExams', () => {
  it('GETs the exam list without a query string when no filters are set', async () => {
    const listing = { items: [], meta: { page: 1, per_page: 20, total: 0 } }
    const fetchMock = okFetch(listing)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useExams(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(listing)
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams')
    expect(init.credentials).toBe('include')
    expect(headers.Authorization).toBe('Bearer tok')
  })

  it('builds the query string from page, per_page, status and search', async () => {
    const fetchMock = okFetch({ items: [], meta: { page: 2, per_page: 25, total: 0 } })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(
      () => useExams({ page: 2, per_page: 25, status: 'active', search: 'due date' }),
      { wrapper },
    )
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(firstCall(fetchMock).url).toBe('/api/v1/exams?page=2&per_page=25&status=active&search=due+date')
  })

  it('drops a zero page and empty search from the query string', async () => {
    const fetchMock = okFetch({ items: [], meta: { page: 1, per_page: 20, total: 0 } })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useExams({ page: 0, search: '' }), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(firstCall(fetchMock).url).toBe('/api/v1/exams')
  })

  it('omits Authorization when no token is cached', async () => {
    const fetchMock = okFetch({ items: [], meta: { page: 1, per_page: 20, total: 0 } })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper(null)
    const { result } = renderHook(() => useExams(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(firstCall(fetchMock).headers.Authorization).toBeUndefined()
  })

  it('surfaces the error envelope as an ExamApiError with code, status and fields', async () => {
    vi.stubGlobal(
      'fetch',
      errorFetch(
        { code: 'VALIDATION_ERROR', message: 'bad page', fields: [{ field: 'page', message: 'min 1' }] },
        400,
      ),
    )
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useExams({ page: -1 }), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toBeInstanceOf(ExamApiError)
    expect(result.current.error).toMatchObject({
      code: 'VALIDATION_ERROR',
      message: 'bad page',
      httpStatus: 400,
      fields: [{ field: 'page', message: 'min 1' }],
      unsatisfiedRules: undefined,
    })
  })

  it('falls back to ERR_UNKNOWN when a failing response has no error envelope', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: false, status: 500, json: async () => ({ data: null, error: null }) }),
    )
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useExams(), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toMatchObject({
      code: 'ERR_UNKNOWN',
      message: 'Request failed: 500',
      httpStatus: 500,
    })
  })
})

describe('useExam', () => {
  it('GETs the exam detail by id and returns the data', async () => {
    const fetchMock = okFetch(sampleExam)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useExam('exam-1'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(sampleExam)
    const { url, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams/exam-1')
    expect(headers.Authorization).toBe('Bearer tok')
  })

  it('sends no request while the id is null', () => {
    const fetchMock = okFetch(sampleExam)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useExam(null), { wrapper })
    expect(result.current.fetchStatus).toBe('idle')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('surfaces a 404 envelope as an ExamApiError', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'EXAM_NOT_FOUND', message: 'missing' }, 404))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useExam('exam-x'), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toMatchObject({ code: 'EXAM_NOT_FOUND', httpStatus: 404 })
  })
})

describe('useCreateExam', () => {
  it('POSTs the JSON body with the content type header and returns the created exam', async () => {
    const fetchMock = okFetch(sampleExam)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useCreateExam(), { wrapper })
    const created = await result.current.mutateAsync(samplePayload)
    expect(created).toEqual(sampleExam)
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams')
    expect(init.method).toBe('POST')
    expect(headers['Content-Type']).toBe('application/json')
    expect(headers.Authorization).toBe('Bearer tok')
    expect(JSON.parse(init.body as string)).toEqual(samplePayload)
  })

  it('invalidates every exams query after a successful create', async () => {
    vi.stubGlobal('fetch', okFetch(sampleExam))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useCreateExam(), { wrapper })
    await result.current.mutateAsync(samplePayload)
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams'] })
  })

  it('rejects with an ExamApiError carrying field errors and does not invalidate', async () => {
    vi.stubGlobal(
      'fetch',
      errorFetch(
        { code: 'VALIDATION_ERROR', message: 'invalid', fields: [{ field: 'title', message: 'required' }] },
        422,
      ),
    )
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useCreateExam(), { wrapper })
    await expect(result.current.mutateAsync({ ...samplePayload, title: '' })).rejects.toMatchObject({
      code: 'VALIDATION_ERROR',
      httpStatus: 422,
      fields: [{ field: 'title', message: 'required' }],
    })
    expect(invalidate).not.toHaveBeenCalled()
  })
})

describe('useUpdateExam', () => {
  it('PUTs the body to the exam URL and returns the updated exam', async () => {
    const updated = { ...sampleExam, title: 'Safety advanced' }
    const fetchMock = okFetch(updated)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useUpdateExam('exam-1'), { wrapper })
    const out = await result.current.mutateAsync({ ...samplePayload, title: 'Safety advanced' })
    expect(out).toEqual(updated)
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams/exam-1')
    expect(init.method).toBe('PUT')
    expect(headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual({ ...samplePayload, title: 'Safety advanced' })
  })

  it('invalidates the list and the exam detail after a successful update', async () => {
    vi.stubGlobal('fetch', okFetch(sampleExam))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useUpdateExam('exam-1'), { wrapper })
    await result.current.mutateAsync(samplePayload)
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams'] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams', 'exam-1'] })
  })

  it('rejects with the API error code on failure', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'EXAM_ARCHIVED', message: 'read only' }, 409))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useUpdateExam('exam-1'), { wrapper })
    await expect(result.current.mutateAsync(samplePayload)).rejects.toMatchObject({
      code: 'EXAM_ARCHIVED',
      httpStatus: 409,
    })
  })
})

describe('usePublishExam', () => {
  it('POSTs to the publish URL with no body and returns the published exam', async () => {
    const published = { ...sampleExam, status: 'active' as const }
    const fetchMock = okFetch(published)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => usePublishExam('exam-1'), { wrapper })
    const out = await result.current.mutateAsync()
    expect(out).toEqual(published)
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams/exam-1/publish')
    expect(init.method).toBe('POST')
    expect(init.body).toBeUndefined()
    expect(headers['Content-Type']).toBeUndefined()
  })

  it('invalidates the list and the exam detail after publishing', async () => {
    vi.stubGlobal('fetch', okFetch(sampleExam))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => usePublishExam('exam-1'), { wrapper })
    await result.current.mutateAsync()
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams'] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams', 'exam-1'] })
  })

  it('exposes unsatisfied rules when the publish error details are an array', async () => {
    const detail: PublishValidationDetail = {
      rule_id: 'rule-9',
      required: 5,
      available: 2,
      filter: { difficulty: 'easy' },
    }
    vi.stubGlobal(
      'fetch',
      errorFetch({ code: 'INSUFFICIENT_QUESTIONS', message: 'not enough', details: [detail] }, 409),
    )
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => usePublishExam('exam-1'), { wrapper })
    const err = await result.current.mutateAsync().catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ExamApiError)
    expect(err).toMatchObject({
      code: 'INSUFFICIENT_QUESTIONS',
      httpStatus: 409,
      unsatisfiedRules: [detail],
      details: [detail],
    })
    expect(invalidate).not.toHaveBeenCalled()
  })

  it('keeps an object-shaped details payload raw and leaves unsatisfiedRules unset', async () => {
    vi.stubGlobal(
      'fetch',
      errorFetch({ code: 'PUBLISH_BLOCKED', message: 'blocked', details: { reason: 'no sections' } }, 409),
    )
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => usePublishExam('exam-1'), { wrapper })
    const err = await result.current.mutateAsync().catch((e: unknown) => e)
    expect(err).toMatchObject({ code: 'PUBLISH_BLOCKED', details: { reason: 'no sections' } })
    expect((err as ExamApiError).unsatisfiedRules).toBeUndefined()
  })
})

describe('useUnpublishExam', () => {
  it('POSTs to the unpublish URL and resolves with the id and status', async () => {
    const fetchMock = okFetch({ id: 'exam-1', status: 'draft' })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useUnpublishExam(), { wrapper })
    const out = await result.current.mutateAsync('exam-1')
    expect(out).toEqual({ id: 'exam-1', status: 'draft' })
    const { url, init } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams/exam-1/unpublish')
    expect(init.method).toBe('POST')
  })

  it('invalidates the list and the given exam detail', async () => {
    vi.stubGlobal('fetch', okFetch({ id: 'exam-7', status: 'draft' }))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useUnpublishExam(), { wrapper })
    await result.current.mutateAsync('exam-7')
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams'] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams', 'exam-7'] })
  })

  it('rejects with the API error code on failure', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'EXAM_HAS_ATTEMPTS', message: 'has attempts' }, 409))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useUnpublishExam(), { wrapper })
    await expect(result.current.mutateAsync('exam-1')).rejects.toMatchObject({
      code: 'EXAM_HAS_ATTEMPTS',
      httpStatus: 409,
    })
  })
})

describe('useArchiveExam', () => {
  it('POSTs to the archive URL and resolves with the id and status', async () => {
    const fetchMock = okFetch({ id: 'exam-1', status: 'archived' })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useArchiveExam(), { wrapper })
    const out = await result.current.mutateAsync('exam-1')
    expect(out).toEqual({ id: 'exam-1', status: 'archived' })
    const { url, init } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams/exam-1/archive')
    expect(init.method).toBe('POST')
  })

  it('invalidates the list and the given exam detail', async () => {
    vi.stubGlobal('fetch', okFetch({ id: 'exam-3', status: 'archived' }))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useArchiveExam(), { wrapper })
    await result.current.mutateAsync('exam-3')
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams'] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams', 'exam-3'] })
  })

  it('rejects with the API error code on failure', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'FORBIDDEN', message: 'no' }, 403))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useArchiveExam(), { wrapper })
    await expect(result.current.mutateAsync('exam-1')).rejects.toMatchObject({
      code: 'FORBIDDEN',
      httpStatus: 403,
    })
  })
})

describe('useAddSection', () => {
  it('POSTs title and sort_order to the sections URL without the examId', async () => {
    const fetchMock = okFetch({ id: 'sec-2', title: 'Part B', sort_order: 1 })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useAddSection(), { wrapper })
    const out = await result.current.mutateAsync({ examId: 'exam-1', title: 'Part B', sort_order: 1 })
    expect(out).toEqual({ id: 'sec-2', title: 'Part B', sort_order: 1 })
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams/exam-1/sections')
    expect(init.method).toBe('POST')
    expect(headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual({ title: 'Part B', sort_order: 1 })
  })

  it('omits title from the body when none is given', async () => {
    const fetchMock = okFetch({ id: 'sec-3', title: null, sort_order: 0 })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useAddSection(), { wrapper })
    await result.current.mutateAsync({ examId: 'exam-1', sort_order: 0 })
    const { init } = firstCall(fetchMock)
    expect(JSON.parse(init.body as string)).toEqual({ sort_order: 0 })
  })

  it('invalidates the exam detail after adding a section', async () => {
    vi.stubGlobal('fetch', okFetch({ id: 'sec-2', title: null, sort_order: 0 }))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useAddSection(), { wrapper })
    await result.current.mutateAsync({ examId: 'exam-1', sort_order: 0 })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams', 'exam-1'] })
  })

  it('rejects with the API error code on failure', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'EXAM_NOT_FOUND', message: 'missing' }, 404))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useAddSection(), { wrapper })
    await expect(result.current.mutateAsync({ examId: 'exam-x', sort_order: 0 })).rejects.toMatchObject({
      code: 'EXAM_NOT_FOUND',
      httpStatus: 404,
    })
  })
})

describe('useAddRule', () => {
  it('POSTs the rule payload to the rules URL and returns the created rule', async () => {
    const fetchMock = okFetch(sampleRule)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useAddRule('exam-1'), { wrapper })
    const out = await result.current.mutateAsync(ruleBody)
    expect(out).toEqual(sampleRule)
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams/exam-1/rules')
    expect(init.method).toBe('POST')
    expect(headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual(ruleBody)
  })

  it('invalidates the exam detail after adding a rule', async () => {
    vi.stubGlobal('fetch', okFetch(sampleRule))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useAddRule('exam-1'), { wrapper })
    await result.current.mutateAsync(ruleBody)
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams', 'exam-1'] })
  })

  it('rejects with field errors and does not invalidate', async () => {
    vi.stubGlobal(
      'fetch',
      errorFetch({ code: 'VALIDATION_ERROR', message: 'bad', fields: [{ field: 'count', message: 'min 1' }] }, 422),
    )
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useAddRule('exam-1'), { wrapper })
    await expect(result.current.mutateAsync({ ...ruleBody, count: 0 })).rejects.toMatchObject({
      code: 'VALIDATION_ERROR',
      fields: [{ field: 'count', message: 'min 1' }],
    })
    expect(invalidate).not.toHaveBeenCalled()
  })
})

describe('useUpdateRule', () => {
  it('PUTs the body to the rule URL and returns the updated rule', async () => {
    const updated = { ...sampleRule, count: 8 }
    const fetchMock = okFetch(updated)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useUpdateRule('exam-1'), { wrapper })
    const out = await result.current.mutateAsync({ ruleId: 'rule-9', body: { ...ruleBody, count: 8 } })
    expect(out).toEqual(updated)
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams/exam-1/rules/rule-9')
    expect(init.method).toBe('PUT')
    expect(headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual({ ...ruleBody, count: 8 })
  })

  it('invalidates the exam detail after updating a rule', async () => {
    vi.stubGlobal('fetch', okFetch(sampleRule))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useUpdateRule('exam-1'), { wrapper })
    await result.current.mutateAsync({ ruleId: 'rule-9', body: ruleBody })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams', 'exam-1'] })
  })

  it('rejects with the API error code on failure', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'RULE_NOT_FOUND', message: 'gone' }, 404))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useUpdateRule('exam-1'), { wrapper })
    await expect(result.current.mutateAsync({ ruleId: 'rule-x', body: ruleBody })).rejects.toMatchObject({
      code: 'RULE_NOT_FOUND',
      httpStatus: 404,
    })
  })
})

describe('useDeleteRule', () => {
  it('DELETEs the rule URL and resolves with undefined on 204', async () => {
    const fetchMock = noContentFetch()
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useDeleteRule('exam-1'), { wrapper })
    const out = await result.current.mutateAsync('rule-9')
    expect(out).toBeUndefined()
    const { url, init } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams/exam-1/rules/rule-9')
    expect(init.method).toBe('DELETE')
    expect(init.body).toBeUndefined()
  })

  it('invalidates the exam detail after deleting a rule', async () => {
    vi.stubGlobal('fetch', noContentFetch())
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useDeleteRule('exam-1'), { wrapper })
    await result.current.mutateAsync('rule-9')
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams', 'exam-1'] })
  })

  it('throws ERR_UNKNOWN when a 204 response is flagged as not ok', async () => {
    vi.stubGlobal('fetch', noContentFetch(false, 204))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useDeleteRule('exam-1'), { wrapper })
    await expect(result.current.mutateAsync('rule-9')).rejects.toMatchObject({
      code: 'ERR_UNKNOWN',
      message: 'Request failed',
      httpStatus: 204,
    })
  })

  it('rejects with the API error envelope on a 404', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'RULE_NOT_FOUND', message: 'gone' }, 404))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useDeleteRule('exam-1'), { wrapper })
    await expect(result.current.mutateAsync('rule-9')).rejects.toMatchObject({
      code: 'RULE_NOT_FOUND',
      httpStatus: 404,
    })
  })
})

describe('useSetRuleQuestions', () => {
  it('PUTs the question list wrapped in a questions key to the rule questions URL', async () => {
    const fetchMock = noContentFetch()
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useSetRuleQuestions('exam-1'), { wrapper })
    const questions = [
      { question_id: 'q-1', sort_order: 0 },
      { question_id: 'q-2', sort_order: 1 },
    ]
    const out = await result.current.mutateAsync({ ruleId: 'rule-9', questions })
    expect(out).toBeUndefined()
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams/exam-1/rules/rule-9/questions')
    expect(init.method).toBe('PUT')
    expect(headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual({ questions })
  })

  it('invalidates the exam detail after setting questions', async () => {
    vi.stubGlobal('fetch', noContentFetch())
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useSetRuleQuestions('exam-1'), { wrapper })
    await result.current.mutateAsync({ ruleId: 'rule-9', questions: [] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams', 'exam-1'] })
  })

  it('rejects with the API error code on failure', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'QUESTION_NOT_IN_BANK', message: 'unknown question' }, 422))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useSetRuleQuestions('exam-1'), { wrapper })
    await expect(
      result.current.mutateAsync({ ruleId: 'rule-9', questions: [{ question_id: 'q-x', sort_order: 0 }] }),
    ).rejects.toMatchObject({ code: 'QUESTION_NOT_IN_BANK', httpStatus: 422 })
  })
})

describe('useExamAssignments', () => {
  it('GETs the assignments list and returns it', async () => {
    const fetchMock = okFetch([sampleAssignment])
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useExamAssignments('exam-1'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual([sampleAssignment])
    const { url, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams/exam-1/assignments')
    expect(headers.Authorization).toBe('Bearer tok')
  })

  it('sends no request while the exam id is empty', () => {
    const fetchMock = okFetch([])
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useExamAssignments(undefined), { wrapper })
    expect(result.current.fetchStatus).toBe('idle')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('surfaces an error envelope as an ExamApiError', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'EXAM_NOT_FOUND', message: 'missing' }, 404))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useExamAssignments('exam-x'), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toMatchObject({ code: 'EXAM_NOT_FOUND', httpStatus: 404 })
  })
})

describe('useAddAssignment', () => {
  it('POSTs the assignment to the assign URL and returns the created assignment', async () => {
    const fetchMock = okFetch(sampleAssignment)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useAddAssignment('exam-1'), { wrapper })
    const body = { assignee_type: 'department' as const, assignee_id: 'dep-1', deadline: '2026-12-31T00:00:00Z' }
    const out = await result.current.mutateAsync(body)
    expect(out).toEqual(sampleAssignment)
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams/exam-1/assign')
    expect(init.method).toBe('POST')
    expect(headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual(body)
  })

  it('invalidates only the assignments query after adding', async () => {
    vi.stubGlobal('fetch', okFetch(sampleAssignment))
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useAddAssignment('exam-1'), { wrapper })
    await result.current.mutateAsync({ assignee_type: 'all' })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams', 'exam-1', 'assignments'] })
  })

  it('rejects with the API error code on failure', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'ASSIGNMENT_EXISTS', message: 'already assigned' }, 409))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useAddAssignment('exam-1'), { wrapper })
    await expect(result.current.mutateAsync({ assignee_type: 'all' })).rejects.toMatchObject({
      code: 'ASSIGNMENT_EXISTS',
      httpStatus: 409,
    })
  })
})

describe('useDeleteAssignment', () => {
  it('DELETEs the assignment URL and resolves with undefined on 204', async () => {
    const fetchMock = noContentFetch()
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useDeleteAssignment('exam-1'), { wrapper })
    const out = await result.current.mutateAsync('asg-1')
    expect(out).toBeUndefined()
    const { url, init } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams/exam-1/assign/asg-1')
    expect(init.method).toBe('DELETE')
  })

  it('invalidates only the assignments query after deleting', async () => {
    vi.stubGlobal('fetch', noContentFetch())
    const { qc, wrapper } = makeWrapper()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useDeleteAssignment('exam-1'), { wrapper })
    await result.current.mutateAsync('asg-1')
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exams', 'exam-1', 'assignments'] })
  })

  it('rejects with the API error envelope on a 404', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'ASSIGNMENT_NOT_FOUND', message: 'gone' }, 404))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useDeleteAssignment('exam-1'), { wrapper })
    await expect(result.current.mutateAsync('asg-x')).rejects.toMatchObject({
      code: 'ASSIGNMENT_NOT_FOUND',
      httpStatus: 404,
    })
  })
})

describe('useEligibleCounts', () => {
  it('GETs the eligible counts and returns them', async () => {
    const payload = { counts: [{ rule_id: 'rule-9', eligible: 12 }] }
    const fetchMock = okFetch(payload)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useEligibleCounts('exam-1'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(payload)
    const { url, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/exams/exam-1/rules/eligible-counts')
    expect(headers.Authorization).toBe('Bearer tok')
  })

  it('sends no request while the exam id is empty', () => {
    const fetchMock = okFetch({ counts: [] })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useEligibleCounts(''), { wrapper })
    expect(result.current.fetchStatus).toBe('idle')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('surfaces an error envelope as an ExamApiError', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'EXAM_NOT_FOUND', message: 'missing' }, 404))
    const { wrapper } = makeWrapper()
    const { result } = renderHook(() => useEligibleCounts('exam-x'), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toMatchObject({ code: 'EXAM_NOT_FOUND', httpStatus: 404 })
  })
})
