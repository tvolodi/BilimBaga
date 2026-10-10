import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { QueryClient } from '@tanstack/react-query'
import { apiFetch } from './apiFetch'

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

// ── FR-BB75: Loyalty Profile Narrative ───────────────────────────────────────

export interface LoyaltyNarrativeResponse {
  narrative: string
  generated_at: string
}

// ---- API helpers -------------------------------------------------------------

// ---- API client functions ----------------------------------------------------

function apiGet<T>(qc: QueryClient, url: string): Promise<T> {
  return apiFetch<T>(qc, url)
}

async function generateQuestions(qc: QueryClient, req: GenerateQuestionsRequest): Promise<GenerateQuestionsResponse> {
  return apiFetch<GenerateQuestionsResponse>(qc, '/api/v1/admin/ai/generate-questions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
}

export function fetchAIInsights(qc: QueryClient, examId: string, refresh = false): Promise<AIInsightsResponse> {
  const url = `/api/v1/admin/ai/insights/${examId}${refresh ? '?refresh=true' : ''}`
  return apiGet<AIInsightsResponse>(qc, url)
}

export function fetchLoyaltyNarrative(qc: QueryClient, sessionId: string): Promise<LoyaltyNarrativeResponse> {
  return apiGet<LoyaltyNarrativeResponse>(qc, `/api/v1/admin/ai/loyalty-summary/${sessionId}`)
}

// ---- React Query hooks -------------------------------------------------------

export function useGenerateQuestions() {
  const qc = useQueryClient()
  return useMutation<GenerateQuestionsResponse, Error & { code?: string }, GenerateQuestionsRequest>({
    mutationFn: (req) => generateQuestions(qc, req),
  })
}

export function useAIInsights(examId: string, refresh = false) {
  const qc = useQueryClient()
  return useQuery<AIInsightsResponse, Error>({
    queryKey: ['ai-insights', examId, refresh],
    queryFn: () => {
      const url = `/api/v1/admin/ai/insights/${examId}${refresh ? '?refresh=true' : ''}`
      return apiGet<AIInsightsResponse>(qc, url)
    },
    staleTime: 0,
    enabled: !!examId,
  })
}
