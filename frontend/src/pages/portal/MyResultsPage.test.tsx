import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { MyResultsPage } from './MyResultsPage'

const singleSession = {
  session_id: 'sess-1',
  exam_id: 'exam-1',
  exam_title: 'Go Fundamentals',
  submitted_at: '2026-05-01T10:00:00Z',
  score_pct: 85.5,
  passed: true,
  time_taken_seconds: 1200,
  certificate_available: true,
}

const emptyResponse = {
  data: { sessions: [], meta: { page: 1, per_page: 20, total: 0 } },
  error: null,
}

const singleResponse = {
  data: { sessions: [singleSession], meta: { page: 1, per_page: 20, total: 1 } },
  error: null,
}

const server = setupServer(
  http.get('/api/v1/portal/results', () => HttpResponse.json(singleResponse)),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function createWrapper(initialPath = '/portal/results') {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialPath]}>{children}</MemoryRouter>
    </QueryClientProvider>
  )
}

describe('MyResultsPage', () => {
  it('renders the page title', async () => {
    const Wrapper = createWrapper()
    render(<Wrapper><MyResultsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 1 })).toBeInTheDocument()
    })
  })

  it('shows exam title after data loads', async () => {
    const Wrapper = createWrapper()
    render(<Wrapper><MyResultsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByText('Go Fundamentals')).toBeInTheDocument()
    })
  })

  it('shows score percentage', async () => {
    const Wrapper = createWrapper()
    render(<Wrapper><MyResultsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByText('85.5%')).toBeInTheDocument()
    })
  })

  it('shows empty state when no sessions', async () => {
    server.use(http.get('/api/v1/portal/results', () => HttpResponse.json(emptyResponse)))
    const Wrapper = createWrapper()
    render(<Wrapper><MyResultsPage /></Wrapper>)
    await waitFor(() => {
      expect(screen.getByText(/no.*exam/i)).toBeInTheDocument()
    })
  })

  it('shows error state on API failure', async () => {
    server.use(
      http.get('/api/v1/portal/results', () =>
        HttpResponse.json({ data: null, error: { code: 'ERR', message: 'fail' } }, { status: 500 }),
      ),
    )
    const Wrapper = createWrapper()
    render(<Wrapper><MyResultsPage /></Wrapper>)
    // loading indicator disappears and error appears
    await waitFor(() => {
      // skeleton rows or error text shown
      const errorEl = screen.queryByText(/fail|error|load/i)
      expect(errorEl).not.toBeNull()
    })
  })

  it('passes sort to API via URL params', async () => {
    let capturedUrl = ''
    server.use(
      http.get('/api/v1/portal/results', ({ request }) => {
        capturedUrl = request.url
        return HttpResponse.json(singleResponse)
      }),
    )
    const Wrapper = createWrapper('/portal/results?sort=score&dir=asc')
    render(<Wrapper><MyResultsPage /></Wrapper>)
    await waitFor(() => {
      expect(capturedUrl).toContain('sort=score')
      expect(capturedUrl).toContain('dir=asc')
    })
  })
})
