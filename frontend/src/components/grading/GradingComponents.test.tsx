import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import '@/i18n'
import { GradingNavigation } from './GradingNavigation'
import { GradingQueueTable } from './GradingQueueTable'
import { SubmitAllGradesButton, type GradeMap } from './SubmitAllGradesButton'
import type { GradingQueueItem } from '@/api/grading'

describe('GradingNavigation (FR-BB47 AC-4)', () => {
  it('shows "Question X of Y" and disables Previous on the first question', () => {
    const onPrev = vi.fn()
    const onNext = vi.fn()
    render(<GradingNavigation current={1} total={3} onPrev={onPrev} onNext={onNext} />)

    expect(screen.getByText('Question 1 of 3')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /previous/i })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: /next/i }))
    expect(onNext).toHaveBeenCalledTimes(1)
    expect(onPrev).not.toHaveBeenCalled()
  })

  it('disables Next on the last question and allows going back', () => {
    const onPrev = vi.fn()
    render(<GradingNavigation current={3} total={3} onPrev={onPrev} onNext={vi.fn()} />)

    expect(screen.getByRole('button', { name: /next/i })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: /previous/i }))
    expect(onPrev).toHaveBeenCalledTimes(1)
  })
})

function LocationProbe() {
  return <div data-testid="loc">{useLocation().pathname}</div>
}

describe('GradingQueueTable (FR-BB47 AC-2, AC-3)', () => {
  const sessions: GradingQueueItem[] = [
    {
      session_id: 'abcdef12-0000-4000-8000-000000000001',
      employee_name: 'Alice Employee',
      exam_id: 'e1',
      exam_title: 'Security Basics',
      submitted_at: '2026-01-02T10:00:00Z',
      pending_question_count: 2,
    },
    {
      session_id: '99999999-0000-4000-8000-000000000002',
      employee_name: 'Bob Employee',
      exam_id: 'e2',
      exam_title: 'Safety 101',
      submitted_at: '2026-01-03T10:00:00Z',
      pending_question_count: 5,
    },
  ]

  function renderTable() {
    return render(
      <MemoryRouter initialEntries={['/admin/grading']}>
        <GradingQueueTable sessions={sessions} />
        <Routes>
          <Route path="*" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>,
    )
  }

  it('renders the first 8 chars of the session id, employee, exam and pending count', () => {
    renderTable()
    expect(screen.getByText('abcdef12')).toBeInTheDocument()
    expect(screen.queryByText(sessions[0].session_id)).not.toBeInTheDocument()
    expect(screen.getByText('Alice Employee')).toBeInTheDocument()
    expect(screen.getByText('Safety 101')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getByText('5')).toBeInTheDocument()
  })

  it('keeps the server-provided row order (oldest first, no client re-sort)', () => {
    renderTable()
    const rows = screen.getAllByRole('row').slice(1)
    expect(rows[0]).toHaveTextContent('Alice Employee')
    expect(rows[1]).toHaveTextContent('Bob Employee')
  })

  it('navigates to the grading detail page when a row is clicked', () => {
    renderTable()
    fireEvent.click(screen.getByText('Bob Employee'))
    expect(screen.getByTestId('loc')).toHaveTextContent(
      '/admin/grading/99999999-0000-4000-8000-000000000002',
    )
  })
})

describe('SubmitAllGradesButton (FR-BB47 AC-6, AC-7, AC-8)', () => {
  const full: GradeMap = {
    q1: { score_pct: 80, feedback: '' },
    q2: { score_pct: 0, feedback: 'x' },
  }

  it('is enabled when every question has a score in 0..100 (0 counts as assigned)', () => {
    const onSubmit = vi.fn()
    render(<SubmitAllGradesButton grades={full} questionCount={2} onSubmit={onSubmit} isLoading={false} />)
    const btn = screen.getByRole('button', { name: 'Submit All Grades' })
    expect(btn).toBeEnabled()
    fireEvent.click(btn)
    expect(onSubmit).toHaveBeenCalledTimes(1)
  })

  it('is disabled when any score is null', () => {
    render(
      <SubmitAllGradesButton
        grades={{ ...full, q2: { score_pct: null, feedback: '' } }}
        questionCount={2}
        onSubmit={vi.fn()}
        isLoading={false}
      />,
    )
    expect(screen.getByRole('button')).toBeDisabled()
  })

  it('is disabled when fewer grades than questions exist', () => {
    render(
      <SubmitAllGradesButton grades={{ q1: full.q1 }} questionCount={2} onSubmit={vi.fn()} isLoading={false} />,
    )
    expect(screen.getByRole('button')).toBeDisabled()
  })

  it.each([101, -1])('is disabled when a score is out of range (%i)', (bad) => {
    render(
      <SubmitAllGradesButton
        grades={{ q1: { score_pct: bad, feedback: '' } }}
        questionCount={1}
        onSubmit={vi.fn()}
        isLoading={false}
      />,
    )
    expect(screen.getByRole('button')).toBeDisabled()
  })

  it('shows a submitting label and is disabled while submitting', () => {
    render(<SubmitAllGradesButton grades={full} questionCount={2} onSubmit={vi.fn()} isLoading />)
    expect(screen.getByRole('button', { name: /submitting/i })).toBeDisabled()
  })
})
