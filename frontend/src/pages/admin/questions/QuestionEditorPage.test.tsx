import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
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

// ISS-005 regression: create payload must send nested translations, not flat stem/body
describe('QuestionEditorPage — create payload shape (ISS-005)', () => {
  it('POSTs translations map with stem/text — not flat stem/body — when saving a new question', async () => {
    let capturedBody: unknown = null

    server.use(
      http.post('/api/v1/questions', async ({ request }) => {
        capturedBody = await request.json()
        return HttpResponse.json({
          data: { ...sampleQuestion, id: 'new-q' },
          error: null,
        })
      }),
    )

    const Wrapper = createWrapper('/admin/questions/new')
    render(
      <Wrapper>
        <QuestionEditorPage />
      </Wrapper>,
    )

    // Wait for Save Draft button to appear
    await waitFor(() => screen.getByRole('button', { name: /save draft/i }))

    // Click Save Draft immediately — handleSaveDraft calls the API without client-side validation
    fireEvent.click(screen.getByRole('button', { name: /save draft/i }))

    await waitFor(() => expect(capturedBody).not.toBeNull(), { timeout: 3000 })

    const body = capturedBody as Record<string, unknown>

    // Must NOT have flat stem field at root level
    expect(body).not.toHaveProperty('stem')
    // Must have nested translations map
    expect(body).toHaveProperty('translations')
    expect(typeof body.translations).toBe('object')

    // Answer options must NOT have flat body field; must have nested translations with text
    if (Array.isArray(body.answer_options) && body.answer_options.length > 0) {
      const opt = body.answer_options[0] as Record<string, unknown>
      expect(opt).not.toHaveProperty('body')
      expect(opt).toHaveProperty('translations')
      const optTr = opt.translations as Record<string, unknown>
      const firstLocale = Object.values(optTr)[0] as Record<string, unknown>
      expect(firstLocale).toHaveProperty('text')
      expect(firstLocale).not.toHaveProperty('body')
    }
  })
})

// ISS-006 regression: default_locale must match the locale where the user entered text
describe('QuestionEditorPage — default_locale follows active locale (ISS-006)', () => {
  it('POSTs default_locale matching the locale tab where the stem was typed', async () => {
    let capturedBody: unknown = null

    server.use(
      http.post('/api/v1/questions', async ({ request }) => {
        capturedBody = await request.json()
        return HttpResponse.json({
          data: { ...sampleQuestion, id: 'new-q-ru' },
          error: null,
        })
      }),
    )

    const Wrapper = createWrapper('/admin/questions/new')
    render(
      <Wrapper>
        <QuestionEditorPage />
      </Wrapper>,
    )

    await waitFor(() => screen.getByRole('button', { name: /save draft/i }))

    // Switch to the RU locale tab
    const ruTab = screen.getByRole('button', { name: /ru/i })
    fireEvent.click(ruTab)

    // Type a stem in the now-active RU locale textarea
    const stemTextarea = document.querySelector('textarea') as HTMLTextAreaElement
    fireEvent.change(stemTextarea, { target: { value: 'Первая планета от Солнца' } })

    // Save draft
    fireEvent.click(screen.getByRole('button', { name: /save draft/i }))

    await waitFor(() => expect(capturedBody).not.toBeNull(), { timeout: 3000 })

    const body = capturedBody as Record<string, unknown>

    // default_locale must be 'ru', not 'en', because the user typed in the RU tab
    expect(body.default_locale).toBe('ru')

    // translations.ru.stem must be non-empty
    const translations = body.translations as Record<string, { stem: string }>
    expect(translations['ru']?.stem).toBe('Первая планета от Солнца')
  })
})
