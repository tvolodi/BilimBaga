import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import i18n from '@/i18n'
import { ExamResultRedirectPage, pickResultSession } from './ExamResultRedirectPage'
import type { ExamHistorySession } from '@/api/sessions'

// FR-BB322 AC-14: the exam-level result opens the session waiting for grading, otherwise the newest attempt, and
// asks for every attempt so the choice is made from all of them.

function attempt(id: string, startedAt: string, status: string): ExamHistorySession {
  return {
    session_id: id,
    started_at: startedAt,
    submitted_at: status === 'in_progress' ? null : startedAt,
    score_pct: null,
    passed: null,
    status,
  }
}

describe('pickResultSession', () => {
  it('prefers the session waiting for grading, even when it is not the newest or the oldest', () => {
    const picked = pickResultSession([
      attempt('a', '2026-10-01T09:00:00Z', 'submitted'),
      attempt('pending', '2026-10-05T09:00:00Z', 'grading_pending'),
      attempt('c', '2026-10-09T09:00:00Z', 'submitted'),
    ])
    expect(picked?.session_id).toBe('pending')
  })

  it('otherwise opens the newest attempt by start time', () => {
    const picked = pickResultSession([
      attempt('old', '2026-10-01T09:00:00Z', 'submitted'),
      attempt('new', '2026-10-09T09:00:00Z', 'submitted'),
      attempt('mid', '2026-10-05T09:00:00Z', 'auto_submitted'),
    ])
    expect(picked?.session_id).toBe('new')
  })

  it('returns nothing without attempts', () => {
    expect(pickResultSession([])).toBeUndefined()
  })
})

describe('ExamResultRedirectPage', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('asks for every attempt and opens the waiting one, not the oldest', async () => {
    const sessions = [
      attempt('first', '2026-10-01T09:00:00Z', 'submitted'),
      attempt('second', '2026-10-03T09:00:00Z', 'submitted'),
      attempt('third', '2026-10-05T09:00:00Z', 'submitted'),
      attempt('waiting', '2026-10-07T09:00:00Z', 'grading_pending'),
    ]
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ data: { exam_id: 'e-1', exam_title: 'Fire', sessions, meta: { page: 1, per_page: 100, total: 4 } }, error: null }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={['/portal/exams/e-1/result']}>
          <Routes>
            <Route path="/portal/exams/:examId/result" element={<ExamResultRedirectPage />} />
            <Route path="/portal/sessions/:sessionId/result" element={<div>Session page for the waiting attempt</div>} />
            <Route path="/portal/results" element={<div>My results</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )

    expect(await screen.findByText('Session page for the waiting attempt')).toBeInTheDocument()
    expect(fetchMock.mock.calls[0][0]).toContain('/api/v1/portal/exams/e-1/history?page=1&per_page=100')
  })
})
