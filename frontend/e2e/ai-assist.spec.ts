/**
 * AI question-generation assist E2E (FR-BB71, issue #16).
 *
 * Requires: make dev running, admin (super_admin) storage state from global-setup.ts.
 * Runs in the chromium-live-admin project.
 *
 * The Anthropic-backed endpoint is always stubbed (POST /api/v1/admin/ai/generate-questions):
 * a live LLM call is non-deterministic, billable and rate limited. POST /api/v1/questions
 * (the "confirm selected" persistence step) is stubbed as well so no questions are created
 * in the shared database. Both stubs are method-scoped; GET requests pass through.
 */
import { test, expect, type Page } from '@playwright/test'

const GENERATE_API = /\/api\/v1\/admin\/ai\/generate-questions$/
const QUESTIONS_API = /\/api\/v1\/questions$/

const DRAFTS = [
  {
    type: 'single_choice',
    difficulty: 'hard',
    stem: 'E2E AI draft one: which port does the API use?',
    explanation: 'The API listens on 8080 by default.',
    options: [
      { text: '8080', is_correct: true },
      { text: '9999', is_correct: false },
    ],
    tags: ['e2e-ai'],
  },
  {
    type: 'true_false',
    difficulty: 'hard',
    stem: 'E2E AI draft two: the sky is green.',
    explanation: 'It is not.',
    options: [
      { text: 'True', is_correct: false },
      { text: 'False', is_correct: true },
    ],
    tags: [],
  },
]

async function openDialog(page: Page) {
  await page.goto('/admin/questions')
  const trigger = page.getByRole('button', { name: /генерация ии|ai generate/i })
  await expect(trigger).toBeVisible({ timeout: 20_000 })
  await trigger.click()
  const dialog = page.getByRole('dialog')
  await expect(dialog.getByRole('heading', { name: /генерация вопросов с ии|generate questions with ai/i })).toBeVisible()
  return dialog
}

test.describe('AI question generation assist (FR-BB71)', () => {
  test('requires a category before generating (no API call)', async ({ page }) => {
    let called = false
    await page.route(GENERATE_API, (route) => {
      called = true
      return route.abort()
    })
    const dialog = await openDialog(page)
    await dialog.getByRole('button', { name: /^генерация ии$|^ai generate$/i }).click()
    await expect(dialog.getByText(/выберите категорию|select a category/i)).toBeVisible()
    expect(called).toBe(false)
  })

  test('generates drafts, lets the user pick some, and confirms the selection', async ({ page }) => {
    let generateBody: Record<string, unknown> | null = null
    let generateAuth: string | undefined
    const created: Array<Record<string, unknown>> = []

    await page.route(GENERATE_API, async (route) => {
      generateBody = route.request().postDataJSON() as Record<string, unknown>
      generateAuth = route.request().headers()['authorization']
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { questions: DRAFTS }, error: null }),
      })
    })
    await page.route(QUESTIONS_API, async (route) => {
      if (route.request().method() !== 'POST') return route.fallback()
      created.push(route.request().postDataJSON() as Record<string, unknown>)
      return route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({ data: { id: `00000000-0000-4000-8000-00000000000${created.length}` }, error: null }),
      })
    })

    const dialog = await openDialog(page)
    const selects = dialog.locator('select')
    // First option is the "all categories" placeholder; pick the first real category.
    const categoryOption = selects.nth(0).locator('option').nth(1)
    const categoryId = await categoryOption.getAttribute('value')
    expect(categoryId).toBeTruthy()
    await selects.nth(0).selectOption(categoryId as string)
    await selects.nth(1).selectOption('hard')
    await dialog.getByRole('spinbutton').fill('2')
    await dialog.locator('textarea').fill('Context for the e2e run')

    await dialog.getByRole('button', { name: /^генерация ии$|^ai generate$/i }).click()

    // Request payload contract; the endpoint is JWT protected so the Bearer header is required.
    expect(generateAuth).toMatch(/^Bearer /)
    await expect.poll(() => generateBody).not.toBeNull()
    expect(generateBody).toMatchObject({
      category_id: categoryId,
      difficulty: 'hard',
      count: 2,
      context_text: 'Context for the e2e run',
    })

    // Drafts preview: all pre-selected.
    await expect(dialog.getByText(DRAFTS[0].stem)).toBeVisible()
    await expect(dialog.getByText(DRAFTS[1].stem)).toBeVisible()
    const checks = dialog.getByRole('checkbox')
    await expect(checks).toHaveCount(2)
    await expect(checks.nth(0)).toBeChecked()
    await expect(checks.nth(1)).toBeChecked()
    const confirm = dialog.getByRole('button', { name: /(подтвердить выбранные|confirm selected) \(2\)/i })
    await expect(confirm).toBeVisible()

    // Deselect the second draft -> confirm counter follows.
    await checks.nth(1).uncheck()
    const confirmOne = dialog.getByRole('button', { name: /(подтвердить выбранные|confirm selected) \(1\)/i })
    await expect(confirmOne).toBeVisible()

    await confirmOne.click()
    await expect(dialog).toBeHidden({ timeout: 15_000 })
    expect(created).toHaveLength(1)
    expect(created[0]).toMatchObject({ type: 'single', category_id: categoryId, difficulty: 'hard' })
    expect(JSON.stringify(created[0])).toContain(DRAFTS[0].stem)
  })

  test('shows the rate-limit message on AI_RATE_LIMITED', async ({ page }) => {
    await page.route(GENERATE_API, (route) =>
      route.fulfill({
        status: 429,
        contentType: 'application/json',
        body: JSON.stringify({ data: null, error: { code: 'AI_RATE_LIMITED', message: 'rate limited' } }),
      }),
    )
    const dialog = await openDialog(page)
    const selects = dialog.locator('select')
    await selects.nth(0).selectOption({ index: 1 })
    await dialog.getByRole('button', { name: /^генерация ии$|^ai generate$/i }).click()
    await expect(dialog.getByText(/достигли лимита генерации|reached the ai generation limit/i)).toBeVisible()
  })

  test('shows the unavailable message when the AI service fails', async ({ page }) => {
    await page.route(GENERATE_API, (route) =>
      route.fulfill({
        status: 503,
        contentType: 'application/json',
        body: JSON.stringify({ data: null, error: { code: 'AI_UNAVAILABLE', message: 'down' } }),
      }),
    )
    const dialog = await openDialog(page)
    await dialog.locator('select').nth(0).selectOption({ index: 1 })
    await dialog.getByRole('button', { name: /^генерация ии$|^ai generate$/i }).click()
    await expect(dialog.getByText(/временно недоступен|temporarily unavailable/i)).toBeVisible()
  })
})
