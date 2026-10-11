import { useMutation, useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { apiFetch } from './apiFetch'
import { retryUnlessNotFound } from '@/lib/apiRetry'

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
  manual_feedback: string | null
  // FR-BB322 AC-8: a question still waiting for grading shows no answer and no explanation.
  grading_status: 'graded' | 'pending_manual'
}

export interface SessionResult {
  session_id: string
  exam_id: string
  exam_title: string
  status: string
  score_pct: number | null
  passed: boolean
  time_taken_seconds: number | null
  attempt_number: number
  submitted_at: string
  show_answers_mode: 'never' | 'after_completion' | 'after_all_attempts'
  per_section_scores: SectionScore[]
  per_question_breakdown: QuestionBreakdownItem[] | undefined
  // FR-BB322 AC-4: answers still waiting, the short-text answers of the session, and who grades the waiting ones.
  pending_question_count: number
  total_gradable_count: number
  grading_expected_by: 'ai' | 'examiner' | null
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

// ---- Hooks ------------------------------------------------------------------

export function useSession(sessionId: string) {
  const qc = useQueryClient()
  return useQuery<ResumeSessionResponse, Error>({
    queryKey: ['portal', 'sessions', sessionId],
    queryFn: () => apiFetch<ResumeSessionResponse>(qc, `/api/v1/portal/sessions/${sessionId}`),
    staleTime: 30 * 1000, // AC-6: 30 seconds — session state changes frequently
    retry: false,
  })
}

export function useSaveAnswer(sessionId: string) {
  const qc = useQueryClient()
  return useMutation<SaveAnswerResponse, Error, { questionId: string; answer: SaveAnswerRequest }>({
    mutationFn: ({ questionId, answer }) =>
      apiFetch<SaveAnswerResponse>(
        qc,
        `/api/v1/portal/sessions/${sessionId}/answers/${questionId}`,
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
  return useMutation<SubmitResult, Error, void>({
    mutationFn: () =>
      apiFetch<SubmitResult>(qc, `/api/v1/portal/sessions/${sessionId}/submit`, {
        method: 'POST',
      }),
  })
}

export function useReportEvent(sessionId: string) {
  const qc = useQueryClient()
  return useMutation<EventResponse, Error, EventPayload>({
    mutationFn: (payload) =>
      apiFetch<EventResponse>(qc, `/api/v1/portal/sessions/${sessionId}/events`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      }),
  })
}

// FR-BB322 AC-6: while a result waits for grading, the page polls every 10 s, does not poll in a hidden tab, and stops
// after 15 minutes. Refresh fetches once and starts the 15 minutes again.
export const RESULT_POLL_INTERVAL_MS = 10_000
export const RESULT_POLL_LIMIT_MS = 15 * 60_000

export function useSessionResult(sessionId: string) {
  const qc = useQueryClient()
  const [armedAt, setArmedAt] = useState(() => Date.now())
  const [stopped, setStopped] = useState(false)

  const query = useQuery<SessionResult, Error>({
    queryKey: ['session-result', sessionId],
    queryFn: () => apiFetch<SessionResult>(qc, `/api/v1/portal/sessions/${sessionId}/result`),
    // A waiting result is stale at once, so it is fetched again; a final result never goes stale.
    staleTime: (q) => (q.state.data?.status === 'grading_pending' ? 0 : Infinity),
    refetchInterval: (q) =>
      !stopped && q.state.data?.status === 'grading_pending' ? RESULT_POLL_INTERVAL_MS : false,
    refetchIntervalInBackground: false,
    retry: false,
  })

  const waiting = query.data?.status === 'grading_pending'
  useEffect(() => {
    if (!waiting || stopped) return
    const timer = setTimeout(() => setStopped(true), Math.max(0, armedAt + RESULT_POLL_LIMIT_MS - Date.now()))
    return () => clearTimeout(timer)
  }, [waiting, stopped, armedAt])

  function refresh() {
    setStopped(false)
    setArmedAt(Date.now())
    return query.refetch()
  }

  return { ...query, pollingStopped: waiting && stopped, refresh }
}

// ---- FR-BB72: Adaptive next question ----------------------------------------

export function useNextAdaptiveQuestion(sessionId: string, enabled: boolean) {
  const qc = useQueryClient()
  return useQuery<NextQuestionResponse, Error>({
    queryKey: ['session-next-question', sessionId],
    queryFn: () => apiFetch<NextQuestionResponse>(qc, `/api/v1/portal/sessions/${sessionId}/next-question`),
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
  // Null while the session waits for grading (FR-BB322 AC-13).
  score_pct: number | null
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
  return useQuery<MyResultsResponse, Error>({
    queryKey: ['my-results', page, sort, dir],
    queryFn: () =>
      apiFetch<MyResultsResponse>(
        qc,
        `/api/v1/portal/results?page=${page}&sort=${sort}&dir=${dir}&per_page=20`,
      ),
    placeholderData: keepPreviousData,
    retry: false,
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

// perPage defaults to 1 (the latest-session lookup of old). The result redirect asks for every attempt (AC-14). The
// query key keeps the ['exam-history', examId] prefix, so invalidating by that prefix still matches (AC-7).
export function useExamHistory(examId: string, perPage = 1) {
  const qc = useQueryClient()
  return useQuery<ExamHistoryResponse, Error>({
    queryKey: ['exam-history', examId, perPage],
    queryFn: () =>
      apiFetch<ExamHistoryResponse>(
        qc,
        `/api/v1/portal/exams/${examId}/history?page=1&per_page=${perPage}`,
      ),
    enabled: !!examId,
    staleTime: 30 * 1000,
    retry: retryUnlessNotFound,
  })
}
