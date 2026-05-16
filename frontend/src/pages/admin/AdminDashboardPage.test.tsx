import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import '@/i18n'
import { AdminDashboardPage } from './AdminDashboardPage'
import type { DashboardMetrics } from '@/api/dashboard'

const metrics: DashboardMetrics = {
  completion_rate_by_exam: [
    {
      exam_id: 'e-1',
      title: 'Safety Exam',
      assigned_count: 10,
      completed_count: 7,
      passed_count: 5,
    },
  ],
  overdue_employees: [
    {
      user_id: 'u-1',
      name: 'Alice Smith',
      exam_title: 'Safety Exam',
      exam_id: 'e-1',
      deadline: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString(),
    },
  ],
  recent_activity: [
    {
      session_id: 's-1',
      employee_name: 'Bob Jones',
      exam_title: 'Safety Exam',
      score_pct: 80,
      passed: true,
      submitted_at: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
    },
  ],
  avg_score_by_track: { security: null, safety: 78.5, loyalty: null },
}

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['auth', 'accessToken'], 'fake-token')
  qc.setQueryData(['tenant', 'config'], {
    app_name: 'BilimBaga',
    primary_color: '#6366f1',
    accent_color: '#a78bfa',
    default_locale: 'en',
    available_locales: ['en'],
  })
  return { qc, ...render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <AdminDashboardPage />
      </MemoryRouter>
    </QueryClientProvider>,
  ) }
}

describe('AdminDashboardPage', () => {
  beforeEach(() => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: metrics, error: null }),
    })
  })

  it('renders the page heading (AC-2)', async () => {
    setup()
    await waitFor(() => {
      expect(screen.getByText('Dashboard')).toBeInTheDocument()
    })
  })

  it('shows four KPI card labels after data loads (AC-2)', async () => {
    setup()
    await waitFor(() => {
      expect(screen.getByText('Total Employees')).toBeInTheDocument()
      expect(screen.getByText('Active Exams')).toBeInTheDocument()
      expect(screen.getByText('Completion Rate')).toBeInTheDocument()
      expect(screen.getByText('Pass Rate')).toBeInTheDocument()
    })
  })

  it('shows computed completion rate (AC-3)', async () => {
    setup()
    // 7/10 = 70.0%
    await waitFor(() => {
      expect(screen.getByText('70.0%')).toBeInTheDocument()
    })
  })

  it('shows computed pass rate (AC-4)', async () => {
    setup()
    // 5/10 = 50.0%
    await waitFor(() => {
      expect(screen.getByText('50.0%')).toBeInTheDocument()
    })
  })

  it('renders overdue employee name (AC-6)', async () => {
    setup()
    await waitFor(() => {
      expect(screen.getByText('Alice Smith')).toBeInTheDocument()
    })
  })

  it('renders recent activity employee name (AC-8)', async () => {
    setup()
    await waitFor(() => {
      expect(screen.getByText('Bob Jones')).toBeInTheDocument()
    })
  })

  it('renders a Refresh button that can be clicked (AC-9)', async () => {
    setup()
    await waitFor(() => screen.getByText('Dashboard'))
    const refreshBtn = screen.getByRole('button', { name: /refresh/i })
    expect(refreshBtn).toBeInTheDocument()
    // Clicking must not throw
    expect(() => fireEvent.click(refreshBtn)).not.toThrow()
  })

  it('shows loading text while fetching', () => {
    // Return a never-resolving promise to keep loading state
    global.fetch = vi.fn().mockReturnValue(new Promise(() => {}))
    setup()
    expect(screen.getByText('Loading…')).toBeInTheDocument()
  })

  it('shows error text when fetch fails', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ data: null, error: { code: 'ERR', message: 'fail' } }),
    })
    setup()
    await waitFor(() => {
      expect(screen.getByText('Failed to load. Please try again.')).toBeInTheDocument()
    })
  })
})
