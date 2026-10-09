import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import '@/i18n'
import { GradingQueuePage } from './GradingQueuePage'

const useGradingQueue = vi.fn()
vi.mock('@/api/grading', () => ({
  useGradingQueue: (...args: unknown[]) => useGradingQueue(...args),
}))
vi.mock('@/api/exams', () => ({
  useExams: () => ({
    data: { items: [{ id: 'ex-1', title: 'Security Basics' }, { id: 'ex-2', title: 'Safety 101' }] },
  }),
}))

function item(n: number) {
  return {
    session_id: `0000000${n}-aaaa-4000-8000-000000000000`,
    employee_name: `Employee ${n}`,
    exam_id: 'ex-1',
    exam_title: 'Security Basics',
    submitted_at: '2026-01-01T00:00:00Z',
    pending_question_count: n,
  }
}

function renderPage() {
  return render(
    <MemoryRouter>
      <GradingQueuePage />
    </MemoryRouter>,
  )
}

describe('GradingQueuePage (FR-BB47)', () => {
  beforeEach(() => {
    useGradingQueue.mockReset()
  })

  it('shows a loading message while fetching', () => {
    useGradingQueue.mockReturnValue({ data: undefined, isLoading: true, isError: false })
    renderPage()
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('shows an error message when the queue fails to load', () => {
    useGradingQueue.mockReturnValue({ data: undefined, isLoading: false, isError: true })
    renderPage()
    expect(screen.getByText(/failed to load|error/i)).toBeInTheDocument()
  })

  it('shows the empty state when no sessions await grading', () => {
    useGradingQueue.mockReturnValue({
      data: { items: [], meta: { page: 1, per_page: 20, total: 0 } },
      isLoading: false,
      isError: false,
    })
    renderPage()
    expect(screen.getByText('No sessions are awaiting manual grading.')).toBeInTheDocument()
  })

  it('renders rows, range label and pagination that advances the requested page', () => {
    useGradingQueue.mockReturnValue({
      data: { items: [item(1), item(2)], meta: { page: 1, per_page: 20, total: 45 } },
      isLoading: false,
      isError: false,
    })
    renderPage()
    expect(screen.getByText('Employee 1')).toBeInTheDocument()
    expect(screen.getByText('Showing 1–20 of 45')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled()
    expect(useGradingQueue).toHaveBeenLastCalledWith(1, undefined)

    fireEvent.click(screen.getByRole('button', { name: 'Next' }))
    expect(useGradingQueue).toHaveBeenLastCalledWith(2, undefined)
    expect(screen.getByText('Showing 21–40 of 45')).toBeInTheDocument()
  })

  it('disables Next on the last page', () => {
    useGradingQueue.mockReturnValue({
      data: { items: [item(1)], meta: { page: 1, per_page: 20, total: 20 } },
      isLoading: false,
      isError: false,
    })
    renderPage()
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
  })

  it('filters by exam and resets to page 1', () => {
    useGradingQueue.mockReturnValue({
      data: { items: [item(1)], meta: { page: 1, per_page: 20, total: 100 } },
      isLoading: false,
      isError: false,
    })
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: 'Next' }))
    expect(useGradingQueue).toHaveBeenLastCalledWith(2, undefined)

    fireEvent.change(screen.getByLabelText('Filter by exam'), { target: { value: 'ex-2' } })
    expect(useGradingQueue).toHaveBeenLastCalledWith(1, 'ex-2')
  })
})
