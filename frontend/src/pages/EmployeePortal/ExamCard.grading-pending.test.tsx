import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import i18n from '@/i18n'
import { ExamCard } from './ExamCard'
import type { PortalExam } from '@/api/portal'

// FR-BB322 AC-3: a waiting session shows its own status, offers no Start or retake, and links to its result.

function waitingExam(overrides: Partial<PortalExam> = {}): PortalExam {
  return {
    id: 'e-1',
    title: 'Fire safety',
    description: null,
    time_limit_minutes: 30,
    passing_score_pct: 60,
    max_attempts: 3,
    attempts_used: 1,
    deadline: null,
    user_status: 'grading_pending',
    open_session_id: null,
    pending_session_id: 'sess-9',
    pending_submitted_at: '2026-10-11T09:00:00Z',
    show_answers: 'after_completion',
    shuffle_questions: false,
    shuffle_options: false,
    ...overrides,
  }
}

function renderCard(exam: PortalExam) {
  render(
    <MemoryRouter initialEntries={['/portal']}>
      <Routes>
        <Route path="/portal" element={<ExamCard exam={exam} onStart={() => {}} />} />
        <Route path="/portal/sessions/:sessionId/result" element={<div>Result page for the waiting session</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
})

describe('ExamCard while a session awaits grading', () => {
  it('shows the awaiting-grading badge and when the answers were submitted', () => {
    renderCard(waitingExam())
    expect(screen.getByText('Awaiting grading')).toBeInTheDocument()
    expect(screen.getByText(/^Submitted /)).toBeInTheDocument()
  })

  it('offers no Start, even with attempts left', () => {
    renderCard(waitingExam({ max_attempts: 3, attempts_used: 1 }))
    expect(screen.queryByRole('button', { name: 'Start exam' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Continue' })).toBeNull()
  })

  it('links to the waiting session result', () => {
    renderCard(waitingExam())
    fireEvent.click(screen.getByRole('button', { name: 'View status' }))
    expect(screen.getByText('Result page for the waiting session')).toBeInTheDocument()
  })
})
