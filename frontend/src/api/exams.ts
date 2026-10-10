import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { QueryClient } from '@tanstack/react-query'
import { apiFetch, type ApiError } from './apiFetch'
import { retryUnlessNotFound } from '@/lib/apiRetry'

// ---- Types ------------------------------------------------------------------

export interface ExamListItem {
  id: string
  title: string
  status: 'draft' | 'active' | 'archived'
  time_limit_minutes: number
  passing_score_pct: number
  max_attempts: number
  available_from: string | null
  available_until: string | null
  created_by: string
  created_at: string
  updated_at: string
}

export interface ExamListResponse {
  items: ExamListItem[]
  meta: { page: number; per_page: number; total: number }
}

export interface ManualQuestionRef {
  question_id: string
  sort_order: number
}

export interface QuestionRuleDetail {
  id: string
  section_id: string | null
  mode: 'manual' | 'random'
  category_id: string | null
  tag_ids: string[]
  difficulty: 'easy' | 'medium' | 'hard' | null
  count: number
  sort_order: number
  questions?: ManualQuestionRef[]
}

export interface SectionDetail {
  id: string
  title: string | null
  sort_order: number
}

export interface ExamDetail {
  id: string
  title: string
  description: string | null
  status: 'draft' | 'active' | 'archived'
  time_limit_minutes: number
  passing_score_pct: number
  max_attempts: number
  available_from: string | null
  available_until: string | null
  shuffle_questions: boolean
  shuffle_options: boolean
  show_answers: 'never' | 'after_completion' | 'after_all_attempts'
  on_tab_switch: 'log' | 'warn' | 'submit'
  certificate_enabled: boolean
  created_by: string
  created_at: string
  updated_at: string
  sections: SectionDetail[]
  rules: QuestionRuleDetail[]
}

export interface CreateExamPayload {
  title: string
  description?: string | null
  time_limit_minutes: number
  passing_score_pct: number
  max_attempts: number
  available_from?: string | null
  available_until?: string | null
  shuffle_questions: boolean
  shuffle_options: boolean
  show_answers: string
  on_tab_switch: string
  certificate_enabled: boolean
}

export interface QuestionRulePayload {
  section_id?: string | null
  mode: 'manual' | 'random'
  category_id?: string | null
  tag_ids: string[]
  difficulty?: string | null
  count: number
  sort_order: number
}

export interface ExamAssignment {
  id: string
  exam_id: string
  assignee_type: 'user' | 'department' | 'all'
  assignee_id: string | null
  deadline: string | null
  assigned_by: string
  assigned_at: string
}

export interface AddAssignmentPayload {
  assignee_type: 'user' | 'department' | 'all'
  assignee_id?: string | null
  deadline?: string | null
}

export interface PublishValidationDetail {
  rule_id: string
  required: number
  available: number
  filter: Record<string, unknown>
}

// ---- API error class --------------------------------------------------------

export class ExamApiError extends Error {
  code: string
  httpStatus: number
  fields?: Array<{ field: string; message: string }>
  unsatisfiedRules?: PublishValidationDetail[]
  /** Raw `error.details` from the backend: array, object, string or absent. Never assume a shape. */
  details?: unknown

  constructor(
    message: string,
    code: string,
    httpStatus: number,
    fields?: Array<{ field: string; message: string }>,
    unsatisfiedRules?: PublishValidationDetail[],
    details?: unknown,
  ) {
    super(message)
    this.name = 'ExamApiError'
    this.code = code
    this.httpStatus = httpStatus
    this.fields = fields
    this.unsatisfiedRules = unsatisfiedRules
    this.details = details
  }
}

// ---- API helper -------------------------------------------------------------

/**
 * Exam calls go through the shared apiFetch (auth, TOKEN_REVOKED handling). This maps its error onto
 * ExamApiError, which the exam pages read for httpStatus, fields and unsatisfied publish rules.
 */
async function examsFetch<T>(qc: QueryClient, url: string, options?: RequestInit): Promise<T> {
  try {
    return await apiFetch<T>(qc, url, options)
  } catch (err) {
    const e = err as ApiError
    if (!e.code) throw err
    throw new ExamApiError(
      e.message,
      e.code,
      e.status ?? 0,
      e.fields,
      Array.isArray(e.details) ? (e.details as PublishValidationDetail[]) : undefined,
      e.details,
    )
  }
}

// ---- Exam list & detail hooks -----------------------------------------------

export function useExams(filters: { page?: number; per_page?: number; status?: string; search?: string } = {}) {
  const qc = useQueryClient()
  const params = new URLSearchParams()
  if (filters.page) params.set('page', String(filters.page))
  if (filters.per_page) params.set('per_page', String(filters.per_page))
  if (filters.status) params.set('status', filters.status)
  if (filters.search) params.set('search', filters.search)
  const qs = params.toString()

  return useQuery<ExamListResponse, ExamApiError>({
    queryKey: ['exams', filters],
    queryFn: () => {
      return examsFetch<ExamListResponse>(qc, `/api/v1/exams${qs ? `?${qs}` : ''}`)
    },
    staleTime: 5 * 60 * 1000,
  })
}

export function useExam(id: string | null | undefined) {
  const qc = useQueryClient()
  return useQuery<ExamDetail, ExamApiError>({
    queryKey: ['exams', id],
    queryFn: () => {
      return examsFetch<ExamDetail>(qc, `/api/v1/exams/${id}`)
    },
    enabled: !!id,
    staleTime: 30_000,
    retry: retryUnlessNotFound,
  })
}

// ---- Mutation hooks ---------------------------------------------------------

export function useCreateExam() {
  const qc = useQueryClient()
  return useMutation<ExamDetail, ExamApiError, CreateExamPayload>({
    mutationFn: (body) => {
      return examsFetch<ExamDetail>(qc, '/api/v1/exams', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['exams'] }),
  })
}

export function useUpdateExam(id: string) {
  const qc = useQueryClient()
  return useMutation<ExamDetail, ExamApiError, CreateExamPayload>({
    mutationFn: (body) => {
      return examsFetch<ExamDetail>(qc, `/api/v1/exams/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['exams'] })
      qc.invalidateQueries({ queryKey: ['exams', id] })
    },
  })
}

export function usePublishExam(id: string) {
  const qc = useQueryClient()
  return useMutation<ExamDetail, ExamApiError, void>({
    mutationFn: () => {
      return examsFetch<ExamDetail>(qc, `/api/v1/exams/${id}/publish`, { method: 'POST' })
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['exams'] })
      qc.invalidateQueries({ queryKey: ['exams', id] })
    },
  })
}

export function useUnpublishExam() {
  const qc = useQueryClient()
  return useMutation<{ id: string; status: string }, ExamApiError, string>({
    mutationFn: (examId) => {
      return examsFetch<{ id: string; status: string }>(
        qc,
        `/api/v1/exams/${examId}/unpublish`,
        { method: 'POST' },
      )
    },
    onSuccess: (_data, examId) => {
      qc.invalidateQueries({ queryKey: ['exams'] })
      qc.invalidateQueries({ queryKey: ['exams', examId] })
    },
  })
}

export function useArchiveExam() {
  const qc = useQueryClient()
  return useMutation<{ id: string; status: string }, ExamApiError, string>({
    mutationFn: (examId) => {
      return examsFetch<{ id: string; status: string }>(
        qc,
        `/api/v1/exams/${examId}/archive`,
        { method: 'POST' },
      )
    },
    onSuccess: (_data, examId) => {
      qc.invalidateQueries({ queryKey: ['exams'] })
      qc.invalidateQueries({ queryKey: ['exams', examId] })
    },
  })
}

// ---- Section hooks ----------------------------------------------------------

export function useAddSection() {
  const qc = useQueryClient()
  return useMutation<SectionDetail, ExamApiError, { examId: string; title?: string; sort_order: number }>({
    mutationFn: ({ examId, ...body }) => {
      return examsFetch<SectionDetail>(qc, `/api/v1/exams/${examId}/sections`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
    },
    onSuccess: (_, { examId }) => qc.invalidateQueries({ queryKey: ['exams', examId] }),
  })
}

// ---- Rule hooks -------------------------------------------------------------

export function useAddRule(examId: string) {
  const qc = useQueryClient()
  return useMutation<QuestionRuleDetail, ExamApiError, QuestionRulePayload>({
    mutationFn: (body) => {
      return examsFetch<QuestionRuleDetail>(qc, `/api/v1/exams/${examId}/rules`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['exams', examId] }),
  })
}

export function useUpdateRule(examId: string) {
  const qc = useQueryClient()
  return useMutation<QuestionRuleDetail, ExamApiError, { ruleId: string; body: QuestionRulePayload }>({
    mutationFn: ({ ruleId, body }) => {
      return examsFetch<QuestionRuleDetail>(qc, `/api/v1/exams/${examId}/rules/${ruleId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['exams', examId] }),
  })
}

export function useDeleteRule(examId: string) {
  const qc = useQueryClient()
  return useMutation<void, ExamApiError, string>({
    mutationFn: (ruleId) => {
      return examsFetch<void>(qc, `/api/v1/exams/${examId}/rules/${ruleId}`, { method: 'DELETE' })
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['exams', examId] }),
  })
}

export function useSetRuleQuestions(examId: string) {
  const qc = useQueryClient()
  return useMutation<
    void,
    ExamApiError,
    { ruleId: string; questions: Array<{ question_id: string; sort_order: number }> }
  >({
    mutationFn: ({ ruleId, questions }) => {
      return examsFetch<void>(qc, `/api/v1/exams/${examId}/rules/${ruleId}/questions`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ questions }),
      })
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['exams', examId] }),
  })
}

// ---- Assignment hooks -------------------------------------------------------

export function useExamAssignments(examId: string | null | undefined) {
  const qc = useQueryClient()
  return useQuery<ExamAssignment[], ExamApiError>({
    queryKey: ['exams', examId, 'assignments'],
    queryFn: () => {
      return examsFetch<ExamAssignment[]>(qc, `/api/v1/exams/${examId}/assignments`)
    },
    enabled: !!examId,
    staleTime: 30_000,
  })
}

export function useAddAssignment(examId: string) {
  const qc = useQueryClient()
  return useMutation<ExamAssignment, ExamApiError, AddAssignmentPayload>({
    mutationFn: (body) => {
      return examsFetch<ExamAssignment>(qc, `/api/v1/exams/${examId}/assign`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['exams', examId, 'assignments'] }),
  })
}

export function useDeleteAssignment(examId: string) {
  const qc = useQueryClient()
  return useMutation<void, ExamApiError, string>({
    mutationFn: (assignmentId) => {
      return examsFetch<void>(qc, `/api/v1/exams/${examId}/assign/${assignmentId}`, { method: 'DELETE' })
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['exams', examId, 'assignments'] }),
  })
}

// ---- Eligible counts hook (FR-BB315) ----------------------------------------

export interface RuleEligibleCount {
  rule_id: string
  eligible: number
}

export function useEligibleCounts(examId: string) {
  const qc = useQueryClient()
  return useQuery<{ counts: RuleEligibleCount[] }, ExamApiError>({
    queryKey: ['exams', examId, 'eligibleCounts'],
    queryFn: () => {
      return examsFetch<{ counts: RuleEligibleCount[] }>(
        qc,
        `/api/v1/exams/${examId}/rules/eligible-counts`,
      )
    },
    enabled: !!examId,
    staleTime: 0,
  })
}
