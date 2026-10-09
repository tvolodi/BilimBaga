import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
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

  it('renders a failed exam with View result button when attempts remain', async () => {
    server.use(
      http.get('/api/v1/portal/exams', () =>
        HttpResponse.json({
          data: [
            makeExam({
              user_status: 'failed',
              attempts_used: 2,
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

  it('renders a failed exam with no CTA and no-attempts sub-label when all attempts exhausted', async () => {
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
      expect(screen.getByText(/no attempts remaining/i)).toBeInTheDocument(),
    )
    expect(screen.queryByRole('button', { name: /view result/i })).not.toBeInTheDocument()
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

  // ISS-038: open_session_id takes priority over user_status for card display
  it('shows "In progress" status and "Continue" button when open_session_id is set even if user_status is "passed"', async () => {
    server.use(
      http.get('/api/v1/portal/exams', () =>
        HttpResponse.json({
          data: [
            makeExam({
              user_status: 'passed',
              open_session_id: 'sess-abc-123',
              attempts_used: 1,
              max_attempts: 2,
            }),
          ],
          error: null,
        }),
      ),
    )
    renderPortal()
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /continue/i })).toBeInTheDocument()
    })
    expect(screen.getByText(/in progress/i)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /view result/i })).toBeNull()
  })

  // ISS-132: the Start flow must navigate on success and must never fail silently.
  describe('Start exam flow (ISS-132)', () => {
    function renderPortalWithRoutes() {
      const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
      qc.setQueryData(['auth', 'accessToken'], EMPLOYEE_TOKEN)
      return render(
        <QueryClientProvider client={qc}>
          <MemoryRouter initialEntries={['/portal']}>
            <Routes>
              <Route path="/portal" element={<EmployeePortal />} />
              <Route path="/portal/sessions/:sessionId" element={<div>SESSION PAGE</div>} />
            </Routes>
          </MemoryRouter>
        </QueryClientProvider>,
      )
    }

    async function openStartModal() {
      server.use(
        http.get('/api/v1/portal/exams', () =>
          HttpResponse.json({ data: [makeExam()], error: null }),
        ),
      )
      renderPortalWithRoutes()
      const user = userEvent.setup()
      await user.click(await screen.findByRole('button', { name: /start exam/i }))
      return user
    }

    it('navigates to the session page after a successful start', async () => {
      let called = false
      server.use(
        http.post('/api/v1/portal/exams/exam-1/sessions', () => {
          called = true
          return HttpResponse.json(
            { data: { session_id: 'sess-42', exam_id: 'exam-1', questions: [] }, error: null },
            { status: 201 },
          )
        }),
      )
      const user = await openStartModal()
      await user.click(screen.getByRole('button', { name: /begin exam/i }))
      expect(await screen.findByText('SESSION PAGE')).toBeInTheDocument()
      expect(called).toBe(true)
    })

    it('shows a translated error in the modal when the backend refuses the start', async () => {
      server.use(
        http.post('/api/v1/portal/exams/exam-1/sessions', () =>
          HttpResponse.json(
            { data: null, error: { code: 'EXAM_OUTSIDE_WINDOW', message: 'raw backend text' } },
            { status: 422 },
          ),
        ),
      )
      const user = await openStartModal()
      await user.click(screen.getByRole('button', { name: /begin exam/i }))
      expect(await screen.findByRole('alert')).toHaveTextContent(/not available at this time/i)
      // The modal stays open and the button is usable again.
      expect(screen.getByRole('button', { name: /begin exam/i })).toBeEnabled()
      expect(screen.queryByText('SESSION PAGE')).toBeNull()
    })

    it('shows the generic error for unknown codes and non-JSON gateway failures', async () => {
      server.use(
        http.post('/api/v1/portal/exams/exam-1/sessions', () =>
          new HttpResponse('<html>Bad Gateway</html>', { status: 502 }),
        ),
      )
      const user = await openStartModal()
      await user.click(screen.getByRole('button', { name: /begin exam/i }))
      expect(await screen.findByRole('alert')).toHaveTextContent(/could not start the exam/i)
    })

    it('clears the error when the modal is closed and reopened', async () => {
      server.use(
        http.post('/api/v1/portal/exams/exam-1/sessions', () =>
          HttpResponse.json(
            { data: null, error: { code: 'ATTEMPTS_EXHAUSTED', message: 'x' } },
            { status: 422 },
          ),
        ),
      )
      const user = await openStartModal()
      await user.click(screen.getByRole('button', { name: /begin exam/i }))
      await screen.findByRole('alert')
      await user.click(screen.getByRole('button', { name: /cancel/i }))
      await user.click(await screen.findByRole('button', { name: /start exam/i }))
      expect(screen.queryByRole('alert')).toBeNull()
    })
  })
})
