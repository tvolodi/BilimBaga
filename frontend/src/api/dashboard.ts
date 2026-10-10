import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { QueryClient } from '@tanstack/react-query'
import { apiFetch } from './apiFetch'

function apiPost<T>(qc: QueryClient, url: string, payload: unknown): Promise<T> {
  return apiFetch<T>(qc, url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}

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

// ---- Hooks ------------------------------------------------------------------

export function useDashboard() {
  const qc = useQueryClient()
  return useQuery<DashboardMetrics, Error>({
    queryKey: ['dashboard'],
    queryFn: () => apiFetch<DashboardMetrics>(qc, '/api/v1/admin/dashboard'),
    staleTime: 5 * 60 * 1000,
    refetchOnWindowFocus: false,
  })
}

export function useRemindEmployee(
  onSuccess?: () => void,
  onError?: (err: Error) => void,
) {
  const qc = useQueryClient()
  return useMutation<unknown, Error, { userId: string; examId: string }>({
    mutationFn: ({ userId, examId }) =>
      apiPost<{ sent_at: string }>(qc, `/api/v1/admin/users/${userId}/remind`, { exam_id: examId }),
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
