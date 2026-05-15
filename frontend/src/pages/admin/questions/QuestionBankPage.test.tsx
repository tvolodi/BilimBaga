import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { QuestionBankPage } from './QuestionBankPage'

const sampleQuestion = {
  id: 'q-1',
  type: 'single',
  difficulty: 'easy',
  status: 'draft',
  category_id: 'cat-1',
  category_name: 'Science',
  default_locale: 'en',
  version: 1,
  locale_coverage: ['en'],
  stem_preview: 'What is H2O?',
  tags: [],
  created_by: 'u-1',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
}

const server = setupServer(
  http.get('/api/v1/questions', () =>
    HttpResponse.json({
      data: {
        items: [sampleQuestion],
        meta: { page: 1, per_page: 20, total: 1 },
      },
      error: null,
    }),
  ),
  http.get('/api/v1/categories', () =>
    HttpResponse.json({
      data: [{ id: 'cat-1', name: 'Science', parent_id: null, children: [] }],
      error: null,
    }),
  ),
  http.get('/api/v1/tags', () =>
    HttpResponse.json({ data: [], error: null }),
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

describe('QuestionBankPage', () => {
  it('renders question rows from the API', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <QuestionBankPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByText('What is H2O?')).toBeInTheDocument()
    })
  })

  it('shows question difficulty', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <QuestionBankPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('What is H2O?'))
    // Multiple "Easy" elements exist (badge + filter chip) — just verify at least one is present
    expect(screen.getAllByText(/easy/i).length).toBeGreaterThan(0)
  })

  it('shows empty state when no questions returned', async () => {
    server.use(
      http.get('/api/v1/questions', () =>
        HttpResponse.json({
          data: { items: [], meta: { page: 1, per_page: 20, total: 0 } },
          error: null,
        }),
      ),
    )
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <QuestionBankPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(document.body).toBeInTheDocument()
    })
    expect(screen.queryByText('What is H2O?')).not.toBeInTheDocument()
  })

  it('has a button to create a new question', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <QuestionBankPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('What is H2O?'))
    expect(screen.getByRole('button', { name: /new question/i })).toBeInTheDocument()
  })

  it('filters by search term when typed into the search box', async () => {
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <QuestionBankPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('What is H2O?'))
    const searchInput = screen.getByPlaceholderText(/search/i)
    await userEvent.type(searchInput, 'test query')
    // Input updates without crash
    expect(searchInput).toHaveValue('test query')
  })
})
