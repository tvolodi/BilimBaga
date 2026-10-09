import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { errorWithCode } from '@/api/errors'

// ---- Types ------------------------------------------------------------------

export interface ExamCompletionRate {
  exam_id: string
  title: string
  assigned_count: number
  completed_count: number
  passed_count: number
}

export interface OverdueEmployee {
  user_id: string
  name: string
  exam_title: string
  exam_id: string
  deadline: string
}

export interface RecentActivity {
  session_id: string
  employee_name: string
  exam_title: string
  score_pct: number | null
  passed: boolean | null
  submitted_at: string
}

export interface DashboardMetrics {
  completion_rate_by_exam: ExamCompletionRate[]
  overdue_employees: OverdueEmployee[]
  recent_activity: RecentActivity[]
  avg_score_by_track: {
    security: number | null
    safety: number | null
    loyalty: number | null
  }
}

// ---- API helpers ------------------------------------------------------------

interface ApiResponse<T> {
  data: T
  error: null | { code: string; message: string }
}

async function apiFetch<T>(url: string, token?: string | null): Promise<T> {
  const res = await fetch(url, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    credentials: 'include',
  })
  const body: ApiResponse<T> = await res.json()
  if (body.error) throw errorWithCode(body.error)
  if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  return body.data
}

async function apiPost<T>(
  url: string,
  payload: unknown,
  token?: string | null,
): Promise<T> {
  const res = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(payload),
    credentials: 'include',
  })
  const body: ApiResponse<T> = await res.json()
  if (body.error) throw errorWithCode(body.error)
  if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  return body.data
}

// ---- Hooks ------------------------------------------------------------------

export function useDashboard() {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<DashboardMetrics, Error>({
    queryKey: ['dashboard'],
    queryFn: () => apiFetch<DashboardMetrics>('/api/v1/admin/dashboard', token),
    staleTime: 5 * 60 * 1000,
    refetchOnWindowFocus: false,
  })
}

export function useRemindEmployee(
  onSuccess?: () => void,
  onError?: (err: Error) => void,
) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useMutation<unknown, Error, { userId: string; examId: string }>({
    mutationFn: ({ userId, examId }) =>
      apiPost<{ sent_at: string }>(`/api/v1/admin/users/${userId}/remind`, { exam_id: examId }, token),
    onSuccess,
    onError,
  })
}

// ---- KPI helpers ------------------------------------------------------------

export function computeKpis(data: DashboardMetrics) {
  const totalAssigned = data.completion_rate_by_exam.reduce(
    (s, e) => s + e.assigned_count,
    0,
  )
  const totalCompleted = data.completion_rate_by_exam.reduce(
    (s, e) => s + e.completed_count,
    0,
  )
  const totalPassed = data.completion_rate_by_exam.reduce(
    (s, e) => s + e.passed_count,
    0,
  )

  const completionRate =
    totalAssigned > 0
      ? ((totalCompleted / totalAssigned) * 100).toFixed(1) + '%'
      : '—'
  const passRate =
    totalAssigned > 0
      ? ((totalPassed / totalAssigned) * 100).toFixed(1) + '%'
      : '—'

  return { completionRate, passRate }
}
