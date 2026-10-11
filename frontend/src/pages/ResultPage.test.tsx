import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import i18n from '@/i18n'
import { ResultPage } from './ResultPage'
import { RESULT_POLL_INTERVAL_MS, RESULT_POLL_LIMIT_MS } from '@/api/sessions'

// FR-BB322 AC-5 to AC-8: the result page while a session waits, the transition to the final result, and the question
// marker. The API is stubbed by URL: the session result and the exam detail.

type Result = Record<string, unknown>

let result: Result
let fetchMock: ReturnType<typeof vi.fn>

function pendingResult(overrides: Result = {}): Result {
  return {
    session_id: 's-1',
    exam_id: 'e-1',
    exam_title: 'Fire safety',
    status: 'grading_pending',
    score_pct: null,
    passed: false,
    time_taken_seconds: 600,
    attempt_number: 1,
    submitted_at: '2026-10-11T09:00:00Z',
    show_answers_mode: 'after_completion',
    per_section_scores: [],
    per_question_breakdown: [],
    pending_question_count: 1,
    total_gradable_count: 2,
    grading_expected_by: 'ai',
    ...overrides,
  }
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
  result = pendingResult()
  fetchMock = vi.fn().mockImplementation(async (url: string) => {
    const body = url.includes('/portal/exams/')
      ? {
          id: 'e-1', title: 'Fire safety', description: null, time_limit_minutes: 30, passing_score_pct: 60,
          max_attempts: 2, attempts_used: 1, deadline: null, user_status: 'not_started', open_session_id: null,
          pending_session_id: 's-1', pending_submitted_at: null, show_answers: 'after_completion',
          shuffle_questions: false, shuffle_options: false, certificate_enabled: false,
        }
      : result
    return { ok: true, status: 200, json: async () => ({ data: body, error: null }) }
  })
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

function renderResultPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const invalidate = vi.spyOn(qc, 'invalidateQueries')
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/portal/sessions/s-1/result']}>
        <Routes>
          <Route path="/portal/sessions/:sessionId/result" element={<ResultPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
  return { qc, invalidate }
}

describe('ResultPage while the session waits for grading', () => {
  it('shows the AI notice and how many answers are still checked', async () => {
    renderResultPage()
    expect(await screen.findByText('Your answers are being checked automatically. This usually takes a few minutes.')).toBeInTheDocument()
    expect(screen.getByText('1 of 2 answers are still being checked')).toBeInTheDocument()
    expect(screen.queryByText('Your Score')).toBeNull()
  })

  it('shows the examiner notice when a person will grade', async () => {
    result = pendingResult({ grading_expected_by: 'examiner' })
    renderResultPage()
    expect(await screen.findByText('Your answers are waiting for an examiner. You will get an email when the result is ready.')).toBeInTheDocument()
  })

  it('marks a question awaiting grading and shows neither its answer nor its explanation', async () => {
    result = pendingResult({
      per_question_breakdown: [
        {
          question_id: 'q-1', stem: 'Describe the exit plan', employee_answer: ['Use the north stairs'],
          correct_answer: ['Model answer text'], points_earned: 0, max_points: 2,
          explanation: 'Explained', manual_feedback: null, grading_status: 'pending_manual',
        },
      ],
    })
    renderResultPage()
    expect(await screen.findByText('Awaiting grading')).toBeInTheDocument()
    expect(screen.queryByText('Model answer text')).toBeNull()
    expect(screen.queryByText('Explained')).toBeNull()
  })
})

describe('ResultPage transition and polling', () => {
  it('announces the result and refreshes the lists when a waiting result turns final', async () => {
    vi.useFakeTimers()
    const { invalidate } = renderResultPage()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
    })
    expect(screen.queryByText('Your result is ready.')).toBeNull()

    result = {
      ...pendingResult(), status: 'submitted', score_pct: 80, passed: true,
      pending_question_count: 0, grading_expected_by: null,
    }
    // Two intervals: the first poll may land after the first fetch has settled.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(RESULT_POLL_INTERVAL_MS * 2)
    })

    expect(screen.getByText('Your result is ready.')).toBeInTheDocument()
    expect(screen.getByText('Your result is ready.').closest('[aria-live="polite"]')).not.toBeNull()
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['portal', 'exams'] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['my-results'] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['exam-history', 'e-1'] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['session-result', 's-1'] })
  })

  it('after 15 minutes offers Refresh, which fetches once more', async () => {
    vi.useFakeTimers()
    renderResultPage()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(RESULT_POLL_LIMIT_MS + RESULT_POLL_INTERVAL_MS)
    })
    expect(screen.getByText('We stopped checking automatically. Refresh to see the latest state.')).toBeInTheDocument()

    const before = fetchMock.mock.calls.length
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Refresh' }))
      await vi.advanceTimersByTimeAsync(0)
    })
    expect(fetchMock.mock.calls.length).toBeGreaterThan(before)
  })
})
