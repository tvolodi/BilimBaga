import { useQuery, useQueryClient } from '@tanstack/react-query'
import type { QueryClient } from '@tanstack/react-query'
import { downloadFile } from '@/api/download'
import type { ExamListItem } from '@/api/exams'
import { apiFetch } from './apiFetch'

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
      const body = await apiFetch<ExamListResponse>(qc, '/api/v1/exams?per_page=200')
      return body.items
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
