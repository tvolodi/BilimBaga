import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

// ---- Types ------------------------------------------------------------------

export interface QuestionListItem {
  id: string
  type: 'single' | 'multiple' | 'truefalse' | 'shorttext' | 'likert'
  difficulty: 'easy' | 'medium' | 'hard'
  status: 'draft' | 'review' | 'active' | 'archived'
  category_id: string
  category_name: string
  default_locale: string
  version: number
  locale_coverage: string[]
  stem_preview: string
  tags: string[]
  created_by: string
  created_by_name: string
  created_at: string
  updated_at: string
}

export interface QuestionFilters {
  page?: number
  per_page?: number
  sort?: 'created_at' | 'updated_at' | 'difficulty'
  order?: 'asc' | 'desc'
  category_id?: string
  tag_ids?: string[]
  statuses?: Array<'draft' | 'review' | 'active' | 'archived'>
  difficulties?: Array<'easy' | 'medium' | 'hard'>
  type?: 'single' | 'multiple' | 'truefalse' | 'shorttext' | 'likert'
  locale_missing?: string
  search?: string
}

export interface QuestionListMeta {
  page: number
  per_page: number
  total: number
}

export interface QuestionListResponse {
  items: QuestionListItem[]
  meta: QuestionListMeta
}

export interface Category {
  id: string
  name: string
  parent_id: string | null
  children?: Category[]
}

export interface Tag {
  id: string
  name: string
}

export interface QuestionTranslation {
  stem: string
  explanation: string
}

export interface AnswerOptionTranslation {
  text: string
}

export interface AnswerOption {
  id: string
  sort_order: number
  is_correct: boolean
  likert_weight: number | null
  likert_polarity: string | null
  translations: Record<string, AnswerOptionTranslation>
}

export interface QuestionDetail {
  id: string
  type: 'single' | 'multiple' | 'truefalse' | 'shorttext' | 'likert'
  difficulty: 'easy' | 'medium' | 'hard'
  status: 'draft' | 'review' | 'active' | 'archived'
  category_id: string
  category_name: string
  default_locale: string
  version: number
  parent_id: string | null
  auto_grade: boolean
  model_answer: string | null
  locale_coverage: string[]
  translations: Record<string, QuestionTranslation>
  answer_options: AnswerOption[]
  tag_ids: string[]
  created_by: string
  created_by_name: string
  created_at: string
  updated_at: string
}

export interface QuestionCreatePayload {
  type: string
  difficulty: string
  category_id: string
  default_locale: string
  auto_grade?: boolean
  model_answer?: string | null
  translations: Record<string, { stem: string; explanation?: string }>
  answer_options?: AnswerOptionCreatePayload[]
  tag_ids?: string[]
}

export interface AnswerOptionCreatePayload {
  sort_order: number
  is_correct: boolean
  likert_weight?: number
  likert_polarity?: string
  translations: Record<string, { text: string }>
}

export interface QuestionUpdatePayload {
  difficulty?: string
  category_id?: string
  auto_grade?: boolean
  model_answer?: string | null
  translations?: Record<string, Partial<QuestionTranslation>>
  answer_options?: AnswerOptionUpdatePayload[]
  tag_ids?: string[]
}

export interface AnswerOptionUpdatePayload extends AnswerOptionCreatePayload {
  id?: string
}

export interface QuestionVersion {
  version: number
  created_at: string
  created_by_name: string
  change_summary: string
}

export interface ImportDryRunResult {
  valid_count: number
  error_rows: ImportErrorRow[]
  warning_rows: ImportWarningRow[]
}

export interface ImportErrorRow {
  row: number
  errors: string[]
}

export interface ImportWarningRow {
  row: number
  similarity_match: {
    question_id: string
    score: number
    stem_preview: string
  }
}

// ---- API helpers ------------------------------------------------------------

interface ApiResponse<T> {
  data: T
  error: null | { code: string; message: string }
}

async function apiFetch<T>(url: string, token?: string | null, options?: RequestInit): Promise<T> {
  const authHeader: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {}
  const res = await fetch(url, {
    ...options,
    credentials: 'include',
    headers: { ...authHeader, ...(options?.headers as Record<string, string>) },
  })
  const body: ApiResponse<T> = await res.json()
  if (body.error) {
    throw new Error(body.error.message)
  }
  if (!res.ok) {
    throw new Error(`Request failed: ${res.status}`)
  }
  return body.data
}

function buildQuestionsQuery(filters: QuestionFilters): string {
  const params = new URLSearchParams()
  if (filters.page) params.set('page', String(filters.page))
  if (filters.per_page) params.set('per_page', String(filters.per_page))
  if (filters.sort) params.set('sort', filters.sort)
  if (filters.order) params.set('order', filters.order)
  if (filters.category_id) params.set('category_id', filters.category_id)
  if (filters.tag_ids?.length) params.set('tag_ids', filters.tag_ids.join(','))
  if (filters.statuses?.length) params.set('statuses', filters.statuses.join(','))
  if (filters.difficulties?.length) params.set('difficulties', filters.difficulties.join(','))
  if (filters.type) params.set('type', filters.type)
  if (filters.locale_missing) params.set('locale_missing', filters.locale_missing)
  if (filters.search) params.set('search', filters.search)
  const qs = params.toString()
  return qs ? `?${qs}` : ''
}

// ---- Hooks ------------------------------------------------------------------

export function useQuestions(filters: QuestionFilters = {}) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<QuestionListResponse, Error>({
    queryKey: ['questions', filters],
    queryFn: () =>
      apiFetch<QuestionListResponse>(`/api/v1/questions${buildQuestionsQuery(filters)}`, token),
    staleTime: 30_000,
    refetchOnWindowFocus: false,
  })
}

export function useCategories() {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<Category[], Error>({
    queryKey: ['categories'],
    queryFn: () => apiFetch<Category[]>('/api/v1/categories', token),
    staleTime: 300_000,
  })
}

export function useTags() {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<Tag[], Error>({
    queryKey: ['tags'],
    queryFn: () => apiFetch<Tag[]>('/api/v1/tags', token),
    staleTime: 300_000,
  })
}

export function useQuestionVersions(id: string) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<QuestionVersion[], Error>({
    queryKey: ['questions', id, 'versions'],
    queryFn: () =>
      apiFetch<QuestionVersion[]>(`/api/v1/questions/${id}/versions`, token),
    staleTime: 0,
    enabled: !!id,
  })
}

export function useTransitionStatus() {
  const queryClient = useQueryClient()
  return useMutation<
    void,
    Error,
    { id: string; target_status: string },
    { previousQueries: [readonly unknown[], unknown][] }
  >({
    mutationFn: ({ id, target_status }) => {
      const token = queryClient.getQueryData<string | null>(['auth', 'accessToken'])
      return apiFetch<void>(`/api/v1/questions/${id}/status`, token, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status: target_status }),
      })
    },
    onMutate: async ({ id, target_status }) => {
      await queryClient.cancelQueries({ queryKey: ['questions'] })
      const previousQueries = queryClient.getQueriesData<QuestionListResponse>({
        queryKey: ['questions'],
      })
      queryClient.setQueriesData<QuestionListResponse>(
        { queryKey: ['questions'] },
        (old) => {
          if (!old?.items) return old
          return {
            ...old,
            items: old.items.map((item) =>
              item.id === id
                ? { ...item, status: target_status as QuestionListItem['status'] }
                : item,
            ),
          }
        },
      )
      return { previousQueries }
    },
    onError: (_, __, context) => {
      if (context?.previousQueries) {
        for (const [key, data] of context.previousQueries) {
          queryClient.setQueryData(key as readonly unknown[], data)
        }
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['questions'] })
    },
  })
}

export function useDeleteQuestion() {
  const queryClient = useQueryClient()
  return useMutation<void, Error, string>({
    mutationFn: (id) => {
      const token = queryClient.getQueryData<string | null>(['auth', 'accessToken'])
      return apiFetch<void>(`/api/v1/questions/${id}`, token, { method: 'DELETE' })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['questions'] })
    },
  })
}

export function useImportQuestions() {
  const qc = useQueryClient()
  return useMutation<ImportDryRunResult, Error, { file: File; dryRun: boolean }>({
    mutationFn: ({ file, dryRun }) => {
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      const formData = new FormData()
      formData.append('file', file)
      return apiFetch<ImportDryRunResult>(
        `/api/v1/questions/import${dryRun ? '?dry_run=true' : ''}`,
        token,
        { method: 'POST', body: formData },
      )
    },
  })
}

export function useQuestion(id: string | null) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<QuestionDetail, Error>({
    queryKey: ['questions', id],
    queryFn: () => apiFetch<QuestionDetail>(`/api/v1/questions/${id}`, token),
    enabled: id !== null && id !== undefined && id !== '',
    staleTime: 0,
  })
}

export function useCreateQuestion() {
  const queryClient = useQueryClient()
  return useMutation<QuestionDetail, Error, QuestionCreatePayload>({
    mutationFn: (payload) => {
      const token = queryClient.getQueryData<string | null>(['auth', 'accessToken'])
      return apiFetch<QuestionDetail>('/api/v1/questions', token, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['questions'] })
    },
  })
}

export function useUpdateQuestion() {
  const queryClient = useQueryClient()
  return useMutation<QuestionDetail, Error, { id: string; payload: QuestionUpdatePayload }>({
    mutationFn: ({ id, payload }) => {
      const token = queryClient.getQueryData<string | null>(['auth', 'accessToken'])
      return apiFetch<QuestionDetail>(`/api/v1/questions/${id}`, token, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
    },
    onSuccess: (result, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['questions', id] })
      // When an active question is updated a new version is created with a different ID.
      // Invalidate the new question's cache so it is fresh when navigating to it.
      if (result?.id && result.id !== id) {
        queryClient.invalidateQueries({ queryKey: ['questions', result.id] })
      }
      queryClient.invalidateQueries({ queryKey: ['questions'] })
    },
  })
}

export function useUpdateQuestionStatus() {
  const queryClient = useQueryClient()
  return useMutation<
    void,
    Error,
    { id: string; target_status: string },
    { previousDetail: QuestionDetail | undefined }
  >({
    mutationFn: ({ id, target_status }) => {
      const token = queryClient.getQueryData<string | null>(['auth', 'accessToken'])
      return apiFetch<void>(`/api/v1/questions/${id}/status`, token, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status: target_status }),
      })
    },
    onMutate: async ({ id, target_status }) => {
      await queryClient.cancelQueries({ queryKey: ['questions', id] })
      const previousDetail = queryClient.getQueryData<QuestionDetail>(['questions', id])
      queryClient.setQueryData<QuestionDetail>(['questions', id], (old) => {
        if (!old) return old
        return { ...old, status: target_status as QuestionDetail['status'] }
      })
      return { previousDetail }
    },
    onError: (_, { id }, context) => {
      if (context?.previousDetail) {
        queryClient.setQueryData(['questions', id], context.previousDetail)
      }
    },
    onSettled: (_, __, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['questions', id] })
      queryClient.invalidateQueries({ queryKey: ['questions'] })
    },
  })
}

export function useCreateTag() {
  const queryClient = useQueryClient()
  return useMutation<Tag, Error, { name: string }>({
    mutationFn: (payload) => {
      const token = queryClient.getQueryData<string | null>(['auth', 'accessToken'])
      return apiFetch<Tag>('/api/v1/tags', token, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tags'] })
    },
  })
}
