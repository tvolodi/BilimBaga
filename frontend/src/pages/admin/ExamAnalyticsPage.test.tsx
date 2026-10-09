import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import '@/i18n'
import { ExamAnalyticsPage } from './ExamAnalyticsPage'

// Data hooks are stubbed in loading state: these tests cover the inner role guard, not the charts.
vi.mock('@/api/analytics', () => ({
  useExamAnalytics: () => ({ data: undefined, isLoading: true, isError: false, refetch: vi.fn() }),
}))
vi.mock('@/api/ai', () => ({
  useAIInsights: () => ({ data: undefined, isLoading: true, isError: false, refetch: vi.fn() }),
}))
vi.mock('@/api/useTenantConfig', () => ({
  useTenantConfig: () => ({ data: undefined }),
}))

// ---- helpers ----------------------------------------------------------------

/** Build a minimal JWT with the given role claim for use with RequireRole. */
function makeToken(role: string): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const payload = btoa(JSON.stringify({ role, sub: 'user-1' }))
  return `${header}.${payload}.fake-sig`
}

function renderAsRole(role: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['auth', 'accessToken'], makeToken(role))
  global.fetch = vi.fn()

  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/admin/exams/exam-1/analytics']}>
        <Routes>
          <Route path="/admin/exams/:examId/analytics" element={<ExamAnalyticsPage />} />
          <Route path="/login" element={<div>Login Page</div>} />
          <Route path="/admin" element={<div>Admin Home</div>} />
          <Route path="/portal" element={<div>Portal Home</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

// ---- tests ------------------------------------------------------------------

describe('ExamAnalyticsPage inner role guard (ISS-313)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('admits department_admin instead of redirecting it to the dashboard', () => {
    renderAsRole('department_admin')

    expect(screen.getByRole('heading', { name: 'Exam Analytics' })).toBeInTheDocument()
    expect(screen.queryByText('Admin Home')).not.toBeInTheDocument()
    expect(screen.queryByText('Portal Home')).not.toBeInTheDocument()
  })

  it.each(['examiner', 'hr_admin', 'super_admin'])('still admits %s', (role) => {
    renderAsRole(role)

    expect(screen.getByRole('heading', { name: 'Exam Analytics' })).toBeInTheDocument()
    expect(screen.queryByText('Admin Home')).not.toBeInTheDocument()
  })

  it('still redirects an employee away from the page', () => {
    renderAsRole('employee')

    expect(screen.getByText('Portal Home')).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Exam Analytics' })).not.toBeInTheDocument()
  })
})
