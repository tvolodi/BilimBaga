import { useQuery, useQueryClient } from '@tanstack/react-query'

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

interface ApiResponse<T> {
  data: T
  error: null | { code: string; message: string }
}

async function apiGet<T>(url: string, token?: string | null): Promise<T> {
  const res = await fetch(url, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    credentials: 'include',
  })
  const body: ApiResponse<T> = await res.json()
  if (body.error) throw new Error(body.error.message)
  if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  return body.data
}

// ---- Hooks ------------------------------------------------------------------

export function useExamAnalytics(examId: string) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<ExamAnalytics, Error>({
    queryKey: ['exam-analytics', examId],
    queryFn: () => apiGet<ExamAnalytics>(`/api/v1/admin/exams/${examId}/analytics`, token),
    staleTime: 5 * 60 * 1000,
    enabled: !!examId,
  })
}
