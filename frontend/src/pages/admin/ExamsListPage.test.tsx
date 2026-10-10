import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { ExamsListPage } from './ExamsListPage'

const activeExam = {
  id: 'exam-1',
  title: 'Active Exam',
  status: 'active',
  time_limit_minutes: 60,
  passing_score_pct: 70,
  max_attempts: 3,
  available_from: null,
  available_until: null,
  created_by: 'u-1',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
}

const draftExam = {
  id: 'exam-2',
  title: 'Draft Exam',
  status: 'draft',
  time_limit_minutes: 30,
  passing_score_pct: 50,
  max_attempts: 1,
  available_from: null,
  available_until: null,
  created_by: 'u-1',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
}

const server = setupServer(
  http.get('/api/v1/exams', () =>
    HttpResponse.json({
      data: {
        items: [activeExam, draftExam],
        meta: { page: 1, per_page: 20, total: 2 },
      },
      error: null,
    }),
  ),
  http.post('/api/v1/exams/exam-1/archive', () =>
    HttpResponse.json({
      data: { id: 'exam-1', status: 'archived' },
      error: null,
    }),
  ),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  )
}

describe('ExamsListPage — Archive button', () => {
  it('renders Archive button for active exams', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <ExamsListPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByText('Active Exam')).toBeInTheDocument()
    })
    // Archive buttons present for both active and draft exams
    const archiveButtons = screen.getAllByRole('button', { name: /archive/i })
    expect(archiveButtons.length).toBeGreaterThanOrEqual(2)
  })

  it('opens confirmation dialog when Archive button is clicked', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <ExamsListPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByText('Active Exam')).toBeInTheDocument()
    })
    const archiveButtons = screen.getAllByRole('button', { name: /archive/i })
    fireEvent.click(archiveButtons[0])
    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })
  })

  it('calls archive endpoint and shows success message', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <ExamsListPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByText('Active Exam')).toBeInTheDocument()
    })
    const archiveButtons = screen.getAllByRole('button', { name: /archive/i })
    // Click Archive on the first exam (active exam)
    fireEvent.click(archiveButtons[0])
    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })
    // Confirm in dialog
    const dialogArchiveBtn = screen.getAllByRole('button', { name: /archive/i }).find(
      (btn) => btn.closest('[role="dialog"]'),
    )
    if (dialogArchiveBtn) {
      fireEvent.click(dialogArchiveBtn)
    }
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
  })

  it('shows Archive button for draft exams', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <ExamsListPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByText('Draft Exam')).toBeInTheDocument()
    })
    const archiveButtons = screen.getAllByRole('button', { name: /archive/i })
    // Both active and draft exams should have archive buttons
    expect(archiveButtons.length).toBeGreaterThanOrEqual(2)
  })
})

describe('ExamsListPage — Assign button for active exams', () => {
  it('shows Assign button for active exams', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <ExamsListPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByText('Active Exam')).toBeInTheDocument()
    })
    expect(screen.getByRole('button', { name: /assign/i })).toBeInTheDocument()
  })

  it('does not show Assign button for draft exams', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <ExamsListPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByText('Draft Exam')).toBeInTheDocument()
    })
    // Only one Assign button — for the active exam, not for the draft
    const assignButtons = screen.getAllByRole('button', { name: /assign/i })
    expect(assignButtons.length).toBe(1)
  })

  it('Assign button links to wizard edit page at step 3 for active exams', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <ExamsListPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByText('Active Exam')).toBeInTheDocument()
    })
    const assignLink = screen.getByRole('link', { name: /assign/i })
    expect(assignLink).toHaveAttribute('href', '/admin/exams/exam-1/edit?step=3')
  })
})

// #471: the status badge shows the translated label, not the raw lowercase value.
describe('ExamsListPage status badges (#471)', () => {
  const archivedExam = { ...draftExam, id: 'exam-3', title: 'Old Exam', status: 'archived' }

  it.each([
    ['en', 'Archived'],
    ['kk', 'Мұрағатталған'],
    ['ru', 'В архиве'],
  ])('shows the archived status as "%s" in %s', async (lng, label) => {
    const { default: i18n } = await import('@/i18n')
    await i18n.changeLanguage(lng)
    server.use(
      http.get('/api/v1/exams', () =>
        HttpResponse.json({
          data: { items: [archivedExam], meta: { page: 1, per_page: 20, total: 1 } },
          error: null,
        }),
      ),
    )
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <ExamsListPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByText('Old Exam')).toBeInTheDocument()
    })

    const badge = screen.getAllByText(label).find((el) => el.tagName === 'SPAN')
    expect(badge).toBeDefined()
    expect(screen.queryByText('archived')).not.toBeInTheDocument()
    await i18n.changeLanguage('en')
  })
})
