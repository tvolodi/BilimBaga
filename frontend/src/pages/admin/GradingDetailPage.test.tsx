import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import '@/i18n'
import { GradingDetailPage } from './GradingDetailPage'
import { GradingQueuePage } from './GradingQueuePage'
import type { GradingSessionDetail } from '@/api/grading'

const useGradingSession = vi.fn()
const mutateAsync = vi.fn()
vi.mock('@/api/exams', () => ({ useExams: () => ({ data: { items: [] } }) }))
vi.mock('@/api/grading', () => ({
  useGradingQueue: () => ({ data: { items: [], meta: { total: 0, per_page: 20 } }, isLoading: false, isError: false }),
  useGradingSession: (...a: unknown[]) => useGradingSession(...a),
  useSubmitGrade: () => ({ mutateAsync }),
}))

function session(): GradingSessionDetail {
  return {
    session_id: 's1',
    employee_name: 'Alice Employee',
    exam_title: 'Security Basics',
    submitted_at: '2026-01-01T00:00:00Z',
    session_status: 'grading_pending',
    questions: [
      {
        question_id: 'q1',
        stem: 'Explain phishing',
        text_answer: 'Fake emails',
        grading_status: 'pending_manual',
        current_score_pct: null,
        manual_feedback: null,
        ai_suggested_score_pct: null,
        ai_suggested_at: null,
        ai_reasoning: null,
        ai_flags: null,
      },
      {
        question_id: 'q2',
        stem: 'Explain MFA',
        text_answer: 'Two factors',
        grading_status: 'graded',
        current_score_pct: 75,
        manual_feedback: 'Good enough',
        ai_suggested_score_pct: null,
        ai_suggested_at: null,
        ai_reasoning: null,
        ai_flags: null,
      },
    ],
  }
}

/** A final session: q1 is graded by a person and unchanged, q2 was graded by AI and can still be overridden. */
function finalSession(): GradingSessionDetail {
  const s = session()
  return {
    ...s,
    session_status: 'submitted',
    questions: [
      { ...s.questions[1], question_id: 'q1' },
      {
        ...s.questions[0],
        question_id: 'q2',
        stem: 'Explain MFA again',
        grading_status: 'ai_graded',
        current_score_pct: 40,
        ai_reasoning: 'Partly right.',
      },
    ],
  }
}

function Probe() {
  return <div data-testid="loc">{useLocation().pathname}</div>
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/admin/grading/s1']}>
      <Routes>
        <Route path="/admin/grading/:sessionId" element={<GradingDetailPage />} />
        <Route path="/admin/grading" element={<><Probe /><GradingQueuePage /></>} />
      </Routes>
    </MemoryRouter>,
  )
}

/** Set the numeric score field of the currently displayed question. */
function setScore(value: string) {
  fireEvent.change(screen.getByRole('spinbutton'), { target: { value } })
}

describe('GradingDetailPage (FR-BB47)', () => {
  beforeEach(() => {
    useGradingSession.mockReset()
    mutateAsync.mockReset()
    useGradingSession.mockReturnValue({ data: session(), isLoading: false, isError: false })
  })

  it('shows loading and error states', () => {
    useGradingSession.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    const { unmount } = renderPage()
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
    unmount()

    useGradingSession.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderPage()
    expect(screen.getByText(/failed to load|error/i)).toBeInTheDocument()
  })

  it('shows one question at a time with a position indicator and navigation (AC-4)', () => {
    renderPage()
    expect(screen.getByText(/Alice Employee/)).toBeInTheDocument()
    expect(screen.getByText('Question 1 of 2')).toBeInTheDocument()
    expect(screen.getByText('Explain phishing')).toBeInTheDocument()
    expect(screen.queryByText('Explain MFA')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /next/i }))
    expect(screen.getByText('Question 2 of 2')).toBeInTheDocument()
    expect(screen.getByText('Explain MFA')).toBeInTheDocument()
  })

  it('pre-fills the score and feedback of already graded questions (AC-5)', () => {
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /next/i }))
    expect(screen.getByRole('spinbutton')).toHaveValue(75)
    expect(screen.getByDisplayValue('Good enough')).toBeInTheDocument()
  })

  it('enables Submit only when every question has a score (AC-7)', () => {
    renderPage()
    const submit = screen.getByRole('button', { name: 'Submit All Grades' })
    expect(submit).toBeDisabled() // q1 has no score yet

    setScore('90')
    expect(submit).toBeEnabled() // q2 was pre-filled with 75
  })

  it('submits sequentially, then navigates to the queue when all graded (AC-8, AC-9)', async () => {
    mutateAsync
      .mockResolvedValueOnce({ all_graded: false })
      .mockResolvedValueOnce({ all_graded: true })
    renderPage()
    setScore('90')
    fireEvent.click(screen.getByRole('button', { name: /next/i }))
    setScore('80') // q2 is changed, so it is posted too (FR-BB73 AC-19)
    fireEvent.click(screen.getByRole('button', { name: 'Submit All Grades' }))

    await waitFor(() => expect(screen.getByTestId('loc')).toHaveTextContent('/admin/grading'))
    // AC-9: the success toast is visible on the queue page, and dismissible.
    const toast = await screen.findByRole('status')
    expect(toast).toHaveTextContent('All grades submitted. Session has been finalized.')
    fireEvent.click(screen.getByRole('button', { name: 'Dismiss' }))
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    expect(mutateAsync).toHaveBeenCalledTimes(2)
    expect(mutateAsync).toHaveBeenNthCalledWith(1, { questionId: 'q1', scorePct: 90, feedback: '' })
    expect(mutateAsync).toHaveBeenNthCalledWith(2, {
      questionId: 'q2',
      scorePct: 80,
      feedback: 'Good enough',
    })
  })

  it('posts only the answers that changed or still wait, not an unchanged graded answer (FR-BB73 AC-19)', async () => {
    mutateAsync.mockResolvedValueOnce({ all_graded: true })
    renderPage()
    setScore('90') // q1 waits, so it is posted; q2 is pre-filled and unchanged, so it is not
    fireEvent.click(screen.getByRole('button', { name: 'Submit All Grades' }))

    await waitFor(() => expect(screen.getByTestId('loc')).toHaveTextContent('/admin/grading'))
    expect(mutateAsync).toHaveBeenCalledTimes(1)
    expect(mutateAsync).toHaveBeenCalledWith({ questionId: 'q1', scorePct: 90, feedback: '' })
  })

  it('posts the suggested value when the examiner accepts it (FR-BB73 AC-17)', async () => {
    useGradingSession.mockReturnValue({
      data: {
        ...session(),
        questions: [
          {
            ...session().questions[0],
            ai_suggested_score_pct: 60,
            ai_suggested_at: '2026-10-11T06:00:00Z',
            ai_reasoning: 'Covers the main point.',
            ai_flags: ['decisive_pass'],
          },
        ],
      },
      isLoading: false,
      isError: false,
    })
    mutateAsync.mockResolvedValueOnce({ all_graded: false })
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: 'Accept' }))

    await waitFor(() => expect(mutateAsync).toHaveBeenCalledTimes(1))
    expect(mutateAsync).toHaveBeenCalledWith({ questionId: 'q1', scorePct: 60, feedback: '', accepted: true })
  })

  it('shows a final session read-only and overrides only the AI-graded answer (FR-BB47 AC-5, FR-BB73 AC-19)', async () => {
    useGradingSession.mockReturnValue({ data: finalSession(), isLoading: false, isError: false })
    mutateAsync.mockResolvedValueOnce({ all_graded: true })
    renderPage()
    expect(screen.getByText(/session is final/i)).toBeInTheDocument()
    expect(screen.getByRole('slider')).toBeDisabled() // q1 is graded and final

    fireEvent.click(screen.getByRole('button', { name: /next/i }))
    setScore('90') // q2 was graded by AI and can still be overridden
    fireEvent.click(screen.getByRole('button', { name: 'Submit All Grades' }))

    await waitFor(() => expect(screen.getByTestId('loc')).toHaveTextContent('/admin/grading'))
    expect(mutateAsync).toHaveBeenCalledTimes(1)
    expect(mutateAsync).toHaveBeenCalledWith({ questionId: 'q2', scorePct: 90, feedback: '' })
  })

  it('hides Submit for a final session with no AI grade to override', () => {
    const s = finalSession()
    useGradingSession.mockReturnValue({
      data: { ...s, questions: [s.questions[0]] },
      isLoading: false,
      isError: false,
    })
    renderPage()
    expect(screen.queryByRole('button', { name: 'Submit All Grades' })).not.toBeInTheDocument()
  })

  it('stops after a failed submission, names the failed question, and re-enables Submit (AC-11)', async () => {
    mutateAsync.mockRejectedValueOnce(new Error('boom'))
    renderPage()
    setScore('90')
    fireEvent.click(screen.getByRole('button', { name: 'Submit All Grades' }))

    expect(
      await screen.findByText('Failed to submit grade for question 1. Please retry.'),
    ).toBeInTheDocument()
    expect(mutateAsync).toHaveBeenCalledTimes(1) // q2 never fired
    expect(screen.queryByTestId('loc')).not.toBeInTheDocument() // stayed on the detail page
    expect(screen.getByRole('button', { name: 'Submit All Grades' })).toBeEnabled()
  })
})
