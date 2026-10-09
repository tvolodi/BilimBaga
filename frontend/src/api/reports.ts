import { useQuery, useQueryClient } from '@tanstack/react-query'
import type { QueryClient } from '@tanstack/react-query'
import { downloadFile } from '@/api/download'
import type { ExamListItem } from '@/api/exams'
import { errorWithCode } from '@/api/errors'

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
      const body = (await res.json().catch(() => null)) as { data: ExamListResponse; error: null | { code: string; message: string } } | null
      if (body?.error) throw errorWithCode(body.error)
      if (!res.ok || !body) throw new Error('ERR_FETCH_EXAMS')
      return body.data.items
    },
    staleTime: 60 * 1000,
  })
}

// ---- Download helpers -------------------------------------------------------

/** Download the dashboard PDF report for the given date range. */
export function downloadDashboardPdf(qc: QueryClient, from: string, to: string): Promise<void> {
  return downloadFile(
    qc,
    `/api/v1/admin/dashboard/export?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
    `dashboard-report-${from}-${to}.pdf`,
  )
}

/** Download the per-exam results CSV. */
export function downloadExamCsv(qc: QueryClient, examId: string, examTitle: string): Promise<void> {
  return downloadFile(
    qc,
    `/api/v1/admin/exams/${examId}/results/export`,
    `exam-results-${examTitle.replace(/\s+/g, '-').toLowerCase()}.csv`,
  )
}
