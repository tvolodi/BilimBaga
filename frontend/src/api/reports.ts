import { useQuery, useQueryClient } from '@tanstack/react-query'
import type { ExamListItem } from '@/api/exams'

// ---- Types ------------------------------------------------------------------

export interface ExamListResponse {
  items: ExamListItem[]
  meta: { page: number; per_page: number; total: number }
}

// ---- Hooks ------------------------------------------------------------------

/** Fetch all exams for the Reports Hub table. */
export function useExamsListForReports() {
  const qc = useQueryClient()
  return useQuery({
    queryKey: ['exams-list-reports'],
    queryFn: async (): Promise<ExamListItem[]> => {
      const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
      const res = await fetch('/api/v1/exams?per_page=200', {
        credentials: 'include',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      if (!res.ok) throw new Error('ERR_FETCH_EXAMS')
      const body = await res.json() as { data: ExamListResponse; error: null | { code: string; message: string } }
      if (body.error) throw new Error(body.error.message)
      return body.data.items
    },
    staleTime: 60 * 1000,
  })
}

// ---- Download helpers -------------------------------------------------------

/** Download the dashboard PDF report for the given date range. */
export async function downloadDashboardPdf(from: string, to: string): Promise<void> {
  const res = await fetch(
    `/api/v1/admin/dashboard/export?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
    { credentials: 'include' },
  )
  if (!res.ok) throw new Error('ERR_EXPORT_PDF')
  const blob = await res.blob()
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `dashboard-report-${from}-${to}.pdf`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  window.URL.revokeObjectURL(url)
}

/** Download the per-exam results CSV. */
export async function downloadExamCsv(examId: string, examTitle: string): Promise<void> {
  const res = await fetch(`/api/v1/admin/exams/${examId}/results/export`, {
    credentials: 'include',
  })
  if (!res.ok) throw new Error('ERR_EXPORT_CSV')
  const blob = await res.blob()
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `exam-results-${examTitle.replace(/\s+/g, '-').toLowerCase()}.csv`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  window.URL.revokeObjectURL(url)
}
