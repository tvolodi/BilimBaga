import { describe, it, expect, vi, afterEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement, type ReactNode } from 'react'
import {
  useQuestions,
  useCategories,
  useTags,
  useQuestionVersions,
  useTransitionStatus,
  useDeleteQuestion,
  useImportQuestions,
  useQuestion,
  useCreateQuestion,
  useUpdateQuestion,
  useUpdateQuestionStatus,
  useCreateTag,
  type QuestionDetail,
  type QuestionListResponse,
  type QuestionListItem,
} from './questions'

afterEach(() => vi.unstubAllGlobals())

function makeClient(token: string | null = 'tok') {
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

function errorFetch(error: { code: string; message: string }, status = 400) {
  return vi.fn().mockResolvedValue({ ok: false, status, json: async () => ({ data: null, error }) })
}

function firstCall(fetchMock: ReturnType<typeof vi.fn>) {
  const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
  return { url, init, headers: (init?.headers ?? {}) as Record<string, string> }
}

const listItem: QuestionListItem = {
  id: 'q-1',
  type: 'single',
  difficulty: 'easy',
  status: 'draft',
  category_id: 'cat-1',
  category_name: 'Safety',
  default_locale: 'en',
  version: 1,
  locale_coverage: ['en'],
  stem_preview: 'What is PPE?',
  tags: ['ppe'],
  created_by: 'user-1',
  created_by_name: 'Admin',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const listResponse: QuestionListResponse = {
  items: [listItem],
  meta: { page: 1, per_page: 20, total: 1 },
}

const detail: QuestionDetail = {
  id: 'q-1',
  type: 'single',
  difficulty: 'easy',
  status: 'draft',
  category_id: 'cat-1',
  category_name: 'Safety',
  default_locale: 'en',
  version: 1,
  parent_id: null,
  auto_grade: true,
  model_answer: null,
  locale_coverage: ['en'],
  translations: { en: { stem: 'What is PPE?', explanation: '' } },
  answer_options: [],
  tag_ids: ['tag-1'],
  created_by: 'user-1',
  created_by_name: 'Admin',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const createPayload = {
  type: 'single',
  difficulty: 'easy',
  category_id: 'cat-1',
  default_locale: 'en',
  translations: { en: { stem: 'What is PPE?' } },
}

describe('useQuestions', () => {
  it('GETs the bare questions list when no filters are set', async () => {
    const fetchMock = okFetch(listResponse)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useQuestions(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(listResponse)
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/questions')
    expect(init.credentials).toBe('include')
    expect(headers.Authorization).toBe('Bearer tok')
  })

  it('serialises every filter into the query string', async () => {
    const fetchMock = okFetch(listResponse)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient()
    const { result } = renderHook(
      () =>
        useQuestions({
          page: 2,
          per_page: 50,
          sort: 'difficulty',
          order: 'desc',
          category_id: 'cat-1',
          tag_ids: ['tag-1', 'tag-2'],
          statuses: ['draft', 'review'],
          difficulties: ['easy', 'hard'],
          type: 'likert',
          locale: 'kk',
          locale_missing: 'ru',
          include_versions: true,
          search: 'PPE gloves',
        }),
      { wrapper },
    )
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const { url } = firstCall(fetchMock)
    const params = new URLSearchParams(url.split('?')[1])
    expect(url.startsWith('/api/v1/questions?')).toBe(true)
    expect(Object.fromEntries(params.entries())).toEqual({
      page: '2',
      per_page: '50',
      sort: 'difficulty',
      order: 'desc',
      category_id: 'cat-1',
      tag_ids: 'tag-1,tag-2',
      statuses: 'draft,review',
      difficulties: 'easy,hard',
      type: 'likert',
      locale: 'kk',
      locale_missing: 'ru',
      include_versions: 'true',
      search: 'PPE gloves',
    })
  })

  it('omits empty arrays, falsy values and include_versions=false from the query string', async () => {
    const fetchMock = okFetch(listResponse)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient()
    const { result } = renderHook(
      () =>
        useQuestions({
          page: 0,
          tag_ids: [],
          statuses: [],
          difficulties: [],
          include_versions: false,
          search: '',
        }),
      { wrapper },
    )
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(firstCall(fetchMock).url).toBe('/api/v1/questions')
  })

  it('omits Authorization when no token is cached', async () => {
    const fetchMock = okFetch(listResponse)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient(null)
    const { result } = renderHook(() => useQuestions(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(firstCall(fetchMock).headers.Authorization).toBeUndefined()
  })

  it('surfaces the error code from the envelope', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'FORBIDDEN', message: 'no access' }, 403))
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useQuestions(), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toBeInstanceOf(Error)
    expect(result.current.error).toMatchObject({ code: 'FORBIDDEN', message: 'no access' })
  })

  it('falls back to ERR_UNKNOWN when the envelope has no code', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: false, status: 500, json: async () => ({ data: null, error: { message: 'boom' } }) }),
    )
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useQuestions(), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toMatchObject({ message: 'boom', code: 'ERR_UNKNOWN' })
  })

  it('throws a plain request-failed error when a failing response has no envelope', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: false, status: 502, json: async () => ({ data: null, error: null }) }),
    )
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useQuestions(), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toMatchObject({ message: 'Request failed: 502' })
    expect((result.current.error as Error & { code?: string }).code).toBeUndefined()
  })
})

describe('useCategories', () => {
  it('GETs the category tree and returns it', async () => {
    const tree = [{ id: 'cat-1', name: 'Safety', parent_id: null }]
    const fetchMock = okFetch(tree)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useCategories(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(tree)
    const { url, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/categories')
    expect(headers.Authorization).toBe('Bearer tok')
  })

  it('surfaces the error code from the envelope', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'UNAUTHORIZED', message: 'login' }, 401))
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useCategories(), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toMatchObject({ code: 'UNAUTHORIZED' })
  })
})

describe('useTags', () => {
  it('GETs the tag list and returns it', async () => {
    const tags = [{ id: 'tag-1', name: 'ppe' }]
    const fetchMock = okFetch(tags)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useTags(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(tags)
    expect(firstCall(fetchMock).url).toBe('/api/v1/tags')
  })

  it('surfaces the error code from the envelope', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'FORBIDDEN', message: 'no' }, 403))
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useTags(), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toMatchObject({ code: 'FORBIDDEN' })
  })
})

describe('useQuestionVersions', () => {
  it('GETs the version history for the question', async () => {
    const versions = [
      { version: 1, created_at: '2026-01-01T00:00:00Z', created_by_name: 'Admin', change_summary: 'created' },
    ]
    const fetchMock = okFetch(versions)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useQuestionVersions('q-1'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(versions)
    expect(firstCall(fetchMock).url).toBe('/api/v1/questions/q-1/versions')
  })

  it('sends no request while the id is empty', () => {
    const fetchMock = okFetch([])
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useQuestionVersions(''), { wrapper })
    expect(result.current.fetchStatus).toBe('idle')
    expect(fetchMock).not.toHaveBeenCalled()
  })
})

describe('useQuestion', () => {
  it('GETs the question detail and returns it', async () => {
    const fetchMock = okFetch(detail)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useQuestion('q-1'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(detail)
    const { url, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/questions/q-1')
    expect(headers.Authorization).toBe('Bearer tok')
  })

  it.each([null, ''])('sends no request when the id is %j', (id) => {
    const fetchMock = okFetch(detail)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useQuestion(id), { wrapper })
    expect(result.current.fetchStatus).toBe('idle')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('surfaces a 404 envelope with its code', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'QUESTION_NOT_FOUND', message: 'missing' }, 404))
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useQuestion('q-x'), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toMatchObject({ code: 'QUESTION_NOT_FOUND', message: 'missing' })
  })
})

describe('useCreateQuestion', () => {
  it('POSTs the JSON payload to the questions URL and returns the created question', async () => {
    const fetchMock = okFetch(detail)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useCreateQuestion(), { wrapper })
    const out = await result.current.mutateAsync(createPayload)
    expect(out).toEqual(detail)
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/questions')
    expect(init.method).toBe('POST')
    expect(headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual(createPayload)
  })

  it('invalidates the questions queries after a successful create', async () => {
    vi.stubGlobal('fetch', okFetch(detail))
    const { qc, wrapper } = makeClient()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useCreateQuestion(), { wrapper })
    await result.current.mutateAsync(createPayload)
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['questions'] })
  })

  it('rejects with the API error code and does not invalidate', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'VALIDATION_ERROR', message: 'stem required' }, 422))
    const { qc, wrapper } = makeClient()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useCreateQuestion(), { wrapper })
    await expect(result.current.mutateAsync(createPayload)).rejects.toMatchObject({
      code: 'VALIDATION_ERROR',
      message: 'stem required',
    })
    expect(invalidate).not.toHaveBeenCalled()
  })
})

describe('useUpdateQuestion', () => {
  it('PUTs the payload to the question URL and returns the updated question', async () => {
    const fetchMock = okFetch(detail)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useUpdateQuestion(), { wrapper })
    const out = await result.current.mutateAsync({ id: 'q-1', payload: { difficulty: 'hard' } })
    expect(out).toEqual(detail)
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/questions/q-1')
    expect(init.method).toBe('PUT')
    expect(headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual({ difficulty: 'hard' })
  })

  it('invalidates the question detail and the list when the id is unchanged', async () => {
    vi.stubGlobal('fetch', okFetch(detail))
    const { qc, wrapper } = makeClient()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useUpdateQuestion(), { wrapper })
    await result.current.mutateAsync({ id: 'q-1', payload: { difficulty: 'hard' } })
    expect(invalidate.mock.calls).toEqual([
      [{ queryKey: ['questions', 'q-1'] }],
      [{ queryKey: ['questions'] }],
    ])
  })

  it('also invalidates the new version id when an active question was versioned', async () => {
    vi.stubGlobal('fetch', okFetch({ ...detail, id: 'q-2', version: 2, parent_id: 'q-1' }))
    const { qc, wrapper } = makeClient()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useUpdateQuestion(), { wrapper })
    await result.current.mutateAsync({ id: 'q-1', payload: { difficulty: 'hard' } })
    expect(invalidate.mock.calls).toEqual([
      [{ queryKey: ['questions', 'q-1'] }],
      [{ queryKey: ['questions', 'q-2'] }],
      [{ queryKey: ['questions'] }],
    ])
  })

  it('rejects with the API error code on failure', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'QUESTION_LOCKED', message: 'locked' }, 409))
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useUpdateQuestion(), { wrapper })
    await expect(result.current.mutateAsync({ id: 'q-1', payload: {} })).rejects.toMatchObject({
      code: 'QUESTION_LOCKED',
      message: 'locked',
    })
  })
})

describe('useTransitionStatus', () => {
  it('POSTs the target status to the status URL and clears the list cache on settle', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204 })
    vi.stubGlobal('fetch', fetchMock)
    const { qc, wrapper } = makeClient()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useTransitionStatus(), { wrapper })
    await result.current.mutateAsync({ id: 'q-1', target_status: 'review' })
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/questions/q-1/status')
    expect(init.method).toBe('POST')
    expect(headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual({ status: 'review' })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['questions'] })
  })

  it('optimistically sets the status on the matching list item while the request is pending', async () => {
    let reject!: (e: Error) => void
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(() => new Promise((_, rej) => { reject = rej })),
    )
    const { qc, wrapper } = makeClient()
    const listKey = ['questions', { page: 1 }]
    qc.setQueryData<QuestionListResponse>(listKey, listResponse)
    const { result } = renderHook(() => useTransitionStatus(), { wrapper })
    act(() => {
      result.current.mutate({ id: 'q-1', target_status: 'active' })
    })
    await waitFor(() => {
      const cached = qc.getQueryData<QuestionListResponse>(listKey)
      expect(cached?.items[0].status).toBe('active')
    })
    await act(async () => {
      reject(new Error('network down'))
    })
    await waitFor(() => expect(result.current.isError).toBe(true))
  })

  it('restores the previous list cache when the request fails', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'INVALID_TRANSITION', message: 'not allowed' }, 409))
    const { qc, wrapper } = makeClient()
    const listKey = ['questions', { page: 1 }]
    qc.setQueryData<QuestionListResponse>(listKey, listResponse)
    const { result } = renderHook(() => useTransitionStatus(), { wrapper })
    await expect(result.current.mutateAsync({ id: 'q-1', target_status: 'active' })).rejects.toMatchObject({
      code: 'INVALID_TRANSITION',
    })
    expect(qc.getQueryData<QuestionListResponse>(listKey)?.items[0].status).toBe('draft')
  })

  it('leaves cache entries without an items array untouched', async () => {
    vi.stubGlobal('fetch', okFetch(undefined))
    const { qc, wrapper } = makeClient()
    qc.setQueryData(['questions', 'q-1'], detail)
    const { result } = renderHook(() => useTransitionStatus(), { wrapper })
    await result.current.mutateAsync({ id: 'q-1', target_status: 'active' })
    expect(qc.getQueryData(['questions', 'q-1'])).toEqual(detail)
  })
})

describe('useDeleteQuestion', () => {
  it('DELETEs the question URL and resolves with undefined on 204', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204 })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useDeleteQuestion(), { wrapper })
    const out = await result.current.mutateAsync('q-1')
    expect(out).toBeUndefined()
    const { url, init } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/questions/q-1')
    expect(init.method).toBe('DELETE')
  })

  it('invalidates the questions queries after a successful delete', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 204 }))
    const { qc, wrapper } = makeClient()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useDeleteQuestion(), { wrapper })
    await result.current.mutateAsync('q-1')
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['questions'] })
  })

  it('rejects with the API error code on a failed delete', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'QUESTION_IN_USE', message: 'used by exam' }, 409))
    const { qc, wrapper } = makeClient()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useDeleteQuestion(), { wrapper })
    await expect(result.current.mutateAsync('q-1')).rejects.toMatchObject({ code: 'QUESTION_IN_USE' })
    expect(invalidate).not.toHaveBeenCalled()
  })
})

describe('useImportQuestions', () => {
  it('POSTs the file as multipart form data to the import URL without dry_run', async () => {
    const summary = { valid_count: 3, error_rows: [], warning_rows: [] }
    const fetchMock = okFetch(summary)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useImportQuestions(), { wrapper })
    const file = new File(['csv'], 'questions.csv', { type: 'text/csv' })
    const out = await result.current.mutateAsync({ file, dryRun: false })
    expect(out).toEqual(summary)
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/questions/import')
    expect(init.method).toBe('POST')
    expect(headers.Authorization).toBe('Bearer tok')
    expect(headers['Content-Type']).toBeUndefined()
    expect(init.body).toBeInstanceOf(FormData)
    const sent = (init.body as FormData).get('file') as File
    expect(sent.name).toBe('questions.csv')
  })

  it('adds dry_run=true to the URL when dryRun is set', async () => {
    const fetchMock = okFetch({ valid_count: 0, error_rows: [], warning_rows: [] })
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useImportQuestions(), { wrapper })
    await result.current.mutateAsync({ file: new File(['csv'], 'q.csv'), dryRun: true })
    expect(firstCall(fetchMock).url).toBe('/api/v1/questions/import?dry_run=true')
  })

  it('rejects with the API error code when the import is rejected', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'UNSUPPORTED_FILE', message: 'xlsx not supported' }, 415))
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useImportQuestions(), { wrapper })
    await expect(
      result.current.mutateAsync({ file: new File(['x'], 'q.pdf'), dryRun: false }),
    ).rejects.toMatchObject({ code: 'UNSUPPORTED_FILE', message: 'xlsx not supported' })
  })
})

describe('useCreateTag', () => {
  it('POSTs the name to the tags URL and returns the created tag', async () => {
    const created = { id: 'tag-9', name: 'fire' }
    const fetchMock = okFetch(created)
    vi.stubGlobal('fetch', fetchMock)
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useCreateTag(), { wrapper })
    const out = await result.current.mutateAsync({ name: 'fire' })
    expect(out).toEqual(created)
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/tags')
    expect(init.method).toBe('POST')
    expect(headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual({ name: 'fire' })
  })

  it('invalidates the tags query after a successful create', async () => {
    vi.stubGlobal('fetch', okFetch({ id: 'tag-9', name: 'fire' }))
    const { qc, wrapper } = makeClient()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useCreateTag(), { wrapper })
    await result.current.mutateAsync({ name: 'fire' })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['tags'] })
  })

  it('rejects with the API error code on a duplicate name', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'TAG_EXISTS', message: 'duplicate' }, 409))
    const { wrapper } = makeClient()
    const { result } = renderHook(() => useCreateTag(), { wrapper })
    await expect(result.current.mutateAsync({ name: 'fire' })).rejects.toMatchObject({ code: 'TAG_EXISTS' })
  })
})

describe('useUpdateQuestionStatus', () => {
  it('POSTs the target status and invalidates the detail and list on settle', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204 })
    vi.stubGlobal('fetch', fetchMock)
    const { qc, wrapper } = makeClient()
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    const { result } = renderHook(() => useUpdateQuestionStatus(), { wrapper })
    await result.current.mutateAsync({ id: 'q-1', target_status: 'active' })
    const { url, init, headers } = firstCall(fetchMock)
    expect(url).toBe('/api/v1/questions/q-1/status')
    expect(init.method).toBe('POST')
    expect(headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body as string)).toEqual({ status: 'active' })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['questions', 'q-1'] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['questions'] })
  })

  it('optimistically sets the status on the detail cache while the request is pending', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(() => new Promise(() => {})),
    )
    const { qc, wrapper } = makeClient()
    qc.setQueryData<QuestionDetail>(['questions', 'q-1'], detail)
    const { result } = renderHook(() => useUpdateQuestionStatus(), { wrapper })
    act(() => {
      result.current.mutate({ id: 'q-1', target_status: 'archived' })
    })
    await waitFor(() => {
      expect(qc.getQueryData<QuestionDetail>(['questions', 'q-1'])?.status).toBe('archived')
    })
  })

  it('restores the previous detail cache when the request fails', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'INVALID_TRANSITION', message: 'not allowed' }, 409))
    const { qc, wrapper } = makeClient()
    qc.setQueryData<QuestionDetail>(['questions', 'q-1'], detail)
    const { result } = renderHook(() => useUpdateQuestionStatus(), { wrapper })
    await expect(result.current.mutateAsync({ id: 'q-1', target_status: 'archived' })).rejects.toMatchObject({
      code: 'INVALID_TRANSITION',
    })
    expect(qc.getQueryData<QuestionDetail>(['questions', 'q-1'])?.status).toBe('draft')
  })

  it('does not create a detail cache entry when none existed before the optimistic update', async () => {
    vi.stubGlobal('fetch', errorFetch({ code: 'INVALID_TRANSITION', message: 'no' }, 409))
    const { qc, wrapper } = makeClient()
    const { result } = renderHook(() => useUpdateQuestionStatus(), { wrapper })
    await expect(result.current.mutateAsync({ id: 'q-9', target_status: 'active' })).rejects.toMatchObject({
      code: 'INVALID_TRANSITION',
    })
    expect(qc.getQueryData(['questions', 'q-9'])).toBeUndefined()
  })
})
