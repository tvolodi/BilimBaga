import { useMutation } from '@tanstack/react-query'

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

// ---- API client -------------------------------------------------------------

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

// ---- React Query hooks ------------------------------------------------------

export function useGenerateQuestions() {
  return useMutation<GenerateQuestionsResponse, Error & { code?: string }, GenerateQuestionsRequest>({
    mutationFn: generateQuestions,
  })
}
