import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import '@/i18n'
import { ResultsTable } from './ResultsTable'
import type { SessionHistoryItem } from '@/api/sessions'

const session: SessionHistoryItem = {
  session_id: 'sess-1',
  exam_id: 'exam-1',
  exam_title: 'Go Fundamentals',
  submitted_at: '2026-05-01T10:00:00Z',
  score_pct: 88.0,
  passed: true,
  time_taken_seconds: 600,
  certificate_available: true,
}

const failedSession: SessionHistoryItem = {
  ...session,
  session_id: 'sess-2',
  exam_title: 'Hard Exam',
  score_pct: 40.0,
  passed: false,
  time_taken_seconds: null,
  certificate_available: false,
}

function renderTable(
  sessions: SessionHistoryItem[],
  sortCol: 'date' | 'score' = 'date',
  sortDir: 'asc' | 'desc' = 'desc',
  onSort = vi.fn(),
) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <ResultsTable
          sessions={sessions}
          sortCol={sortCol}
          sortDir={sortDir}
          onSort={onSort}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('ResultsTable', () => {
  it('renders exam title', () => {
    renderTable([session])
    expect(screen.getByText('Go Fundamentals')).toBeInTheDocument()
  })

  it('renders score percentage', () => {
    renderTable([session])
    expect(screen.getByText('88.0%')).toBeInTheDocument()
  })

  it('renders Passed badge for passed session', () => {
    renderTable([session])
    expect(screen.getByText('Passed')).toBeInTheDocument()
  })

  it('renders Failed badge for failed session', () => {
    renderTable([failedSession])
    expect(screen.getByText('Failed')).toBeInTheDocument()
  })

  it('renders formatted duration for sessions with time', () => {
    renderTable([session])
    expect(screen.getByText('10m 0s')).toBeInTheDocument()
  })

  it('renders dash when no time taken', () => {
    renderTable([failedSession])
    const cells = screen.getAllByText('—')
    expect(cells.length).toBeGreaterThanOrEqual(1)
  })

  it('renders Download button when certificate available', () => {
    renderTable([session])
    expect(screen.getByRole('button', { name: /download/i })).toBeInTheDocument()
  })

  it('does not render Download button when certificate unavailable', () => {
    renderTable([failedSession])
    expect(screen.queryByRole('button', { name: /download/i })).toBeNull()
  })

  it('calls onSort with "date" when Date column header is clicked', () => {
    const onSort = vi.fn()
    renderTable([session], 'date', 'desc', onSort)
    fireEvent.click(screen.getByRole('button', { name: /date/i }))
    expect(onSort).toHaveBeenCalledWith('date')
  })

  it('calls onSort with "score" when Score column header is clicked', () => {
    const onSort = vi.fn()
    renderTable([session], 'date', 'desc', onSort)
    fireEvent.click(screen.getByRole('button', { name: /score/i }))
    expect(onSort).toHaveBeenCalledWith('score')
  })

  it('exam title links to session result page', () => {
    renderTable([session])
    const link = screen.getByRole('link', { name: 'Go Fundamentals' })
    expect(link).toHaveAttribute('href', '/portal/sessions/sess-1/result')
  })
})

describe('ResultsTable certificate download', () => {
  it('uses a Bearer-capable fetch to the portal endpoint and shows an error on failure', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('{}', { status: 500 }))
    vi.stubGlobal('fetch', fetchMock)
    renderTable([session])
    fireEvent.click(screen.getByRole('button', { name: /download/i }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Download failed')
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/portal/sessions/sess-1/certificate')
    vi.unstubAllGlobals()
  })
})
