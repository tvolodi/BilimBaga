import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import '@/i18n'
import { EmployeePortal } from './index'

const EMPLOYEE_TOKEN =
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9' +
  '.eyJzdWIiOiJ1aWQtMiIsInJvbGUiOiJlbXBsb3llZSIsImV4cCI6OTk5OTk5OTk5OX0' +
  '.fake-signature'

const makeExam = (overrides: object = {}) => ({
  id: 'exam-1',
  title: 'Go Fundamentals',
  description: 'Learn Go',
  time_limit_minutes: 30,
  passing_score_pct: 70,
  max_attempts: 3,
  attempts_used: 0,
  deadline: null,
  user_status: 'not_started',
  open_session_id: null,
  show_answers: 'after_completion',
  shuffle_questions: false,
  shuffle_options: false,
  certificate_enabled: false,
  ...overrides,
})

const server = setupServer()

beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function renderPortal() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['auth', 'accessToken'], EMPLOYEE_TOKEN)

  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/portal']}>
        <EmployeePortal />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('EmployeePortal', () => {
  it('shows loading skeletons while exams are fetching', () => {
    server.use(
      http.get('/api/v1/portal/exams', () => new Promise(() => {})),
    )
    renderPortal()
    // skeleton elements have animate-pulse class
    const skeletons = document.querySelectorAll('.animate-pulse')
    expect(skeletons.length).toBeGreaterThan(0)
  })

  it('renders exam cards when data loads', async () => {
    server.use(
      http.get('/api/v1/portal/exams', () =>
        HttpResponse.json({ data: [makeExam()], error: null }),
      ),
    )
    renderPortal()
    await waitFor(() =>
      expect(screen.getByText('Go Fundamentals')).toBeInTheDocument(),
    )
    expect(screen.getByText(/not started/i)).toBeInTheDocument()
  })

  it('renders EmptyPortal when no exams are assigned', async () => {
    // Pre-seed the cache with empty array so the component skips loading state immediately.
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    qc.setQueryData(['auth', 'accessToken'], EMPLOYEE_TOKEN)

    server.use(
      http.get('/api/v1/portal/exams', () =>
        HttpResponse.json({ data: [], error: null }),
      ),
    )

    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={['/portal']}>
          <EmployeePortal />
        </MemoryRouter>
      </QueryClientProvider>,
    )

    await waitFor(
      () => {
        // EmptyPortal renders an h2 heading and a description paragraph
        const heading = screen.queryByRole('heading', { level: 2 })
        expect(heading).not.toBeNull()
      },
      { timeout: 6000 },
    )
    // Also confirm no exam cards rendered
    expect(screen.queryByRole('button', { name: /start exam/i })).toBeNull()
  })

  it('shows error message when the API fails', async () => {
    server.use(
      http.get('/api/v1/portal/exams', () =>
        HttpResponse.json({ data: null, error: { code: 'ERR', message: 'fail' } }),
      ),
    )
    renderPortal()
    await waitFor(() =>
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument(),
    )
  })

  it('renders a failed exam with View result button', async () => {
    server.use(
      http.get('/api/v1/portal/exams', () =>
        HttpResponse.json({
          data: [
            makeExam({
              user_status: 'failed',
              attempts_used: 3,
              max_attempts: 3,
              show_answers: 'after_completion',
            }),
          ],
          error: null,
        }),
      ),
    )
    renderPortal()
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /view result/i })).toBeInTheDocument(),
    )
  })

  it('renders an expired exam without CTA button', async () => {
    server.use(
      http.get('/api/v1/portal/exams', () =>
        HttpResponse.json({
          data: [
            makeExam({
              user_status: 'expired',
              deadline: '2020-01-01T00:00:00Z',
            }),
          ],
          error: null,
        }),
      ),
    )
    renderPortal()
    // Wait for the exam title to confirm the card rendered
    await waitFor(
      () => expect(screen.getByText('Go Fundamentals')).toBeInTheDocument(),
      { timeout: 5000 },
    )
    // Expired exams have no start / continue / view-result CTA
    expect(screen.queryByRole('button', { name: /start exam|continue|view result/i })).toBeNull()
  })

  it('disables View result button for failed exam with show_answers=never', async () => {
    server.use(
      http.get('/api/v1/portal/exams', () =>
        HttpResponse.json({
          data: [
            makeExam({
              user_status: 'failed',
              show_answers: 'never',
            }),
          ],
          error: null,
        }),
      ),
    )
    renderPortal()
    await waitFor(() => {
      const btn = screen.getByRole('button', { name: /view result/i })
      expect(btn).toBeDisabled()
    })
  })
})
