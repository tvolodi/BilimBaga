import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import '@/i18n'
import { Step4Review } from './Step4Review'
import { ExamApiError, type ExamDetail } from '@/api/exams'

const useExam = vi.fn()
const publish = vi.fn()

vi.mock('@/api/exams', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/exams')>()),
  useExam: (...a: unknown[]) => useExam(...a),
  useEligibleCounts: () => ({ data: { counts: [] }, isLoading: false, isError: false }),
  usePublishExam: () => ({ mutateAsync: publish, isPending: false }),
  useUnpublishExam: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useArchiveExam: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

const exam = {
  id: 'exam-1', title: 'T', description: null, status: 'draft', time_limit_minutes: 30,
  passing_score_pct: 70, max_attempts: 1, available_from: null, available_until: null,
  shuffle_questions: false, shuffle_options: false, show_answers: 'never', on_tab_switch: 'log',
  certificate_enabled: false, created_by: 'u', created_at: '', updated_at: '', sections: [], rules: [],
} as unknown as ExamDetail

async function publishWith(err: ExamApiError) {
  publish.mockRejectedValue(err)
  render(
    <MemoryRouter>
      <Step4Review examId="exam-1" onBack={vi.fn()} onPublished={vi.fn()} />
    </MemoryRouter>,
  )
  fireEvent.click(screen.getByRole('button', { name: 'Publish' }))
  const buttons = await screen.findAllByRole('button', { name: 'Publish' })
  fireEvent.click(buttons[buttons.length - 1])
  return screen.findByRole('alert')
}

function adaptive(details: unknown) {
  return new ExamApiError('Adaptive exam requires at least 5 questions per difficulty level per rule.', 'INSUFFICIENT_ADAPTIVE_QUESTIONS', 422, undefined, undefined, details)
}

describe('Step4Review publish errors (ISS-170)', () => {
  beforeEach(() => {
    useExam.mockReturnValue({ data: exam, isLoading: false })
    publish.mockReset()
  })

  it('INSUFFICIENT_ADAPTIVE_QUESTIONS with object details shows counts in an alert', async () => {
    const alert = await publishWith(adaptive({ rule_id: 'abcdef123', difficulty: 'medium', required: 5, available: 3 }))
    expect(alert).toHaveTextContent(/only 3 Medium questions, 5 are required/)
    expect(alert).not.toHaveTextContent('[object Object]')
  })

  it('INSUFFICIENT_ADAPTIVE_QUESTIONS with string details shows the generic localized text', async () => {
    const alert = await publishWith(adaptive('not enough'))
    expect(alert).toHaveTextContent(/at least 5 questions per difficulty/)
    expect(alert).not.toHaveTextContent('[object Object]')
  })

  it('INSUFFICIENT_ADAPTIVE_QUESTIONS with missing details shows the generic localized text', async () => {
    const alert = await publishWith(adaptive(undefined))
    expect(alert).toHaveTextContent(/at least 5 questions per difficulty/)
  })

  it.each([
    ['object', { x: 1 }],
    ['string', 'nope'],
    ['missing', undefined],
  ])('INSUFFICIENT_QUESTIONS with %s details shows the no-questions message', async (_n, details) => {
    const alert = await publishWith(new ExamApiError('The exam has no questions.', 'INSUFFICIENT_QUESTIONS', 422, undefined, undefined, details))
    expect(alert).toHaveTextContent(/This exam has no questions/)
    expect(alert).not.toHaveTextContent('[object Object]')
  })

  it('still renders per-rule list for array details (EXAM_RULES_UNSATISFIED)', async () => {
    const rules = [{ rule_id: 'abcdef123', required: 5, available: 1, filter: {} }]
    const alert = await publishWith(new ExamApiError('x', 'EXAM_RULES_UNSATISFIED', 422, undefined, rules, rules))
    await waitFor(() => expect(alert).toHaveTextContent(/needs 5, only 1 available/))
  })
})
