import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { type ReactNode } from 'react'
import '@/i18n'
import { AIGenerateDialog } from './AIGenerateDialog'

// ---- Fixtures ---------------------------------------------------------------

const categories = [
  { id: 'cat-1', name: 'Biology', parent_id: null, children: [] },
  {
    id: 'cat-2',
    name: 'Science',
    parent_id: null,
    children: [{ id: 'cat-3', name: 'Physics', parent_id: 'cat-2', children: [] }],
  },
]

const draftQuestions = [
  {
    type: 'single_choice',
    difficulty: 'easy',
    stem: 'What is 2 plus 2?',
    explanation: 'Basic addition',
    options: [
      { text: '3', is_correct: false },
      { text: '4', is_correct: true },
    ],
    tags: ['math', 'arithmetic'],
  },
  {
    type: 'multiple_choice',
    difficulty: 'medium',
    stem: 'Pick the prime numbers',
    explanation: 'Primes have exactly two divisors',
    options: [
      { text: '2', is_correct: true },
      { text: '4', is_correct: false },
      { text: '5', is_correct: true },
    ],
    tags: [],
  },
  {
    type: 'true_false',
    difficulty: 'hard',
    stem: 'The Earth is flat',
    explanation: 'The Earth is an oblate spheroid',
    options: [
      { text: 'True', is_correct: false },
      { text: 'False', is_correct: true },
    ],
    tags: ['geo'],
  },
]

// ---- MSW --------------------------------------------------------------------

const GENERATE_URL = '/api/v1/admin/ai/generate-questions'
const QUESTIONS_URL = '/api/v1/questions'

const server = setupServer(
  http.get('/api/v1/categories', () =>
    HttpResponse.json({ data: categories, error: null }),
  ),
  http.post(GENERATE_URL, () =>
    HttpResponse.json({ data: { questions: draftQuestions }, error: null }),
  ),
  http.post(QUESTIONS_URL, () =>
    HttpResponse.json({ data: { id: 'q-new' }, error: null }, { status: 201 }),
  ),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

// ---- Helpers ----------------------------------------------------------------

function renderDialog({
  role = 'examiner',
  open = true,
  onClose = vi.fn(),
  onSuccess = vi.fn(),
}: {
  role?: string | null
  open?: boolean
  onClose?: () => void
  onSuccess?: (count: number) => void
} = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  if (role !== null) {
    queryClient.setQueryData(['auth', 'currentUser'], { role })
  }
  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
  const utils = render(
    <Wrapper>
      <AIGenerateDialog open={open} onClose={onClose} onSuccess={onSuccess} />
    </Wrapper>,
  )
  return { ...utils, onClose, onSuccess }
}

function categorySelect() {
  return screen.getAllByRole('combobox')[0]
}

function difficultySelect() {
  return screen.getAllByRole('combobox')[1]
}

/** Opens the dialog, waits for categories, and picks the first category. */
async function fillFormWithCategory(categoryId = 'cat-1') {
  await screen.findByRole('option', { name: 'Biology' })
  fireEvent.change(categorySelect(), { target: { value: categoryId } })
}

async function generateDrafts() {
  await fillFormWithCategory()
  fireEvent.click(screen.getByRole('button', { name: 'AI Generate' }))
  await screen.findByText('Preview Generated Questions')
}

// ---- Tests ------------------------------------------------------------------

describe('AIGenerateDialog', () => {
  describe('visibility and role gating', () => {
    it.each(['examiner', 'department_admin', 'super_admin'])(
      'renders the dialog for the %s role',
      (role) => {
        renderDialog({ role })
        expect(screen.getByText('Generate Questions with AI')).toBeInTheDocument()
      },
    )

    it.each(['employee', 'auditor'])('renders nothing for the %s role', (role) => {
      renderDialog({ role })
      expect(screen.queryByText('Generate Questions with AI')).not.toBeInTheDocument()
    })

    it('renders nothing when no current user is cached', () => {
      renderDialog({ role: null })
      expect(screen.queryByText('Generate Questions with AI')).not.toBeInTheDocument()
    })

    it('renders no dialog content when open is false', () => {
      renderDialog({ open: false })
      expect(screen.queryByText('Generate Questions with AI')).not.toBeInTheDocument()
    })
  })

  describe('open and close', () => {
    it('calls onClose when Cancel is clicked', () => {
      const { onClose, onSuccess } = renderDialog()
      fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
      expect(onClose).toHaveBeenCalledTimes(1)
      expect(onSuccess).not.toHaveBeenCalled()
    })

    it('calls onClose when Escape closes the dialog', async () => {
      const user = userEvent.setup()
      const { onClose } = renderDialog()
      await user.keyboard('{Escape}')
      expect(onClose).toHaveBeenCalledTimes(1)
    })

    it('discards generated drafts when Cancel is clicked, before closing', async () => {
      const onClose = vi.fn()
      renderDialog({ onClose })
      await generateDrafts()

      fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))

      expect(onClose).toHaveBeenCalledTimes(1)
      expect(screen.queryByText('Preview Generated Questions')).not.toBeInTheDocument()
      expect(categorySelect()).toBeInTheDocument()
    })
  })

  describe('form', () => {
    it('lists the root and nested categories in the category select', async () => {
      renderDialog()
      expect(await screen.findByRole('option', { name: 'Biology' })).toBeInTheDocument()
      expect(screen.getByRole('option', { name: 'Science' })).toBeInTheDocument()
      expect(screen.getByRole('option', { name: 'Physics' })).toBeInTheDocument()
    })

    it('defaults difficulty to medium and count to 5', () => {
      renderDialog()
      expect(difficultySelect()).toHaveValue('medium')
      expect(screen.getByRole('spinbutton')).toHaveValue(5)
    })

    it('clamps the question count to the range 1 to 10', () => {
      renderDialog()
      const count = screen.getByRole('spinbutton')

      fireEvent.change(count, { target: { value: '15' } })
      expect(count).toHaveValue(10)

      fireEvent.change(count, { target: { value: '0' } })
      expect(count).toHaveValue(1)

      fireEvent.change(count, { target: { value: '7' } })
      expect(count).toHaveValue(7)
    })

    it('shows the context character counter as the user types', () => {
      renderDialog()
      expect(screen.getByText('0/2000')).toBeInTheDocument()
      fireEvent.change(screen.getByRole('textbox'), { target: { value: 'Photosynthesis' } })
      expect(screen.getByText('14/2000')).toBeInTheDocument()
    })

    it('shows a category error and makes no request when generating without a category', async () => {
      const generateRequests: string[] = []
      server.use(
        http.post(GENERATE_URL, () => {
          generateRequests.push(GENERATE_URL)
          return HttpResponse.json({ data: { questions: draftQuestions }, error: null })
        }),
      )
      renderDialog()
      await screen.findByRole('option', { name: 'Biology' })

      fireEvent.click(screen.getByRole('button', { name: 'AI Generate' }))

      expect(
        await screen.findByText('Please select a category before generating.'),
      ).toBeInTheDocument()
      expect(generateRequests).toHaveLength(0)
    })
  })

  describe('generate request', () => {
    it('POSTs the form values to the generate endpoint', async () => {
      const captured: { method: string; path: string; body: unknown }[] = []
      server.use(
        http.post(GENERATE_URL, async ({ request }) => {
          captured.push({
            method: request.method,
            path: new URL(request.url).pathname,
            body: await request.json(),
          })
          return HttpResponse.json({ data: { questions: draftQuestions }, error: null })
        }),
      )
      renderDialog()
      await screen.findByRole('option', { name: 'Biology' })

      fireEvent.change(categorySelect(), { target: { value: 'cat-2' } })
      fireEvent.change(difficultySelect(), { target: { value: 'hard' } })
      fireEvent.change(screen.getByRole('spinbutton'), { target: { value: '3' } })
      fireEvent.change(screen.getByRole('textbox'), {
        target: { value: 'Focus on the water cycle' },
      })
      fireEvent.click(screen.getByRole('button', { name: 'AI Generate' }))
      await screen.findByText('Preview Generated Questions')

      expect(captured).toEqual([
        {
          method: 'POST',
          path: GENERATE_URL,
          body: {
            category_id: 'cat-2',
            difficulty: 'hard',
            count: 3,
            context_text: 'Focus on the water cycle',
          },
        },
      ])
    })

    it('omits context_text from the body when the context is left empty', async () => {
      const bodies: Record<string, unknown>[] = []
      server.use(
        http.post(GENERATE_URL, async ({ request }) => {
          bodies.push((await request.json()) as Record<string, unknown>)
          return HttpResponse.json({ data: { questions: draftQuestions }, error: null })
        }),
      )
      renderDialog()
      await generateDrafts()

      expect(bodies).toHaveLength(1)
      expect(bodies[0]).toEqual({ category_id: 'cat-1', difficulty: 'medium', count: 5 })
      expect(bodies[0]).not.toHaveProperty('context_text')
    })

    it('shows the loading state while the request is in flight', async () => {
      let release!: () => void
      const gate = new Promise<void>((resolve) => {
        release = resolve
      })
      server.use(
        http.post(GENERATE_URL, async () => {
          await gate
          return HttpResponse.json({ data: { questions: draftQuestions }, error: null })
        }),
      )
      renderDialog()
      await fillFormWithCategory()
      fireEvent.click(screen.getByRole('button', { name: 'AI Generate' }))

      expect(await screen.findByRole('button', { name: 'Generating...' })).toBeDisabled()
      expect(screen.getByText('Generating...', { selector: 'p' })).toBeInTheDocument()

      release()
      expect(await screen.findByText('Preview Generated Questions')).toBeInTheDocument()
      expect(screen.queryByText('Generating...', { selector: 'p' })).not.toBeInTheDocument()
    })

    it('clears a previous error when a new generate attempt starts', async () => {
      server.use(
        http.post(GENERATE_URL, () =>
          HttpResponse.json(
            { data: null, error: { code: 'AI_RATE_LIMITED', message: 'limited' } },
            { status: 429 },
          ),
        ),
      )
      renderDialog()
      await fillFormWithCategory()
      fireEvent.click(screen.getByRole('button', { name: 'AI Generate' }))
      expect(
        await screen.findByText(
          'You have reached the AI generation limit. Please try again in an hour.',
        ),
      ).toBeInTheDocument()

      server.resetHandlers()
      fireEvent.click(screen.getByRole('button', { name: 'AI Generate' }))

      await screen.findByText('Preview Generated Questions')
      expect(screen.queryByText(/generation limit/)).not.toBeInTheDocument()
    })
  })

  describe('error branch', () => {
    it('shows the rate limit message for AI_RATE_LIMITED and keeps the form', async () => {
      server.use(
        http.post(GENERATE_URL, () =>
          HttpResponse.json(
            { data: null, error: { code: 'AI_RATE_LIMITED', message: 'limited' } },
            { status: 429 },
          ),
        ),
      )
      renderDialog()
      await fillFormWithCategory()
      fireEvent.click(screen.getByRole('button', { name: 'AI Generate' }))

      expect(
        await screen.findByText(
          'You have reached the AI generation limit. Please try again in an hour.',
        ),
      ).toBeInTheDocument()
      expect(categorySelect()).toBeInTheDocument()
      expect(screen.queryByText('Preview Generated Questions')).not.toBeInTheDocument()
    })

    it('shows the unavailable message for any other error', async () => {
      server.use(
        http.post(GENERATE_URL, () =>
          HttpResponse.json(
            { data: null, error: { code: 'INTERNAL', message: 'boom' } },
            { status: 500 },
          ),
        ),
      )
      renderDialog()
      await fillFormWithCategory()
      fireEvent.click(screen.getByRole('button', { name: 'AI Generate' }))

      expect(
        await screen.findByText('AI service is temporarily unavailable. Please try again later.'),
      ).toBeInTheDocument()
      expect(screen.queryByText('Preview Generated Questions')).not.toBeInTheDocument()
    })

    it('shows the unavailable message when the response carries no error code', async () => {
      server.use(
        http.post(GENERATE_URL, () => HttpResponse.text('upstream down', { status: 502 })),
      )
      renderDialog()
      await fillFormWithCategory()
      fireEvent.click(screen.getByRole('button', { name: 'AI Generate' }))

      expect(
        await screen.findByText('AI service is temporarily unavailable. Please try again later.'),
      ).toBeInTheDocument()
    })
  })

  describe('draft preview', () => {
    it('lists each generated draft with its type and difficulty, all pre-selected', async () => {
      renderDialog()
      await generateDrafts()

      expect(screen.getByText('What is 2 plus 2?')).toBeInTheDocument()
      expect(screen.getByText('Pick the prime numbers')).toBeInTheDocument()
      expect(screen.getByText('The Earth is flat')).toBeInTheDocument()
      expect(screen.getByText('single_choice · easy')).toBeInTheDocument()
      expect(screen.getByText('multiple_choice · medium')).toBeInTheDocument()
      expect(screen.getByText('true_false · hard')).toBeInTheDocument()

      expect(screen.getByRole('checkbox', { name: 'Select draft question 1' })).toBeChecked()
      expect(screen.getByRole('checkbox', { name: 'Select draft question 2' })).toBeChecked()
      expect(screen.getByRole('checkbox', { name: 'Select draft question 3' })).toBeChecked()
      expect(screen.getByRole('button', { name: 'Confirm Selected (3)' })).toBeEnabled()
      // Everything is selected, so Select All is hidden.
      expect(screen.queryByRole('button', { name: 'Select All' })).not.toBeInTheDocument()
    })

    it('expands a draft to show explanation, options and tags, and collapses it again', async () => {
      renderDialog()
      await generateDrafts()

      expect(screen.queryByText('Basic addition')).not.toBeInTheDocument()

      const expandButtons = screen.getAllByRole('button', { name: '▼' })
      fireEvent.click(expandButtons[0])

      expect(screen.getByText('Basic addition')).toBeInTheDocument()
      expect(screen.getByText('✓ 4')).toBeInTheDocument()
      expect(screen.getByText('○ 3')).toBeInTheDocument()
      expect(screen.getByText('math, arithmetic')).toBeInTheDocument()

      fireEvent.click(screen.getByRole('button', { name: '▲' }))
      expect(screen.queryByText('Basic addition')).not.toBeInTheDocument()
    })

    it('expands only one draft at a time', async () => {
      renderDialog()
      await generateDrafts()

      // Capture the toggles once: after the first click its label flips to ▲.
      const [firstToggle, secondToggle] = screen.getAllByRole('button', { name: '▼' })
      fireEvent.click(firstToggle)
      fireEvent.click(secondToggle)

      expect(screen.queryByText('Basic addition')).not.toBeInTheDocument()
      expect(screen.getByText('Primes have exactly two divisors')).toBeInTheDocument()
    })

    it('toggles a draft selection and updates the confirm count', async () => {
      renderDialog()
      await generateDrafts()

      const first = screen.getByRole('checkbox', { name: 'Select draft question 1' })
      fireEvent.click(first)
      expect(first).not.toBeChecked()
      expect(screen.getByRole('button', { name: 'Confirm Selected (2)' })).toBeEnabled()
      // Once a draft is deselected, Select All is offered again.
      expect(screen.getByRole('button', { name: 'Select All' })).toBeInTheDocument()

      fireEvent.click(first)
      expect(first).toBeChecked()
      expect(screen.getByRole('button', { name: 'Confirm Selected (3)' })).toBeEnabled()
      expect(screen.queryByRole('button', { name: 'Select All' })).not.toBeInTheDocument()
    })

    it('Select All re-selects every draft after some were deselected', async () => {
      renderDialog()
      await generateDrafts()

      fireEvent.click(screen.getByRole('checkbox', { name: 'Select draft question 2' }))
      fireEvent.click(screen.getByRole('checkbox', { name: 'Select draft question 3' }))
      expect(screen.getByRole('button', { name: 'Confirm Selected (1)' })).toBeEnabled()

      fireEvent.click(screen.getByRole('button', { name: 'Select All' }))

      expect(screen.getByRole('checkbox', { name: 'Select draft question 1' })).toBeChecked()
      expect(screen.getByRole('checkbox', { name: 'Select draft question 2' })).toBeChecked()
      expect(screen.getByRole('checkbox', { name: 'Select draft question 3' })).toBeChecked()
      expect(screen.getByRole('button', { name: 'Confirm Selected (3)' })).toBeEnabled()
    })

    it('disables confirm when no draft is selected', async () => {
      renderDialog()
      await generateDrafts()

      for (const n of [1, 2, 3]) {
        fireEvent.click(screen.getByRole('checkbox', { name: `Select draft question ${n}` }))
      }
      expect(screen.getByRole('button', { name: 'Confirm Selected (0)' })).toBeDisabled()
    })

    it('returns to the generation form when the reset control is clicked', async () => {
      renderDialog()
      await generateDrafts()

      fireEvent.click(screen.getByRole('button', { name: '↺' }))

      expect(screen.queryByText('Preview Generated Questions')).not.toBeInTheDocument()
      expect(categorySelect()).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'AI Generate' })).toBeInTheDocument()
    })
  })

  describe('confirm and create questions', () => {
    it('creates each selected draft with the mapped question type and hands the count to the caller', async () => {
      const created: unknown[] = []
      server.use(
        http.post(QUESTIONS_URL, async ({ request }) => {
          created.push(await request.json())
          return HttpResponse.json({ data: { id: 'q-new' }, error: null }, { status: 201 })
        }),
      )
      const { onClose, onSuccess } = renderDialog()
      await generateDrafts()

      fireEvent.click(screen.getByRole('button', { name: 'Confirm Selected (3)' }))

      await waitFor(() => expect(onSuccess).toHaveBeenCalledWith(3))
      expect(onClose).toHaveBeenCalledTimes(1)

      expect(created).toEqual([
        {
          type: 'single',
          difficulty: 'easy',
          category_id: 'cat-1',
          default_locale: 'en',
          translations: {
            en: { stem: 'What is 2 plus 2?', explanation: 'Basic addition' },
          },
          answer_options: [
            { sort_order: 0, is_correct: false, translations: { en: { text: '3' } } },
            { sort_order: 1, is_correct: true, translations: { en: { text: '4' } } },
          ],
          tag_ids: [],
        },
        {
          type: 'multiple',
          difficulty: 'medium',
          category_id: 'cat-1',
          default_locale: 'en',
          translations: {
            en: {
              stem: 'Pick the prime numbers',
              explanation: 'Primes have exactly two divisors',
            },
          },
          answer_options: [
            { sort_order: 0, is_correct: true, translations: { en: { text: '2' } } },
            { sort_order: 1, is_correct: false, translations: { en: { text: '4' } } },
            { sort_order: 2, is_correct: true, translations: { en: { text: '5' } } },
          ],
          tag_ids: [],
        },
        {
          type: 'truefalse',
          difficulty: 'hard',
          category_id: 'cat-1',
          default_locale: 'en',
          translations: {
            en: { stem: 'The Earth is flat', explanation: 'The Earth is an oblate spheroid' },
          },
          answer_options: [
            { sort_order: 0, is_correct: false, translations: { en: { text: 'True' } } },
            { sort_order: 1, is_correct: true, translations: { en: { text: 'False' } } },
          ],
          tag_ids: [],
        },
      ])
    })

    it('creates only the drafts that are still selected', async () => {
      const created: { stem: string }[] = []
      server.use(
        http.post(QUESTIONS_URL, async ({ request }) => {
          const body = (await request.json()) as { translations: { en: { stem: string } } }
          created.push({ stem: body.translations.en.stem })
          return HttpResponse.json({ data: { id: 'q-new' }, error: null }, { status: 201 })
        }),
      )
      const { onSuccess } = renderDialog()
      await generateDrafts()

      fireEvent.click(screen.getByRole('checkbox', { name: 'Select draft question 2' }))
      fireEvent.click(screen.getByRole('button', { name: 'Confirm Selected (2)' }))

      await waitFor(() => expect(onSuccess).toHaveBeenCalledWith(2))
      expect(created.map((c) => c.stem)).toEqual(['What is 2 plus 2?', 'The Earth is flat'])
    })

    it('shows confirm progress while questions are being created', async () => {
      let release!: () => void
      const gate = new Promise<void>((resolve) => {
        release = resolve
      })
      server.use(
        http.post(QUESTIONS_URL, async () => {
          await gate
          return HttpResponse.json({ data: { id: 'q-new' }, error: null }, { status: 201 })
        }),
      )
      const { onSuccess } = renderDialog()
      await generateDrafts()

      fireEvent.click(screen.getByRole('button', { name: 'Confirm Selected (3)' }))

      const progress = await screen.findByRole('button', { name: 'Processing 0 / 3' })
      expect(progress).toBeDisabled()

      release()
      await waitFor(() => expect(onSuccess).toHaveBeenCalledWith(3))
    })

    it('keeps going after one create fails and reports only the questions that were created', async () => {
      let call = 0
      server.use(
        http.post(QUESTIONS_URL, () => {
          call += 1
          if (call === 2) {
            return HttpResponse.json(
              { data: null, error: { code: 'VALIDATION', message: 'bad' } },
              { status: 422 },
            )
          }
          return HttpResponse.json({ data: { id: 'q-new' }, error: null }, { status: 201 })
        }),
      )
      const { onClose, onSuccess } = renderDialog()
      await generateDrafts()

      fireEvent.click(screen.getByRole('button', { name: 'Confirm Selected (3)' }))

      await waitFor(() => expect(onSuccess).toHaveBeenCalledWith(2))
      expect(call).toBe(3)
      expect(onClose).toHaveBeenCalledTimes(1)
    })

    it('shows an error and stays open when every create fails, without reporting success (#338)', async () => {
      server.use(
        http.post(QUESTIONS_URL, () =>
          HttpResponse.json({ data: null, error: { code: 'VALIDATION', message: 'bad' } }, { status: 422 }),
        ),
      )
      const { onClose, onSuccess } = renderDialog()
      await generateDrafts()

      fireEvent.click(screen.getByRole('button', { name: 'Confirm Selected (3)' }))

      expect(await screen.findByText('An unexpected error occurred. Please try again.')).toBeInTheDocument()
      expect(onSuccess).not.toHaveBeenCalled()
      expect(onClose).not.toHaveBeenCalled()
      // Drafts stay selectable and confirm is re-enabled so the user can retry.
      expect(screen.getByRole('button', { name: 'Confirm Selected (3)' })).toBeEnabled()
    })
  })
})
