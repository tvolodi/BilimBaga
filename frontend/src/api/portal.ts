import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

// ---- Types ------------------------------------------------------------------

export type UserStatus = 'not_started' | 'in_progress' | 'passed' | 'failed' | 'expired'
export type ShowAnswers = 'never' | 'after_completion' | 'after_all_attempts'

export interface PortalExam {
  id: string
  title: string
  description: string | null
  time_limit_minutes: number
  passing_score_pct: number
  max_attempts: number
  attempts_used: number
  deadline: string | null
  user_status: UserStatus
  open_session_id: string | null
  show_answers: ShowAnswers
  shuffle_questions: boolean
  shuffle_options: boolean
}

export interface PortalExamDetail extends PortalExam {
  certificate_enabled: boolean
}

export interface SessionQuestion {
  id: string
  [key: string]: unknown
}

export interface CreateSessionResponse {
  session_id: string
  exam_id: string
  started_at: string
  expires_at: string
  remaining_seconds: number
  questions: SessionQuestion[]
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

export function usePortalExams() {
  return useQuery<PortalExam[], Error>({
    queryKey: ['portal', 'exams'],
    queryFn: () => apiFetch<PortalExam[]>('/api/v1/portal/exams'),
    refetchInterval: 30_000,
  })
}

export function usePortalExam(examId: string | undefined) {
  return useQuery<PortalExamDetail, Error>({
    queryKey: ['portal', 'exams', examId],
    queryFn: () => apiFetch<PortalExamDetail>(`/api/v1/portal/exams/${examId}`),
    staleTime: Infinity,
    enabled: !!examId,
  })
}

export function useCreateSession(examId: string) {
  const queryClient = useQueryClient()
  return useMutation<CreateSessionResponse, Error, void>({
    mutationFn: () =>
      apiFetch<CreateSessionResponse>(`/api/v1/portal/exams/${examId}/sessions`, {
        method: 'POST',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portal', 'exams'] })
    },
  })
}
