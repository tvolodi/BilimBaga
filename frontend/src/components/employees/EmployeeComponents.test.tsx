import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, within, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/i18n'
import { ExamStatusChip } from './ExamStatusChip'
import { EmployeeInfoHeader } from './EmployeeInfoHeader'
import { TrackProgressCards } from './TrackProgressCards'
import { SessionHistoryTable } from './SessionHistoryTable'
import { EmployeeRecordSkeleton } from './EmployeeRecordSkeleton'
import type { SessionRecord, TrackSummary } from '@/api/employees'
import type { User } from '@/api/users'

const downloadAdminCertificate = vi.fn()
vi.mock('@/api/employees', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/employees')>()),
  downloadAdminCertificate: (...a: unknown[]) => downloadAdminCertificate(...a),
}))

describe('ExamStatusChip (FR-BB58 AC-8)', () => {
  it('uses the success pair when passed', () => {
    render(<ExamStatusChip title="Security 1" passed attempts={2} />)
    const chip = screen.getByTitle('Security 1 — 2 attempt(s)')
    expect(chip.className).toMatch(/bg-bg-success/)
    expect(chip.className).toMatch(/text-success/)
  })

  it('uses the danger pair when failed', () => {
    render(<ExamStatusChip title="Security 1" passed={false} attempts={1} />)
    expect(screen.getByTitle(/Security 1/).className).toMatch(/bg-bg-danger/)
    expect(screen.getByTitle(/Security 1/).className).toMatch(/text-danger/)
  })

  it('uses the neutral token pair when never attempted (passed = null)', () => {
    render(<ExamStatusChip title="Security 1" passed={null} attempts={0} />)
    const chip = screen.getByTitle('Security 1 — 0 attempt(s)')
    expect(chip.className).toMatch(/bg-muted/)
    expect(chip.className).toMatch(/text-muted-foreground/)
    expect(chip.className).not.toMatch(/success|danger/)
  })
})

function makeUser(overrides: Partial<User> = {}): User {
  return {
    id: 'u1',
    email: 'alice@example.com',
    full_name: 'Alice Marie Employee',
    department_id: 'd1',
    department_name: 'Finance',
    role_id: 'r1',
    role_name: 'employee',
    status: 'active',
    force_password_change: false,
    is_locked: false,
    created_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('EmployeeInfoHeader (FR-BB58 AC-2)', () => {
  it('shows name, initials (max 2), department, email link, role and status', () => {
    render(<EmployeeInfoHeader user={makeUser()} />)
    expect(screen.getByRole('heading', { name: 'Alice Marie Employee' })).toBeInTheDocument()
    expect(screen.getByText('AM')).toBeInTheDocument()
    expect(screen.getByText('Finance')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'alice@example.com' })).toHaveAttribute(
      'href',
      'mailto:alice@example.com',
    )
  })

  it('shows a dash when the employee has no department', () => {
    render(<EmployeeInfoHeader user={makeUser({ department_name: null })} />)
    expect(screen.getByText('—')).toBeInTheDocument()
  })
})

describe('TrackProgressCards (FR-BB58 AC-7, AC-8)', () => {
  const tracks: TrackSummary[] = [
    {
      track: 'security',
      questions_answered: 42,
      last_activity: null,
      required_exams: [
        { exam_id: 'e1', title: 'Phishing 101', passed: true, attempts: 1 },
        { exam_id: 'e2', title: 'Passwords', passed: null, attempts: 0 },
      ],
    },
    { track: 'safety', questions_answered: 0, last_activity: null, required_exams: [] },
    {
      track: 'loyalty',
      questions_answered: 7,
      last_activity: new Date(Date.now() - 3 * 24 * 3600 * 1000).toISOString(),
      required_exams: [],
    },
  ]

  it('renders one card per track with counts, relative last activity and exam chips', () => {
    render(<TrackProgressCards tracks={tracks} />)
    expect(screen.getByRole('heading', { name: 'Security' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Safety' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Loyalty' })).toBeInTheDocument()
    expect(screen.getByText('42')).toBeInTheDocument()
    expect(screen.getByText('7')).toBeInTheDocument()
    expect(screen.getByText('3 days ago')).toBeInTheDocument()
    expect(screen.getAllByText('Never')).toHaveLength(2)
    expect(screen.getByText('Phishing 101')).toBeInTheDocument()
    expect(screen.getByText('Passwords')).toBeInTheDocument()
  })

  it('omits the required-exams block for a track without required exams', () => {
    render(<TrackProgressCards tracks={tracks} />)
    // Only the security track has required exams
    expect(screen.getAllByText('Required Exams')).toHaveLength(1)
  })
})

function makeSession(overrides: Partial<SessionRecord> = {}): SessionRecord {
  return {
    session_id: 'sess-1',
    exam_id: 'e1',
    exam_title: 'Phishing 101',
    started_at: '2026-02-01T09:00:00Z',
    submitted_at: '2026-02-01T09:30:00Z',
    score_pct: 87.456,
    passed: true,
    time_taken_seconds: 1800,
    status: 'graded',
    certificate_id: 'cert-1',
    exam_category_track: 'security',
    ...overrides,
  }
}

function renderTable(sessions: SessionRecord[]) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <SessionHistoryTable sessions={sessions} />
    </QueryClientProvider>,
  )
}

describe('SessionHistoryTable (FR-BB58 AC-3, AC-4)', () => {
  beforeEach(() => {
    downloadAdminCertificate.mockReset()
  })

  it('shows an empty message when there are no sessions', () => {
    renderTable([])
    expect(screen.getByText('No exam sessions recorded for this employee.')).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('shows score (1 decimal), pass/fail/grading-pending badge and time taken', () => {
    renderTable([
      makeSession(),
      makeSession({ session_id: 's2', passed: false, score_pct: 40, certificate_id: null, time_taken_seconds: 65 }),
      makeSession({
        session_id: 's3',
        passed: null,
        score_pct: null,
        time_taken_seconds: null,
        submitted_at: null,
        certificate_id: null,
      }),
    ])
    const rows = screen.getAllByRole('row').slice(1)
    expect(within(rows[0]).getByText('87.5%')).toBeInTheDocument()
    expect(within(rows[0]).getByText('Passed')).toBeInTheDocument()
    expect(within(rows[0]).getByText('30m 0s')).toBeInTheDocument()
    expect(within(rows[1]).getByText('40.0%')).toBeInTheDocument()
    expect(within(rows[1]).getByText('Failed')).toBeInTheDocument()
    expect(within(rows[1]).getByText('1m 5s')).toBeInTheDocument()
    expect(within(rows[2]).getByText('Grading Pending')).toBeInTheDocument()
  })

  it('shows the certificate button only for rows with a certificate', () => {
    renderTable([makeSession(), makeSession({ session_id: 's2', certificate_id: null })])
    expect(screen.getAllByRole('button', { name: 'Download' })).toHaveLength(1)
  })

  it('downloads the certificate for the clicked session', async () => {
    downloadAdminCertificate.mockResolvedValue(undefined)
    renderTable([makeSession()])
    fireEvent.click(screen.getByRole('button', { name: 'Download' }))
    await waitFor(() => expect(downloadAdminCertificate).toHaveBeenCalledTimes(1))
    expect(downloadAdminCertificate.mock.calls[0].slice(1)).toEqual(['sess-1', 'cert-1'])
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('shows a generic error alert when the download fails with an unknown code', async () => {
    downloadAdminCertificate.mockRejectedValue(Object.assign(new Error('x'), { code: 'ERR_DOWNLOAD' }))
    renderTable([makeSession()])
    fireEvent.click(screen.getByRole('button', { name: 'Download' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Download failed. Please try again.')
  })

  it('maps SESSION_NOT_PASSED to its own i18n message', async () => {
    downloadAdminCertificate.mockRejectedValue(Object.assign(new Error('x'), { code: 'SESSION_NOT_PASSED' }))
    renderTable([makeSession()])
    fireEvent.click(screen.getByRole('button', { name: 'Download' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('exam session was not passed')
  })

  it('maps EXAM_NOT_CERTIFIABLE to its own i18n message', async () => {
    downloadAdminCertificate.mockRejectedValue(Object.assign(new Error('x'), { code: 'EXAM_NOT_CERTIFIABLE' }))
    renderTable([makeSession()])
    fireEvent.click(screen.getByRole('button', { name: 'Download' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('does not issue certificates')
  })

  it('shows a session-expired message on an unauthorized download', async () => {
    downloadAdminCertificate.mockRejectedValue(Object.assign(new Error('x'), { code: 'ERR_UNAUTHORIZED' }))
    renderTable([makeSession()])
    fireEvent.click(screen.getByRole('button', { name: 'Download' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Your session has expired')
  })
})

describe('EmployeeRecordSkeleton (FR-BB58 AC-12)', () => {
  it('renders placeholder blocks without any real content', () => {
    const { container } = render(<EmployeeRecordSkeleton />)
    expect(container.querySelectorAll('.animate-pulse').length).toBeGreaterThan(0)
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })
})
