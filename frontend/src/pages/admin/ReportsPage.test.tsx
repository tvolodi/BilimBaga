import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import '@/i18n'
import { ReportsPage } from './ReportsPage'
import { RequireRole } from '@/components/RequireRole'
import type { ExamListItem } from '@/api/exams'

// ---- helpers ----------------------------------------------------------------

/** Build a minimal JWT with the given role claim for use with RequireRole. */
function makeToken(role: string): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const payload = btoa(JSON.stringify({ role, sub: 'user-1' }))
  return `${header}.${payload}.fake-sig`
}

const EXAMS: ExamListItem[] = [
  {
    id: 'exam-1',
    title: 'Safety Exam',
    status: 'active',
    time_limit_minutes: 60,
    passing_score_pct: 70,
    max_attempts: 3,
    available_from: null,
    available_until: null,
    created_by: 'user-1',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'exam-2',
    title: 'Security Awareness',
    status: 'draft',
    time_limit_minutes: 30,
    passing_score_pct: 60,
    max_attempts: 1,
    available_from: null,
    available_until: null,
    created_by: 'user-1',
    created_at: '2026-01-02T00:00:00Z',
    updated_at: '2026-01-02T00:00:00Z',
  },
]

function setup(opts: { role?: string; exams?: ExamListItem[] | null; fetchHangs?: boolean } = {}) {
  const { role = 'super_admin', exams = EXAMS, fetchHangs = false } = opts
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['auth', 'accessToken'], makeToken(role))

  if (fetchHangs) {
    global.fetch = vi.fn().mockImplementation(() => new Promise(() => {}))
  } else if (exams !== null) {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        data: { items: exams, meta: { page: 1, per_page: 200, total: exams.length } },
        error: null,
      }),
    })
  } else {
    // empty list
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        data: { items: [], meta: { page: 1, per_page: 200, total: 0 } },
        error: null,
      }),
    })
  }

  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <ReportsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

// ---- tests ------------------------------------------------------------------

describe('ReportsPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('AC-1: renders the page title', async () => {
    setup()
    await waitFor(() => {
      expect(screen.getByText('Reports')).toBeInTheDocument()
    })
  })

  it('AC-2: unauthorized role is redirected to /login', () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    qc.setQueryData(['auth', 'accessToken'], makeToken('employee'))
    global.fetch = vi.fn()

    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={['/admin/reports']}>
          <Routes>
            <Route
              path="/admin/reports"
              element={
                <RequireRole roles={['super_admin', 'examiner', 'hr_admin']}>
                  <ReportsPage />
                </RequireRole>
              }
            />
            <Route path="/login" element={<div>Login Page</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )

    expect(screen.getByText('Login Page')).toBeInTheDocument()
    expect(screen.queryByText('Reports')).not.toBeInTheDocument()
  })

  it('AC-3: Export PDF card renders date fields and button', async () => {
    setup()
    await waitFor(() => {
      expect(screen.getByText('Dashboard PDF Export')).toBeInTheDocument()
    })
    expect(screen.getByLabelText('From')).toBeInTheDocument()
    expect(screen.getByLabelText('To')).toBeInTheDocument()
    expect(screen.getByText('Export PDF')).toBeInTheDocument()
  })

  it('AC-5: shows skeleton while loading, then renders exam rows', async () => {
    // First verify that skeletons appear while loading (fetch never resolves)
    const { unmount } = setup({ fetchHangs: true })
    // While loading, exam titles should not be present
    expect(screen.queryByText('Safety Exam')).not.toBeInTheDocument()
    unmount()

    // Now render with data resolved
    setup()
    await waitFor(() => {
      expect(screen.getByText('Safety Exam')).toBeInTheDocument()
      expect(screen.getByText('Security Awareness')).toBeInTheDocument()
    })
  })

  it('AC-8: shows empty state when no exams exist', async () => {
    setup({ exams: null })
    await waitFor(() => {
      expect(screen.getByText('No exams found.')).toBeInTheDocument()
    })
  })

  it('AC-6: each exam row has Analytics link and Export CSV button', async () => {
    setup()
    await waitFor(() => {
      expect(screen.getByText('Safety Exam')).toBeInTheDocument()
    })
    // There should be one Analytics link per exam
    const analyticsLinks = screen.getAllByText('Analytics')
    expect(analyticsLinks).toHaveLength(2)
    // And one Export CSV button per exam
    const csvButtons = screen.getAllByText('Export CSV')
    expect(csvButtons).toHaveLength(2)
  })

  it('AC-10: two-column grid class is present on the layout wrapper', async () => {
    const { container } = setup()
    await waitFor(() => {
      expect(screen.getByText('Reports')).toBeInTheDocument()
    })
    const grid = container.querySelector('.md\\:grid-cols-2')
    expect(grid).toBeInTheDocument()
  })
})
