import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query'

// ---- Types ------------------------------------------------------------------

export type QuestionType = 'single_choice' | 'multiple_choice' | 'true_false' | 'likert' | 'short_text' | 'single' | 'multiple' | 'truefalse' | 'shorttext'

export interface QuestionOption {
  id: string
  text: string
}

export interface SessionQuestion {
  id: string
  sort_order: number
  stem: string
  type: QuestionType
  options: QuestionOption[]
}

export interface SavedAnswer {
  selected_option_ids: string[]
  text_answer: string | null
  time_spent_seconds: number
  saved_at: string
}

export interface ResumeSessionResponse {
  session_id: string
  exam_id: string
  exam_title: string
  certificate_enabled: boolean
  status: 'in_progress' | 'submitted' | 'auto_submitted' | 'expired'
  started_at: string
  expires_at: string
  remaining_seconds: number
  questions: SessionQuestion[]
  answers: Record<string, SavedAnswer>
  adaptive: boolean
}

export interface NextQuestionResponse {
  question: SessionQuestion | null
  questions_answered: number
  done: boolean
}

export interface SaveAnswerRequest {
  selected_option_ids: string[]
  text_answer: string | null
  time_spent_seconds: number
}

export interface SaveAnswerResponse {
  question_id: string
  saved_at: string
  remaining_seconds: number
}

export interface SubmitResult {
  session_id: string
  status: string
  submitted_at: string
  score_pct: number | null
  passed: boolean | null
}

export interface SectionScore {
  section_id: string
  title: string
  score_pct: number
}

export interface QuestionBreakdownItem {
  question_id: string
  stem: string
  employee_answer: string[]
  correct_answer: string[]
  points_earned: number
  max_points: number
  explanation: string | null
}

export interface SessionResult {
  session_id: string
  exam_id: string
  exam_title: string
  score_pct: number | null
  passed: boolean
  time_taken_seconds: number | null
  attempt_number: number
  submitted_at: string
  show_answers_mode: 'never' | 'after_completion' | 'after_all_attempts'
  per_section_scores: SectionScore[]
  per_question_breakdown: QuestionBreakdownItem[] | undefined
}

export interface EventPayload {
  type: 'tab_switch' | 'blur' | 'fullscreen_exit'
}

export interface EventResponse {
  warn: boolean
  event_count?: number
  session_id?: string
  status?: string
  score_pct?: number | null
  passed?: boolean | null
}

// ---- API helper -------------------------------------------------------------

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

// ---- Hooks ------------------------------------------------------------------

export function useSession(sessionId: string) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<ResumeSessionResponse, Error>({
    queryKey: ['portal', 'sessions', sessionId],
    queryFn: () => apiFetch<ResumeSessionResponse>(`/api/v1/portal/sessions/${sessionId}`, token),
    staleTime: 30 * 1000, // AC-6: 30 seconds — session state changes frequently
    retry: false,
  })
}

export function useSaveAnswer(sessionId: string) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useMutation<SaveAnswerResponse, Error, { questionId: string; answer: SaveAnswerRequest }>({
    mutationFn: ({ questionId, answer }) =>
      apiFetch<SaveAnswerResponse>(
        `/api/v1/portal/sessions/${sessionId}/answers/${questionId}`,
        token,
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(answer),
        },
      ),
  })
}

export function useSubmitSession(sessionId: string) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useMutation<SubmitResult, Error, void>({
    mutationFn: () =>
      apiFetch<SubmitResult>(`/api/v1/portal/sessions/${sessionId}/submit`, token, {
        method: 'POST',
      }),
  })
}

export function useReportEvent(sessionId: string) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useMutation<EventResponse, Error, EventPayload>({
    mutationFn: (payload) =>
      apiFetch<EventResponse>(`/api/v1/portal/sessions/${sessionId}/events`, token, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      }),
  })
}

export function useSessionResult(sessionId: string) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<SessionResult, Error>({
    queryKey: ['session-result', sessionId],
    queryFn: () => apiFetch<SessionResult>(`/api/v1/portal/sessions/${sessionId}/result`, token),
    staleTime: Infinity,
  })
}

// ---- FR-BB72: Adaptive next question ----------------------------------------

export function useNextAdaptiveQuestion(sessionId: string, enabled: boolean) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<NextQuestionResponse, Error>({
    queryKey: ['session-next-question', sessionId],
    queryFn: () => apiFetch<NextQuestionResponse>(`/api/v1/portal/sessions/${sessionId}/next-question`, token),
    enabled,
    staleTime: 0, // always re-fetch on invalidation
    retry: false,
  })
}

// ---- FR-BB46: My Results ----------------------------------------------------

export interface SessionHistoryItem {
  session_id: string
  exam_id: string
  exam_title: string
  submitted_at: string
  score_pct: number
  passed: boolean
  time_taken_seconds: number | null
  certificate_available: boolean
}

export interface MyResultsMeta {
  page: number
  per_page: number
  total: number
}

export interface MyResultsResponse {
  sessions: SessionHistoryItem[]
  meta: MyResultsMeta
}

export function useMyResults(page: number, sort: 'date' | 'score', dir: 'asc' | 'desc') {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<MyResultsResponse, Error>({
    queryKey: ['my-results', page, sort, dir],
    queryFn: () =>
      apiFetch<MyResultsResponse>(
        `/api/v1/portal/results?page=${page}&sort=${sort}&dir=${dir}&per_page=20`,
        token,
      ),
    placeholderData: keepPreviousData,
  })
}

// ---- FR-BB41: Exam history (latest session redirect) ------------------------

export interface ExamHistorySession {
  session_id: string
  started_at: string
  submitted_at: string | null
  score_pct: number | null
  passed: boolean | null
  status: string
}

export interface ExamHistoryResponse {
  exam_id: string
  exam_title: string
  sessions: ExamHistorySession[]
  meta: { page: number; per_page: number; total: number }
}

export function useExamHistory(examId: string) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<ExamHistoryResponse, Error>({
    queryKey: ['exam-history', examId],
    queryFn: () =>
      apiFetch<ExamHistoryResponse>(
        `/api/v1/portal/exams/${examId}/history?page=1&per_page=1`,
        token,
      ),
    enabled: !!examId,
    staleTime: 30 * 1000,
  })
}
