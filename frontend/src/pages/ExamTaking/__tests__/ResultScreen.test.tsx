import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi } from 'vitest'
import { ResultScreen } from '../ResultScreen'
import type { SubmitResult } from '@/api/sessions'

const navigate = vi.fn()
vi.mock('react-router-dom', async (orig) => ({
  ...(await orig<typeof import('react-router-dom')>()),
  useNavigate: () => navigate,
}))
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) => (vars ? `${key}:${JSON.stringify(vars)}` : key),
  }),
}))

function renderResult(partial: Partial<SubmitResult>) {
  const result: SubmitResult = {
    session_id: 's',
    status: 'submitted',
    submitted_at: '2026-01-01T00:00:00Z',
    score_pct: null,
    passed: null,
    ...partial,
  }
  return render(
    <MemoryRouter>
      <ResultScreen result={result} examTitle="Safety 101" certificateEnabled={false} />
    </MemoryRouter>,
  )
}

describe('ResultScreen', () => {
  it('shows the pending state when grading has not finished', () => {
    renderResult({ passed: null })
    expect(screen.getByText('Safety 101')).toBeInTheDocument()
    expect(screen.getByText('exam.taking.result.pending')).toBeInTheDocument()
    expect(screen.queryByText(/result\.score/)).not.toBeInTheDocument()
  })

  it('shows passed with a rounded score', () => {
    renderResult({ passed: true, score_pct: 84.6 })
    expect(screen.getByText('exam.taking.result.passed')).toBeInTheDocument()
    expect(screen.getByText('exam.taking.result.score:{"score":85}')).toBeInTheDocument()
  })

  it('shows failed with a rounded score', () => {
    renderResult({ passed: false, score_pct: 41.2 })
    expect(screen.getByText('exam.taking.result.failed')).toBeInTheDocument()
    expect(screen.getByText('exam.taking.result.score:{"score":41}')).toBeInTheDocument()
  })

  it('omits the score line when no score is available', () => {
    renderResult({ passed: true, score_pct: null })
    expect(screen.queryByText(/result\.score/)).not.toBeInTheDocument()
  })

  it('returns to the portal on button click', async () => {
    renderResult({ passed: true, score_pct: 90 })
    await userEvent.click(screen.getByRole('button', { name: 'exam.taking.result.backToPortal' }))
    expect(navigate).toHaveBeenCalledWith('/portal')
  })
})
