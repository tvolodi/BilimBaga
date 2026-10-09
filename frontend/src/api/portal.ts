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
  available_from?: string | null
  available_until?: string | null
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

export interface PortalApiError extends Error {
  code: string
}

async function apiFetch<T>(url: string, token?: string | null, options?: RequestInit): Promise<T> {
  const authHeader: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {}
  const res = await fetch(url, {
    ...options,
    credentials: 'include',
    headers: { ...authHeader, ...(options?.headers as Record<string, string>) },
  })
  // A proxy/gateway error (502, HTML body) must still surface as a normal Error, not a JSON SyntaxError.
  let body: ApiResponse<T> | null = null
  try {
    body = (await res.json()) as ApiResponse<T>
  } catch {
    body = null
  }
  if (body?.error) {
    const err = new Error(body.error.message) as PortalApiError
    err.code = body.error.code
    throw err
  }
  if (!res.ok || !body) {
    const err = new Error(`Request failed: ${res.status}`) as PortalApiError
    err.code = 'ERR_HTTP'
    throw err
  }
  return body.data
}

// ---- Hooks ------------------------------------------------------------------

export function usePortalExams() {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<PortalExam[], Error>({
    queryKey: ['portal', 'exams'],
    queryFn: () => apiFetch<PortalExam[]>('/api/v1/portal/exams', token),
    refetchInterval: 30_000,
  })
}

export function usePortalExam(examId: string | undefined) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<PortalExamDetail, Error>({
    queryKey: ['portal', 'exams', examId],
    queryFn: () => apiFetch<PortalExamDetail>(`/api/v1/portal/exams/${examId}`, token),
    staleTime: Infinity,
    enabled: !!examId,
  })
}

export function useCreateSession(examId: string) {
  const queryClient = useQueryClient()
  return useMutation<CreateSessionResponse, Error, void>({
    mutationFn: () => {
      const token = queryClient.getQueryData<string | null>(['auth', 'accessToken'])
      return apiFetch<CreateSessionResponse>(`/api/v1/portal/exams/${examId}/sessions`, token, {
        method: 'POST',
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['portal', 'exams'] })
    },
  })
}
