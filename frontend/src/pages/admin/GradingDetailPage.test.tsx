import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import '@/i18n'
import { GradingDetailPage } from './GradingDetailPage'
import type { GradingSessionDetail } from '@/api/grading'

const useGradingSession = vi.fn()
const mutateAsync = vi.fn()
vi.mock('@/api/grading', () => ({
  useGradingSession: (...a: unknown[]) => useGradingSession(...a),
  useSubmitGrade: () => ({ mutateAsync }),
}))

function session(): GradingSessionDetail {
  return {
    session_id: 's1',
    employee_name: 'Alice Employee',
    exam_title: 'Security Basics',
    submitted_at: '2026-01-01T00:00:00Z',
    questions: [
      {
        question_id: 'q1',
        stem: 'Explain phishing',
        text_answer: 'Fake emails',
        grading_status: 'pending_manual',
        current_score_pct: null,
        manual_feedback: null,
        ai_reasoning: null,
      },
      {
        question_id: 'q2',
        stem: 'Explain MFA',
        text_answer: 'Two factors',
        grading_status: 'graded',
        current_score_pct: 75,
        manual_feedback: 'Good enough',
        ai_reasoning: null,
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
        <Route path="/admin/grading" element={<Probe />} />
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
    fireEvent.click(screen.getByRole('button', { name: 'Submit All Grades' }))

    await waitFor(() => expect(screen.getByTestId('loc')).toHaveTextContent('/admin/grading'))
    expect(mutateAsync).toHaveBeenCalledTimes(2)
    expect(mutateAsync).toHaveBeenNthCalledWith(1, { questionId: 'q1', scorePct: 90, feedback: '' })
    expect(mutateAsync).toHaveBeenNthCalledWith(2, {
      questionId: 'q2',
      scorePct: 75,
      feedback: 'Good enough',
    })
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
