import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

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

  constructor(
    message: string,
    code: string,
    httpStatus: number,
    fields?: Array<{ field: string; message: string }>,
    unsatisfiedRules?: PublishValidationDetail[],
  ) {
    super(message)
    this.name = 'ExamApiError'
    this.code = code
    this.httpStatus = httpStatus
    this.fields = fields
    this.unsatisfiedRules = unsatisfiedRules
  }
}

// ---- API helper -------------------------------------------------------------

interface ApiErrorBody {
  code: string
  message: string
  fields?: Array<{ field: string; message: string }>
  details?: PublishValidationDetail[]
}

interface ApiResponse<T> {
  data: T
  error: ApiErrorBody | null
}

async function examsFetch<T>(url: string, token?: string | null, options?: RequestInit): Promise<T> {
  const authHeader: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {}
  const res = await fetch(url, {
    ...options,
    credentials: 'include',
    headers: { ...authHeader, ...options?.headers },
  })
  const body: ApiResponse<T> = await res.json()
  if (body.error) {
    throw new ExamApiError(
      body.error.message,
      body.error.code,
      res.status,
      body.error.fields,
      body.error.details,
    )
  }
  if (!res.ok) {
    throw new ExamApiError(`Request failed: ${res.status}`, 'ERR_UNKNOWN', res.status)
  }
  return body.data
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
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      return examsFetch<ExamListResponse>(`/api/v1/exams${qs ? `?${qs}` : ''}`, token)
    },
    staleTime: 5 * 60 * 1000,
  })
}

export function useExam(id: string | null | undefined) {
  const qc = useQueryClient()
  return useQuery<ExamDetail, ExamApiError>({
    queryKey: ['exams', id],
    queryFn: () => {
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      return examsFetch<ExamDetail>(`/api/v1/exams/${id}`, token)
    },
    enabled: !!id,
    staleTime: 30_000,
  })
}

// ---- Mutation hooks ---------------------------------------------------------

export function useCreateExam() {
  const qc = useQueryClient()
  return useMutation<ExamDetail, ExamApiError, CreateExamPayload>({
    mutationFn: (body) => {
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      return examsFetch<ExamDetail>('/api/v1/exams', token, {
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
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      return examsFetch<ExamDetail>(`/api/v1/exams/${id}`, token, {
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
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      return examsFetch<ExamDetail>(`/api/v1/exams/${id}/publish`, token, { method: 'POST' })
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['exams'] })
      qc.invalidateQueries({ queryKey: ['exams', id] })
    },
  })
}

// ---- Section hooks ----------------------------------------------------------

export function useAddSection() {
  const qc = useQueryClient()
  return useMutation<SectionDetail, ExamApiError, { examId: string; title?: string; sort_order: number }>({
    mutationFn: ({ examId, ...body }) => {
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      return examsFetch<SectionDetail>(`/api/v1/exams/${examId}/sections`, token, {
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
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      return examsFetch<QuestionRuleDetail>(`/api/v1/exams/${examId}/rules`, token, {
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
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      return examsFetch<QuestionRuleDetail>(`/api/v1/exams/${examId}/rules/${ruleId}`, token, {
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
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      return examsFetch<void>(`/api/v1/exams/${examId}/rules/${ruleId}`, token, { method: 'DELETE' })
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
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      return examsFetch<void>(`/api/v1/exams/${examId}/rules/${ruleId}/questions`, token, {
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
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      return examsFetch<ExamAssignment[]>(`/api/v1/exams/${examId}/assignments`, token)
    },
    enabled: !!examId,
    staleTime: 30_000,
  })
}

export function useAddAssignment(examId: string) {
  const qc = useQueryClient()
  return useMutation<ExamAssignment, ExamApiError, AddAssignmentPayload>({
    mutationFn: (body) => {
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      return examsFetch<ExamAssignment>(`/api/v1/exams/${examId}/assign`, token, {
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
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      return examsFetch<void>(`/api/v1/exams/${examId}/assign/${assignmentId}`, token, { method: 'DELETE' })
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['exams', examId, 'assignments'] }),
  })
}
