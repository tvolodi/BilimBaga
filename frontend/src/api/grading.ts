import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

// ---- Types ------------------------------------------------------------------

export interface GradingQueueItem {
  session_id: string
  employee_name: string
  exam_id: string
  exam_title: string
  submitted_at: string
  pending_question_count: number
}

export interface GradingQuestion {
  question_id: string
  stem: string
  text_answer: string
  grading_status: 'pending_manual' | 'graded' | 'ai_graded'
  current_score_pct: number | null
  manual_feedback: string | null
  ai_reasoning: string | null
}

export interface GradingSessionDetail {
  session_id: string
  employee_name: string
  exam_title: string
  submitted_at: string | null
  questions: GradingQuestion[]
}

export interface GradeSubmission {
  questionId: string
  scorePct: number
  feedback: string
}

export interface GradingQueueResponse {
  items: GradingQueueItem[]
  meta: { page: number; per_page: number; total: number }
}

export interface GradeAnswerResponse {
  question_id: string
  grading_status: string
  score_pct: number
  session_status: string
  all_graded: boolean
  final_score_pct?: number | null
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

function apiGet<T>(url: string): Promise<T> {
  return apiFetch<T>(`/api/v1${url}`)
}

function apiPost<T>(url: string, payload: unknown): Promise<T> {
  return apiFetch<T>(`/api/v1${url}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}

// ---- Hooks ------------------------------------------------------------------

export function useGradingQueue(page: number, examId?: string) {
  return useQuery({
    queryKey: ['grading-queue', page, examId],
    queryFn: () =>
      apiGet<GradingQueueResponse>(
        `/admin/grading?page=${page}&per_page=20${examId ? `&exam_id=${examId}` : ''}`,
      ),
    staleTime: 0, // AC-6: always fresh — grading queue must reflect current state
  })
}

export function useGradingSession(sessionId: string) {
  return useQuery({
    queryKey: ['grading-session', sessionId],
    queryFn: () => apiGet<GradingSessionDetail>(`/admin/grading/${sessionId}`),
  })
}

export function useSubmitGrade(sessionId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ questionId, scorePct, feedback }: GradeSubmission) =>
      apiPost<GradeAnswerResponse>(`/admin/grading/${sessionId}/answers/${questionId}`, {
        score_pct: scorePct,
        feedback,
      }),
    onSuccess: (data: GradeAnswerResponse) => {
      queryClient.invalidateQueries({ queryKey: ['grading-session', sessionId] })
      if (data.all_graded) {
        queryClient.invalidateQueries({ queryKey: ['grading-queue'] })
      }
    },
  })
}
