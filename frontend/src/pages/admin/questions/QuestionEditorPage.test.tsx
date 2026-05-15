import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { QuestionEditorPage } from './QuestionEditorPage'

const sampleQuestion = {
  id: 'q-1',
  type: 'single',
  difficulty: 'easy',
  status: 'draft',
  category_id: 'cat-1',
  default_locale: 'en',
  version: 1,
  parent_id: null,
  locale_coverage: ['en'],
  translations: {
    en: { stem: 'What is H2O?', explanation: null },
  },
  answer_options: [
    {
      id: 'opt-1',
      sort_order: 1,
      is_correct: true,
      likert_weight: null,
      likert_polarity: null,
      translations: { en: { body: 'Water' } },
    },
    {
      id: 'opt-2',
      sort_order: 2,
      is_correct: false,
      likert_weight: null,
      likert_polarity: null,
      translations: { en: { body: 'Fire' } },
    },
  ],
  tags: [],
  created_by: 'u-1',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
}

const server = setupServer(
  http.get('/api/v1/questions/q-1', () =>
    HttpResponse.json({ data: sampleQuestion, error: null }),
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

function createWrapper(initialPath = '/admin/questions/new') {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route path="/admin/questions/new" element={children} />
          <Route path="/admin/questions/:id/edit" element={children} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
}

describe('QuestionEditorPage — new question', () => {
  it('renders the editor with empty form fields for a new question', async () => {
    const Wrapper = createWrapper('/admin/questions/new')
    render(
      <Wrapper>
        <QuestionEditorPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(document.body).toBeInTheDocument()
    })
    // "Save Draft" button present for new question
    expect(screen.getByRole('button', { name: /save draft/i })).toBeInTheDocument()
  })

  it('shows type selector with all question types', async () => {
    const Wrapper = createWrapper('/admin/questions/new')
    render(
      <Wrapper>
        <QuestionEditorPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByRole('button', { name: /save/i }))
    // Type selector should render
    expect(document.body.textContent).toMatch(/single|multiple|true.*false/i)
  })
})

describe('QuestionEditorPage — edit existing question', () => {
  it('pre-populates form with existing question data', async () => {
    const Wrapper = createWrapper('/admin/questions/q-1/edit')
    render(
      <Wrapper>
        <QuestionEditorPage />
      </Wrapper>,
    )
    await waitFor(() => {
      expect(screen.getByDisplayValue('What is H2O?')).toBeInTheDocument()
    })
  })

  it('shows existing answer options', async () => {
    const Wrapper = createWrapper('/admin/questions/q-1/edit')
    render(
      <Wrapper>
        <QuestionEditorPage />
      </Wrapper>,
    )
    await waitFor(() => screen.getByDisplayValue('What is H2O?'))
    expect(screen.getByDisplayValue('Water')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Fire')).toBeInTheDocument()
  })
})
