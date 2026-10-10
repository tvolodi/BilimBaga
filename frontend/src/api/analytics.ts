import { useQuery, useQueryClient } from '@tanstack/react-query'
import type { QueryClient } from '@tanstack/react-query'
import { apiFetch } from './apiFetch'
import { retryUnlessNotFound } from '@/lib/apiRetry'

function apiGet<T>(qc: QueryClient, url: string): Promise<T> {
  return apiFetch<T>(qc, url)
}

// ---- Types ------------------------------------------------------------------

export interface ExamAnalytics {
  exam_id: string
  exam_title: string
  score_distribution: BucketCount[]
  pass_rate: number
  avg_score: number | null
  median_score: number | null
  total_attempts: number
  unique_participants: number
  per_question_stats: QuestionStat[]
}

export interface BucketCount {
  bucket: string
  count: number
}

export interface QuestionStat {
  question_id: string
  stem_preview: string
  correct_rate: number | null
  avg_time_seconds: number | null
  answer_distribution: AnswerDistributionItem[]
}

export interface AnswerDistributionItem {
  option_id: string
  option_text: string
  select_count: number
}

// ---- API helpers ------------------------------------------------------------

// ---- Hooks ------------------------------------------------------------------

export function useExamAnalytics(examId: string) {
  const qc = useQueryClient()
  return useQuery<ExamAnalytics, Error>({
    queryKey: ['exam-analytics', examId],
    queryFn: () => apiGet<ExamAnalytics>(qc, `/api/v1/admin/exams/${examId}/analytics`),
    staleTime: 5 * 60 * 1000,
    enabled: !!examId,
    retry: retryUnlessNotFound,
  })
}
