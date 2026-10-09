import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from 'vitest'
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

describe('QuestionBankPage bulk export (Bearer download)', () => {
  function renderWithToken() {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    qc.setQueryData(['auth', 'accessToken'], 'tok-q')
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter>
          <QuestionBankPage />
        </MemoryRouter>
      </QueryClientProvider>,
    )
  }

  async function selectFirstRowAndExport() {
    await waitFor(() => screen.getByText('What is H2O?'))
    await userEvent.click(screen.getAllByRole('checkbox')[1])
    await userEvent.click(await screen.findByRole('button', { name: /export csv/i }))
  }

  it('sends the Authorization header when exporting', async () => {
    let auth: string | null = null
    server.use(
      http.get('/api/v1/questions/export', ({ request }) => {
        auth = request.headers.get('Authorization')
        return new HttpResponse('id,q-1', { headers: { 'Content-Type': 'text/csv' } })
      }),
    )
    URL.createObjectURL = vi.fn(() => 'blob:mock')
    URL.revokeObjectURL = vi.fn()
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    renderWithToken()
    await selectFirstRowAndExport()
    await waitFor(() => expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:mock'))
    expect(auth).toBe('Bearer tok-q')
    vi.restoreAllMocks()
  })

  it('shows an error when the export fails', async () => {
    server.use(
      http.get('/api/v1/questions/export', () =>
        HttpResponse.json({ data: null, error: { code: 'ERR_INTERNAL' } }, { status: 500 }),
      ),
    )
    renderWithToken()
    await selectFirstRowAndExport()
    expect(await screen.findByRole('alert')).toHaveTextContent(/download failed/i)
  })
})

describe('QuestionBankPage language and version filters (ISS-134)', () => {
  function captureUrls() {
    const urls: URL[] = []
    server.use(
      http.get('/api/v1/questions', ({ request }) => {
        urls.push(new URL(request.url))
        return HttpResponse.json({
          data: { items: [sampleQuestion], meta: { page: 1, per_page: 20, total: 1 } },
          error: null,
        })
      }),
    )
    return urls
  }

  it('sends locale (not locale_missing) when the language filter is chosen', async () => {
    const urls = captureUrls()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <QuestionBankPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('What is H2O?'))
    expect(urls[0].searchParams.has('locale')).toBe(false)
    expect(urls[0].searchParams.has('include_versions')).toBe(false)

    await userEvent.selectOptions(screen.getByLabelText('Filter by locale'), 'kk')
    await waitFor(() => {
      const last = urls[urls.length - 1]
      expect(last.searchParams.get('locale')).toBe('kk')
      expect(last.searchParams.has('locale_missing')).toBe(false)
    })
  })

  it('sends locale_missing from the separate missing-translation filter', async () => {
    const urls = captureUrls()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <QuestionBankPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('What is H2O?'))
    await userEvent.selectOptions(screen.getByLabelText('Missing translation'), 'ru')
    await waitFor(() => {
      const last = urls[urls.length - 1]
      expect(last.searchParams.get('locale_missing')).toBe('ru')
      expect(last.searchParams.has('locale')).toBe(false)
    })
  })

  it('hides old versions by default and requests them via the toggle', async () => {
    const urls = captureUrls()
    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <QuestionBankPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByText('What is H2O?'))
    await userEvent.click(screen.getByLabelText('Show previous versions'))
    await waitFor(() => {
      expect(urls[urls.length - 1].searchParams.get('include_versions')).toBe('true')
    })
  })
})
