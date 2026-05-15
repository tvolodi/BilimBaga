import { useMutation, useQuery } from '@tanstack/react-query'

// ---- Types ------------------------------------------------------------------

export type QuestionType = 'single_choice' | 'multiple_choice' | 'true_false' | 'likert' | 'short_text'

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

async function apiFetch<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, options)
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
  return useQuery<ResumeSessionResponse, Error>({
    queryKey: ['portal', 'sessions', sessionId],
    queryFn: () => apiFetch<ResumeSessionResponse>(`/api/v1/portal/sessions/${sessionId}`),
    staleTime: Infinity,
    retry: false,
  })
}

export function useSaveAnswer(sessionId: string) {
  return useMutation<SaveAnswerResponse, Error, { questionId: string; answer: SaveAnswerRequest }>({
    mutationFn: ({ questionId, answer }) =>
      apiFetch<SaveAnswerResponse>(
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
  return useMutation<SubmitResult, Error, void>({
    mutationFn: () =>
      apiFetch<SubmitResult>(`/api/v1/portal/sessions/${sessionId}/submit`, {
        method: 'POST',
      }),
  })
}

export function useReportEvent(sessionId: string) {
  return useMutation<EventResponse, Error, EventPayload>({
    mutationFn: (payload) =>
      apiFetch<EventResponse>(`/api/v1/portal/sessions/${sessionId}/events`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      }),
  })
}
