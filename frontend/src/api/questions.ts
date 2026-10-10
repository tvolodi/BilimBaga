import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiFetch } from './apiFetch'

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
  locale?: string
  locale_missing?: string
  include_versions?: boolean
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
  if (filters.locale) params.set('locale', filters.locale)
  if (filters.include_versions) params.set('include_versions', 'true')
  if (filters.locale_missing) params.set('locale_missing', filters.locale_missing)
  if (filters.search) params.set('search', filters.search)
  const qs = params.toString()
  return qs ? `?${qs}` : ''
}

// ---- Hooks ------------------------------------------------------------------

export function useQuestions(filters: QuestionFilters = {}) {
  const qc = useQueryClient()
  return useQuery<QuestionListResponse, Error>({
    queryKey: ['questions', filters],
    queryFn: () =>
      apiFetch<QuestionListResponse>(qc, `/api/v1/questions${buildQuestionsQuery(filters)}`),
    staleTime: 30_000,
    refetchOnWindowFocus: false,
  })
}

export function useCategories() {
  const qc = useQueryClient()
  return useQuery<Category[], Error>({
    queryKey: ['categories'],
    queryFn: () => apiFetch<Category[]>(qc, '/api/v1/categories'),
    staleTime: 300_000,
  })
}

export function useTags() {
  const qc = useQueryClient()
  return useQuery<Tag[], Error>({
    queryKey: ['tags'],
    queryFn: () => apiFetch<Tag[]>(qc, '/api/v1/tags'),
    staleTime: 300_000,
  })
}

export function useQuestionVersions(id: string) {
  const qc = useQueryClient()
  return useQuery<QuestionVersion[], Error>({
    queryKey: ['questions', id, 'versions'],
    queryFn: () =>
      apiFetch<QuestionVersion[]>(qc, `/api/v1/questions/${id}/versions`),
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
      return apiFetch<void>(queryClient, `/api/v1/questions/${id}/status`, {
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
      return apiFetch<void>(queryClient, `/api/v1/questions/${id}`, { method: 'DELETE' })
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
      const formData = new FormData()
      formData.append('file', file)
      return apiFetch<ImportDryRunResult>(
        qc,
        `/api/v1/questions/import${dryRun ? '?dry_run=true' : ''}`,
        { method: 'POST', body: formData },
      )
    },
  })
}

export function useQuestion(id: string | null) {
  const qc = useQueryClient()
  return useQuery<QuestionDetail, Error>({
    queryKey: ['questions', id],
    queryFn: () => apiFetch<QuestionDetail>(qc, `/api/v1/questions/${id}`),
    enabled: id !== null && id !== undefined && id !== '',
    staleTime: 0,
  })
}

export function useCreateQuestion() {
  const queryClient = useQueryClient()
  return useMutation<QuestionDetail, Error, QuestionCreatePayload>({
    mutationFn: (payload) => {
      return apiFetch<QuestionDetail>(queryClient, '/api/v1/questions', {
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
      return apiFetch<QuestionDetail>(queryClient, `/api/v1/questions/${id}`, {
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
      return apiFetch<void>(queryClient, `/api/v1/questions/${id}/status`, {
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
      return apiFetch<Tag>(queryClient, '/api/v1/tags', {
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
