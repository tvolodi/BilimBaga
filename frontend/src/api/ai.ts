import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

// ---- Types ------------------------------------------------------------------

export interface GenerateQuestionsRequest {
  category_id: string
  difficulty: 'easy' | 'medium' | 'hard'
  count: number
  context_text?: string
}

export interface DraftOption {
  text: string
  is_correct: boolean
}

export interface DraftQuestion {
  type: 'single_choice' | 'multiple_choice' | 'true_false'
  difficulty: 'easy' | 'medium' | 'hard'
  stem: string
  explanation: string
  options: DraftOption[]
  tags: string[]
}

export interface GenerateQuestionsResponse {
  questions: DraftQuestion[]
}

// ── FR-BB74: Performance Insight Summaries ────────────────────────────────────

export interface AIInsightsResponse {
  insights: string[]
  generated_at: string
  cached: boolean
}

// ---- API helpers -------------------------------------------------------------

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
  if (body.error) {
    const err = new Error(body.error.message) as Error & { code: string }
    err.code = body.error.code
    throw err
  }
  if (!res.ok) throw new Error(`Request failed: ${res.status}`)
  return body.data
}

// ---- API client functions ----------------------------------------------------

async function generateQuestions(req: GenerateQuestionsRequest): Promise<GenerateQuestionsResponse> {
  const response = await fetch('/api/v1/admin/ai/generate-questions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify(req),
  })

  const json = await response.json()

  if (!response.ok) {
    const code: string = json?.error?.code ?? 'UNKNOWN_ERROR'
    const message: string = json?.error?.message ?? 'An unexpected error occurred'
    const err = new Error(message) as Error & { code: string }
    err.code = code
    throw err
  }

  return json.data as GenerateQuestionsResponse
}

export function fetchAIInsights(examId: string, refresh = false): Promise<AIInsightsResponse> {
  const url = `/api/v1/admin/ai/insights/${examId}${refresh ? '?refresh=true' : ''}`
  return apiGet<AIInsightsResponse>(url)
}

// ---- React Query hooks -------------------------------------------------------

export function useGenerateQuestions() {
  return useMutation<GenerateQuestionsResponse, Error & { code?: string }, GenerateQuestionsRequest>({
    mutationFn: generateQuestions,
  })
}

export function useAIInsights(examId: string, refresh = false) {
  const qc = useQueryClient()
  const token = qc.getQueryData<string | null>(['auth', 'accessToken'])
  return useQuery<AIInsightsResponse, Error>({
    queryKey: ['ai-insights', examId, refresh],
    queryFn: () => {
      const url = `/api/v1/admin/ai/insights/${examId}${refresh ? '?refresh=true' : ''}`
      return apiGet<AIInsightsResponse>(url, token)
    },
    staleTime: 0,
    enabled: !!examId,
  })
}
