import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import i18n from '@/i18n'
import { ResultsTable } from './ResultsTable'
import type { SessionHistoryItem } from '@/api/sessions'

// FR-BB322 AC-13: a session waiting for grading shows the grading label where its score would be, and no pass or fail.

function row(overrides: Partial<SessionHistoryItem>): SessionHistoryItem {
  return {
    session_id: 's-1',
    exam_id: 'e-1',
    exam_title: 'Fire safety',
    submitted_at: '2026-10-11T09:00:00Z',
    score_pct: 80,
    passed: true,
    time_taken_seconds: 600,
    certificate_available: true,
    ...overrides,
  }
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
})

function renderTable(sessions: SessionHistoryItem[]) {
  const qc = new QueryClient()
  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <ResultsTable sessions={sessions} sortCol="date" sortDir="desc" onSort={() => {}} />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('ResultsTable row of a session awaiting grading', () => {
  it('shows the grading label instead of a score, and no pass or fail badge', () => {
    renderTable([row({ score_pct: null, passed: false, certificate_available: false })])
    expect(screen.getByText('Awaiting grading')).toBeInTheDocument()
    expect(screen.queryByText('Failed')).toBeNull()
    expect(screen.queryByText('Passed')).toBeNull()
  })

  it('keeps the score and the badge of a finished session', () => {
    renderTable([row({ score_pct: 80, passed: true })])
    expect(screen.getByText('80.0%')).toBeInTheDocument()
    expect(screen.getByText('Passed')).toBeInTheDocument()
  })
})
