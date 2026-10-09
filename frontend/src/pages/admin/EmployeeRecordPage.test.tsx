import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import '@/i18n'
import { EmployeeRecordPage } from './EmployeeRecordPage'

const useUser = vi.fn()
const useMe = vi.fn()
const useEmployeeRecord = vi.fn()
const useEmployeeProgress = vi.fn()
const exportEmployeeRecord = vi.fn()
const fetchLoyaltyNarrative = vi.fn()

vi.mock('@/api/users', () => ({
  useUser: (...a: unknown[]) => useUser(...a),
  useMe: () => useMe(),
}))
vi.mock('@/api/employees', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/employees')>()),
  useEmployeeRecord: (...a: unknown[]) => useEmployeeRecord(...a),
  useEmployeeProgress: (...a: unknown[]) => useEmployeeProgress(...a),
  exportEmployeeRecord: (...a: unknown[]) => exportEmployeeRecord(...a),
}))
vi.mock('@/api/ai', () => ({
  fetchLoyaltyNarrative: (...a: unknown[]) => fetchLoyaltyNarrative(...a),
}))

const user = {
  id: 'u1',
  email: 'alice@example.com',
  full_name: 'Alice Employee',
  department_id: 'd1',
  department_name: 'Finance',
  role_id: 'r1',
  role_name: 'employee',
  status: 'active',
  force_password_change: false,
  created_at: '2026-01-01T00:00:00Z',
}

function session(n: number, track: string | null = 'security') {
  return {
    session_id: `sess-${n}`,
    exam_id: `e${n}`,
    exam_title: `Exam ${n}`,
    started_at: '2026-02-01T09:00:00Z',
    submitted_at: '2026-02-01T09:30:00Z',
    score_pct: 80,
    passed: true,
    time_taken_seconds: 600,
    status: 'graded',
    certificate_id: null,
    exam_category_track: track,
  }
}

function ok<T>(data: T) {
  return { data, isLoading: false, isError: false, refetch: vi.fn() }
}

function record(sessions: unknown[], total: number, perPage = 20) {
  return ok({ data: { sessions }, meta: { page: 1, per_page: perPage, total } })
}

const progress = ok({
  user_id: 'u1',
  full_name: 'Alice Employee',
  tracks: [
    { track: 'security', questions_answered: 11, last_activity: null, required_exams: [] },
    { track: 'safety', questions_answered: 0, last_activity: null, required_exams: [] },
    { track: 'loyalty', questions_answered: 0, last_activity: null, required_exams: [] },
  ],
})

function renderPage(url = '/admin/users/u1/record') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[url]}>
        <Routes>
          <Route path="/admin/users/:userId/record" element={<EmployeeRecordPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('EmployeeRecordPage (FR-BB58)', () => {
  beforeEach(() => {
    for (const m of [useUser, useMe, useEmployeeRecord, useEmployeeProgress, exportEmployeeRecord, fetchLoyaltyNarrative]) {
      m.mockReset()
    }
    useUser.mockReturnValue(ok(user))
    useMe.mockReturnValue(ok({ ...user, id: 'admin', role_name: 'examiner' }))
    useEmployeeRecord.mockReturnValue(record([session(1), session(2)], 2))
    useEmployeeProgress.mockReturnValue(progress)
  })

  it('renders the skeleton while any of the three queries is loading (AC-12)', () => {
    useEmployeeProgress.mockReturnValue({ data: undefined, isLoading: true, isError: false, refetch: vi.fn() })
    const { container } = renderPage()
    expect(container.querySelector('.animate-pulse')).not.toBeNull()
    expect(screen.queryByText('Employee Record')).not.toBeInTheDocument()
  })

  it('renders an error with a retry that refetches only the failed queries (AC-11)', () => {
    const recordRefetch = vi.fn()
    const userRefetch = vi.fn()
    const progressRefetch = vi.fn()
    useUser.mockReturnValue({ data: undefined, isLoading: false, isError: false, refetch: userRefetch })
    useEmployeeRecord.mockReturnValue({ data: undefined, isLoading: false, isError: true, refetch: recordRefetch })
    useEmployeeProgress.mockReturnValue({ ...progress, refetch: progressRefetch })
    renderPage()

    expect(screen.getByText('Failed to load employee record.')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(recordRefetch).toHaveBeenCalledTimes(1)
    expect(userRefetch).not.toHaveBeenCalled()
    expect(progressRefetch).not.toHaveBeenCalled()
  })

  it('renders header, history, total label and track cards (AC-2, AC-3, AC-6, AC-7)', () => {
    renderPage()
    expect(screen.getByRole('heading', { name: 'Alice Employee' })).toBeInTheDocument()
    expect(screen.getByText('Exam 1')).toBeInTheDocument()
    expect(screen.getByText('Exam 2')).toBeInTheDocument()
    expect(screen.getByText('Showing 1–2 of 2')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Security' })).toBeInTheDocument()
    expect(screen.getByText('11')).toBeInTheDocument()
    expect(useEmployeeRecord).toHaveBeenCalledWith('u1', 1)
  })

  it('paginates 20 per page via the page query param (AC-6)', () => {
    useEmployeeRecord.mockReturnValue(record([session(1)], 45))
    renderPage('/admin/users/u1/record?page=2')
    expect(useEmployeeRecord).toHaveBeenCalledWith('u1', 2)
    expect(screen.getByText('Showing 21–40 of 45')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Previous' })).toBeEnabled()

    fireEvent.click(screen.getByRole('button', { name: 'Next' }))
    expect(useEmployeeRecord).toHaveBeenLastCalledWith('u1', 3)
  })

  it('disables Previous on page 1 and Next on the last page', () => {
    useEmployeeRecord.mockReturnValue(record([session(1)], 45))
    const { unmount } = renderPage()
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled()
    unmount()

    renderPage('/admin/users/u1/record?page=3')
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
  })

  it('hides the pagination row when the employee has no sessions', () => {
    useEmployeeRecord.mockReturnValue(record([], 0))
    renderPage()
    expect(screen.getByText('No exam sessions recorded for this employee.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Next' })).not.toBeInTheDocument()
  })

  it('treats an invalid page param as page 1', () => {
    renderPage('/admin/users/u1/record?page=-4')
    expect(useEmployeeRecord).toHaveBeenCalledWith('u1', 1)
  })

  it('exports the record and shows a translated error if it fails', async () => {
    exportEmployeeRecord.mockRejectedValueOnce(new Error('nope'))
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /export csv/i }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Download failed. Please try again.')
    expect(exportEmployeeRecord.mock.calls[0][1]).toBe('u1')
  })

  it('does not show a values profile section to examiners', () => {
    useEmployeeRecord.mockReturnValue(record([session(1, 'loyalty')], 1))
    renderPage()
    expect(screen.queryByText('Values Profile')).not.toBeInTheDocument()
  })

  it('lets admins generate a values profile narrative for loyalty sessions (FR-BB75)', async () => {
    useMe.mockReturnValue(ok({ ...user, id: 'admin', role_name: 'super_admin' }))
    useEmployeeRecord.mockReturnValue(record([session(1, 'loyalty'), session(2, 'security')], 2))
    fetchLoyaltyNarrative.mockResolvedValue({ narrative: 'Values team work.', generated_at: '2026-03-01T00:00:00Z' })
    renderPage()

    expect(screen.getAllByText('Values Profile')).toHaveLength(1) // loyalty session only
    fireEvent.click(screen.getByRole('button', { name: 'Generate' }))
    expect(await screen.findByText('Values team work.')).toBeInTheDocument()
    expect(fetchLoyaltyNarrative).toHaveBeenCalledWith('sess-1')
    expect(screen.getByRole('button', { name: 'Regenerate' })).toBeInTheDocument()
  })

  it('shows an AI-unavailable message when narrative generation fails', async () => {
    useMe.mockReturnValue(ok({ ...user, id: 'admin', role_name: 'super_admin' }))
    useEmployeeRecord.mockReturnValue(record([session(1, 'loyalty')], 1))
    fetchLoyaltyNarrative.mockRejectedValue(new Error('503'))
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: 'Generate' }))
    await waitFor(() => expect(screen.getByText(/unavailable/i)).toBeInTheDocument())
  })
})
