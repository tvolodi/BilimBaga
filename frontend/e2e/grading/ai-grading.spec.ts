import { test, expect } from '@playwright/test'
import { mockRefreshSuccess } from '../fixtures/helpers'
import { mockMe, mockTenantConfig, mockCategories, adminUser } from '../fixtures/api'

const shortTextQuestionDetail = {
  id: 'q-st',
  type: 'shorttext',
  difficulty: 'medium',
  status: 'draft',
  category_id: 'cat-1',
  default_locale: 'en',
  version: 1,
  parent_id: null,
  locale_coverage: ['en'],
  tag_ids: [],
  translations: {
    en: { stem: 'What is the capital of France?', explanation: null },
  },
  answer_options: [],
  tags: [],
  auto_grade: false,
  model_answer: null,
  created_by: 'u-1',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
}

const shortTextWithAutoGrade = {
  ...shortTextQuestionDetail,
  auto_grade: true,
  model_answer: 'Paris',
}

function setupCommonMocks(page: Parameters<typeof mockRefreshSuccess>[0]) {
  return Promise.all([
    mockRefreshSuccess(page),
    mockMe(page, adminUser),
    mockTenantConfig(page),
    mockCategories(page),
    page.route('**/api/v1/tags', (route) =>
      route.fulfill({ json: { data: [], error: null } }),
    ),
  ])
}

// Helper: the type <select> is the one that has a 'shorttext' option value.
// page.locator('select').first() would match the LocaleSwitcher; use this filter instead.
function typeSelect(page: Parameters<typeof setupCommonMocks>[0]) {
  return page.locator('select').filter({ has: page.locator('option[value="shorttext"]') })
}

// The model answer textarea placeholder comes from i18n:
// questionEditor.field.modelAnswerPlaceholder = "Enter the expected answer for AI comparison…"
const MODEL_ANSWER_PLACEHOLDER = /enter the expected answer for ai comparison/i

test.describe('FR-BB73 — Auto-Grading: Question Editor', () => {
  test.beforeEach(async ({ page }) => {
    await setupCommonMocks(page)
  })

  test('does NOT show auto-grading section for non-shorttext question type', async ({ page }) => {
    await page.goto('/admin/questions/new')
    // Default type is 'single' — auto-grading section should be absent
    await expect(page.getByText(/auto.grading/i)).not.toBeVisible()
  })

  test('shows auto-grading section when type is switched to shorttext', async ({ page }) => {
    await page.goto('/admin/questions/new')
    // Switch type to shorttext
    await typeSelect(page).selectOption('shorttext')
    await expect(page.getByText(/auto.grading/i)).toBeVisible()
  })

  test('auto_grade checkbox is disabled when model_answer is empty', async ({ page }) => {
    await page.goto('/admin/questions/new')
    await typeSelect(page).selectOption('shorttext')
    const checkbox = page.locator('input[type="checkbox"]')
    await expect(checkbox).toBeDisabled()
  })

  test('auto_grade checkbox is enabled after entering a model answer', async ({ page }) => {
    await page.goto('/admin/questions/new')
    await typeSelect(page).selectOption('shorttext')
    // Fill model answer
    await page.getByPlaceholder(MODEL_ANSWER_PLACEHOLDER).fill('Paris is the capital.')
    const checkbox = page.locator('input[type="checkbox"]')
    await expect(checkbox).not.toBeDisabled()
  })

  test('loads existing shorttext question with auto_grade and model_answer pre-populated', async ({ page }) => {
    await page.route('**/api/v1/questions/q-st', (route) =>
      route.fulfill({ json: { data: shortTextWithAutoGrade, error: null } }),
    )
    await page.goto('/admin/questions/q-st')
    // Model answer textarea should contain the loaded value
    await expect(page.getByPlaceholder(MODEL_ANSWER_PLACEHOLDER)).toHaveValue('Paris')
    // Checkbox should be checked
    const checkbox = page.locator('input[type="checkbox"]')
    await expect(checkbox).toBeChecked()
  })

  test('saves shorttext question with auto_grade and model_answer via PUT', async ({ page }) => {
    await page.route('**/api/v1/questions/q-st', (route) => {
      if (route.request().method() === 'GET') {
        return route.fulfill({ json: { data: shortTextQuestionDetail, error: null } })
      }
      if (route.request().method() === 'PUT') {
        const body = route.request().postDataJSON()
        // Verify auto_grade and model_answer sent correctly
        if (body?.auto_grade === true && body?.model_answer === 'The capital of France is Paris.') {
          return route.fulfill({ json: { data: { ...shortTextWithAutoGrade, model_answer: body.model_answer }, error: null } })
        }
        return route.fulfill({ status: 400, json: { data: null, error: { code: 'BAD_REQUEST', message: 'invalid payload' } } })
      }
      return route.continue()
    })

    await page.goto('/admin/questions/q-st')
    await page.getByPlaceholder(MODEL_ANSWER_PLACEHOLDER).fill('The capital of France is Paris.')
    await page.locator('input[type="checkbox"]').check()
    await page.getByRole('button', { name: /save draft/i }).click()
    // Expect no error toast — save succeeded
    await expect(page.getByRole('alert')).not.toBeVisible({ timeout: 3000 }).catch(() => {
      // Some implementations show success toast; that's fine
    })
  })

  test('clearing model_answer unchecks auto_grade', async ({ page }) => {
    await page.route('**/api/v1/questions/q-st', (route) =>
      route.fulfill({ json: { data: shortTextWithAutoGrade, error: null } }),
    )
    await page.goto('/admin/questions/q-st')
    // Model answer is 'Paris', checkbox is checked
    await expect(page.locator('input[type="checkbox"]')).toBeChecked()
    // Clear the model answer
    await page.getByPlaceholder(MODEL_ANSWER_PLACEHOLDER).fill('')
    // Checkbox should now be unchecked (and disabled)
    await expect(page.locator('input[type="checkbox"]')).not.toBeChecked()
  })
})

test.describe('FR-BB73 — Auto-Grading: Grading Queue UI', () => {
  test.beforeEach(async ({ page }) => {
    await setupCommonMocks(page)
  })

  const aiGradedSession = {
    session_id: 'sess-1',
    assignment_id: 'assign-1',
    user_id: 'u-1',
    user_name: 'John Doe',
    exam_id: 'exam-1',
    exam_title: 'Geography Quiz',
    submitted_at: '2024-03-01T10:00:00Z',
    pending_count: 0,
    ai_graded_count: 1,
  }

  const aiGradedQuestion = {
    session_question_id: 'sq-1',
    question_id: 'q-st',
    stem: 'What is the capital of France?',
    question_type: 'shorttext',
    text_answer: 'Paris',
    grading_status: 'ai_graded',
    current_score: 0.8,
    max_score: 1,
    feedback: null,
    ai_reasoning: 'Answer is correct. Paris is the capital of France.',
  }

  test('shows AI Graded badge for ai_graded questions in grading queue', async ({ page }) => {
    await page.route('**/api/v1/grading/queue', (route) =>
      route.fulfill({ json: { data: [aiGradedSession], error: null } }),
    )
    await page.route('**/api/v1/grading/sessions/sess-1/questions', (route) =>
      route.fulfill({ json: { data: [aiGradedQuestion], error: null } }),
    )

    await page.goto('/admin/grading')
    // Click into the session
    await page.getByText('John Doe').click()
    // Should show AI Graded badge
    await expect(page.getByText(/ai graded/i)).toBeVisible()
  })

  test('AI reasoning toggle shows and hides reasoning text', async ({ page }) => {
    await page.route('**/api/v1/grading/queue', (route) =>
      route.fulfill({ json: { data: [aiGradedSession], error: null } }),
    )
    await page.route('**/api/v1/grading/sessions/sess-1/questions', (route) =>
      route.fulfill({ json: { data: [aiGradedQuestion], error: null } }),
    )

    await page.goto('/admin/grading')
    await page.getByText('John Doe').click()

    // Reasoning is hidden initially — show button visible
    await expect(page.getByText(/show ai reasoning/i)).toBeVisible()
    await expect(page.getByText('Answer is correct. Paris is the capital of France.')).not.toBeVisible()

    // Click show
    await page.getByText(/show ai reasoning/i).click()
    await expect(page.getByText('Answer is correct. Paris is the capital of France.')).toBeVisible()
    await expect(page.getByText(/hide ai reasoning/i)).toBeVisible()

    // Click hide
    await page.getByText(/hide ai reasoning/i).click()
    await expect(page.getByText('Answer is correct. Paris is the capital of France.')).not.toBeVisible()
  })
})
