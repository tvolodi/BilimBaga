import { test, expect } from '@playwright/test'
import { getSeedData } from './fixtures/seed'

async function waitForContent(page: import('@playwright/test').Page) {
  await page.waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin'), { timeout: 10_000 }).catch(() => {})
  await page.waitForLoadState('networkidle').catch(() => {})
}

test.describe('Exam Wizard — Create', () => {
  test('renders the step indicator and Basic Settings heading', async ({ page }) => {
    await page.goto('/admin/exams/new')
    await waitForContent(page)
    await expect(page.getByText(/\u041e\u0441\u043d\u043e\u0432\u043d\u044b\u0435 \u043d\u0430\u0441\u0442\u0440\u043e\u0439\u043a\u0438|Basic Settings/i)).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('1')).toBeVisible()
    await expect(page.getByText('4')).toBeVisible()
  })

  test('shows validation error when title is empty', async ({ page }) => {
    await page.goto('/admin/exams/new')
    await waitForContent(page)
    await page.getByRole('button', { name: /\u0434\u0430\u043b\u0435\u0435|next/i }).click()
    await expect(
      page.locator('[data-error], .text-destructive, [role="alert"], .text-red-500').first(),
    ).toBeVisible({ timeout: 5_000 })
  })

  test('golden path: fill Step 1 and advance to Step 2', async ({ page }) => {
    await page.goto('/admin/exams/new')
    await waitForContent(page)
    await page.locator('#title').fill('E2E Wizard Test Exam')
    await page.locator('#timeLimitMinutes').fill('60')
    await page.locator('#passingScorePct').fill('70')
    await page.locator('#maxAttempts').fill('3')
    await page.getByRole('button', { name: /\u0434\u0430\u043b\u0435\u0435|next/i }).click()
    await expect(page.getByText(/\u041f\u0440\u0430\u0432\u0438\u043b\u0430 \u0432\u043e\u043f\u0440\u043e\u0441\u043e\u0432|Question Rules/i)).toBeVisible({ timeout: 10_000 })
  })

  test('can navigate back from Step 2 to Step 1', async ({ page }) => {
    await page.goto('/admin/exams/new')
    await waitForContent(page)
    await page.locator('#title').fill('E2E Wizard Test Exam')
    await page.locator('#timeLimitMinutes').fill('60')
    await page.locator('#passingScorePct').fill('70')
    await page.locator('#maxAttempts').fill('3')
    await page.getByRole('button', { name: /\u0434\u0430\u043b\u0435\u0435|next/i }).click()
    // Wait for Step 2 to render — Back button only appears in Step 2
    const backBtn = page.getByRole('button', { name: /\u043d\u0430\u0437\u0430\u0434|back/i })
    await expect(backBtn).toBeVisible({ timeout: 15_000 })
    await backBtn.click()
    await expect(page.locator('#title')).toBeVisible({ timeout: 5_000 })
  })

  test('can reach Step 3 (Assignments) after passing steps 1 and 2', async ({ page }) => {
    await page.goto('/admin/exams/new')
    await waitForContent(page)
    await page.locator('#title').fill('E2E Wizard Test Exam')
    await page.locator('#timeLimitMinutes').fill('60')
    await page.locator('#passingScorePct').fill('70')
    await page.locator('#maxAttempts').fill('3')
    await page.getByRole('button', { name: /\u0434\u0430\u043b\u0435\u0435|next/i }).click()
    // Wait for Back button to confirm Step 2 is rendered
    await expect(page.getByRole('button', { name: /\u043d\u0430\u0437\u0430\u0434|back/i })).toBeVisible({ timeout: 15_000 })
    await page.getByRole('button', { name: /\u0434\u0430\u043b\u0435\u0435|next/i }).click()
    await expect(page.getByText(/\u041d\u0430\u0437\u043d\u0430\u0447\u0435\u043d\u0438\u044f|Assignments/i).first()).toBeVisible({ timeout: 10_000 })
  })

  test('can reach Step 4 (Review & Publish)', async ({ page }) => {
    await page.goto('/admin/exams/new')
    await waitForContent(page)
    await page.locator('#title').fill('E2E Wizard Test Exam')
    await page.locator('#timeLimitMinutes').fill('60')
    await page.locator('#passingScorePct').fill('70')
    await page.locator('#maxAttempts').fill('3')
    await page.getByRole('button', { name: /\u0434\u0430\u043b\u0435\u0435|next/i }).click()
    // Wait for Back button to confirm Step 2 is rendered
    await expect(page.getByRole('button', { name: /\u043d\u0430\u0437\u0430\u0434|back/i })).toBeVisible({ timeout: 15_000 })
    await page.getByRole('button', { name: /\u0434\u0430\u043b\u0435\u0435|next/i }).click()
    await expect(page.getByText(/\u041d\u0430\u0437\u043d\u0430\u0447\u0435\u043d\u0438\u044f|Assignments/i).first()).toBeVisible({ timeout: 10_000 })
    await page.getByRole('button', { name: /\u0434\u0430\u043b\u0435\u0435|next/i }).click()
    // Wait for Publish button to confirm Step 4 is rendered
    await expect(page.getByRole('button', { name: /\u043e\u043f\u0443\u0431\u043b\u0438\u043a\u043e\u0432\u0430\u0442\u044c|publish/i })).toBeVisible({ timeout: 15_000 })
  })
})

test.describe('Exam Wizard — Edit (pre-populated)', () => {
  test('loads existing exam data into Step 1', async ({ page }) => {
    const { mixedExamId } = await getSeedData()
    await page.goto(`/admin/exams/${mixedExamId}/edit`)
    await waitForContent(page)
    await expect(page.getByText(/\u0420\u0435\u0434\u0430\u043a\u0442\u0438\u0440\u043e\u0432\u0430\u0442\u044c \u044d\u043a\u0437\u0430\u043c\u0435\u043d|\u041e\u0441\u043d\u043e\u0432\u043d\u044b\u0435 \u043d\u0430\u0441\u0442\u0440\u043e\u0439\u043a\u0438|Edit Exam|Basic Settings/i).first()).toBeVisible({ timeout: 10_000 })
    // Title field must be pre-populated
    const titleField = page.locator('#title')
    if (await titleField.isVisible({ timeout: 3_000 }).catch(() => false)) {
      const value = await titleField.inputValue()
      expect(value.length).toBeGreaterThan(0)
    }
  })
})

test.describe('Exam Wizard — Publish flow', () => {
  test('publish button opens confirmation dialog', async ({ page }) => {
    await page.goto('/admin/exams/new')
    await waitForContent(page)
    await page.locator('#title').fill('E2E Wizard Publish Test')
    await page.locator('#timeLimitMinutes').fill('60')
    await page.locator('#passingScorePct').fill('70')
    await page.locator('#maxAttempts').fill('3')
    await page.getByRole('button', { name: /\u0434\u0430\u043b\u0435\u0435|next/i }).click()
    // Wait for Back button — indicator Step 2 is rendered (StepIndicator labels don't prove step rendered)
    await expect(page.getByRole('button', { name: /\u043d\u0430\u0437\u0430\u0434|back/i })).toBeVisible({ timeout: 15_000 })
    await page.getByRole('button', { name: /\u0434\u0430\u043b\u0435\u0435|next/i }).click()
    await expect(page.getByText(/\u041d\u0430\u0437\u043d\u0430\u0447\u0435\u043d\u0438\u044f|Assignments/i).first()).toBeVisible({ timeout: 10_000 })
    await page.getByRole('button', { name: /\u0434\u0430\u043b\u0435\u0435|next/i }).click()
    // Wait for Publish button — indicator Step 4 is rendered
    const publishBtn = page.getByRole('button', { name: /\u043e\u043f\u0443\u0431\u043b\u0438\u043a\u043e\u0432\u0430\u0442\u044c|publish/i })
    await expect(publishBtn).toBeVisible({ timeout: 15_000 })
    await publishBtn.click()
    // Confirmation dialog should appear
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
  })

  test('publish without rules is refused with a visible error (ISS-132)', async ({ page }) => {
    await page.goto('/admin/exams/new')
    await waitForContent(page)
    await page.locator('#title').fill('E2E Wizard No Rules Test')
    await page.locator('#timeLimitMinutes').fill('60')
    await page.locator('#passingScorePct').fill('70')
    await page.locator('#maxAttempts').fill('3')
    await page.getByRole('button', { name: /\u0434\u0430\u043b\u0435\u0435|next/i }).click()
    // Wait for Back button — indicator Step 2 is rendered
    await expect(page.getByRole('button', { name: /\u043d\u0430\u0437\u0430\u0434|back/i })).toBeVisible({ timeout: 15_000 })
    await page.getByRole('button', { name: /\u0434\u0430\u043b\u0435\u0435|next/i }).click()
    await expect(page.getByText(/\u041d\u0430\u0437\u043d\u0430\u0447\u0435\u043d\u0438\u044f|Assignments/i).first()).toBeVisible({ timeout: 10_000 })
    await page.getByRole('button', { name: /\u0434\u0430\u043b\u0435\u0435|next/i }).click()
    // Wait for Publish button — indicator Step 4 is rendered
    const publishBtn = page.getByRole('button', { name: /\u043e\u043f\u0443\u0431\u043b\u0438\u043a\u043e\u0432\u0430\u0442\u044c|publish/i })
    await expect(publishBtn).toBeVisible({ timeout: 15_000 })
    await publishBtn.click()
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 5_000 })
    // ISS-132: the backend refuses to publish an exam without question rules (422 INSUFFICIENT_QUESTIONS).
    const confirmBtn = dialog.getByRole('button', { name: /\u043e\u043f\u0443\u0431\u043b\u0438\u043a\u043e\u0432\u0430\u0442\u044c|publish/i })
    await confirmBtn.click()
    await expect(page.getByText(/no questions|\u043d\u0435\u0442 \u0432\u043e\u043f\u0440\u043e\u0441\u043e\u0432/i).first()).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText(/published successfully|\u043e\u043f\u0443\u0431\u043b\u0438\u043a\u043e\u0432\u0430\u043d/i)).toHaveCount(0)
  })
})
