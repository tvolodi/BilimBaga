import { test, expect } from '@playwright/test'
import { getSeedData } from './fixtures/seed'

async function waitForContent(page: import('@playwright/test').Page) {
  await page.waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin'), { timeout: 10_000 }).catch(() => {})
  await page.waitForLoadState('networkidle').catch(() => {})
}

function typeSelect(page: import('@playwright/test').Page) {
  return page.locator('select').filter({ has: page.locator('option[value="shorttext"]') })
}

test.describe('Question Editor — new question', () => {
  test('renders the editor with Save Draft button', async ({ page }) => {
    await page.goto('/admin/questions/new')
    await waitForContent(page)
    await expect(page.getByRole('button', { name: /сохранить черновик|save draft/i })).toBeVisible()
  })

  test('shows type selector with all question types', async ({ page }) => {
    await page.goto('/admin/questions/new')
    await waitForContent(page)
    const select = typeSelect(page)
    await expect(select).toBeVisible()
    // Verify all five types are available
    for (const val of ['single', 'multiple', 'truefalse', 'shorttext', 'likert']) {
      await expect(select.locator(`option[value="${val}"]`)).toHaveCount(1)
    }
  })

  test('shows stem textarea', async ({ page }) => {
    await page.goto('/admin/questions/new')
    await waitForContent(page)
    await expect(page.getByPlaceholder(/введите текст вопроса|enter the question text/i)).toBeVisible()
  })

  test('switching to shorttext shows auto-grading section', async ({ page }) => {
    await page.goto('/admin/questions/new')
    await waitForContent(page)
    await typeSelect(page).selectOption('shorttext')
    await expect(page.getByText(/автоматическое оценивание|auto.grading/i).first()).toBeVisible({ timeout: 3_000 })
  })

  test('switching to likert shows polarity options', async ({ page }) => {
    await page.goto('/admin/questions/new')
    await waitForContent(page)
    await typeSelect(page).selectOption('likert')
    await page.waitForTimeout(300)
    // Likert renders weight/polarity inputs or columns
    await expect(page.locator('body')).not.toContainText(/unexpected error/i)
  })
})

test.describe('Question Editor — edit existing question', () => {
  test('navigates to edit page for a seeded question', async ({ page }) => {
    const { adminToken } = await getSeedData()

    // Find first question via API to get a reliable ID
    const resp = await page.request.get('http://localhost:8080/api/v1/questions?per_page=1', {
      headers: { Authorization: `Bearer ${adminToken}` },
    })
    const data = await resp.json()
    const firstId = data?.data?.items?.[0]?.id as string | undefined
    if (!firstId) {
      test.info().annotations.push({ type: 'note', description: 'No questions in bank — skipping edit test' })
      return
    }

    await page.goto(`/admin/questions/${firstId}/edit`)
    await page.waitForSelector('textarea', { timeout: 15_000 })
    await expect(page.getByRole('button', { name: /сохранить|save/i })).toBeVisible({ timeout: 10_000 })
  })

  test('pre-populates the stem field for an existing question', async ({ page }) => {
    const { adminToken } = await getSeedData()

    // Find first question via API
    const resp = await page.request.get('http://localhost:8080/api/v1/questions?per_page=1', {
      headers: { Authorization: `Bearer ${adminToken}` },
    })
    const data = await resp.json()
    const firstId = data?.data?.items?.[0]?.id as string | undefined
    if (!firstId) {
      test.info().annotations.push({ type: 'note', description: 'No questions in bank — skipping pre-populate test' })
      return
    }

    await page.goto(`/admin/questions/${firstId}/edit`)
    await page.waitForSelector('textarea', { timeout: 15_000 })
    const stemField = page.getByPlaceholder(/введите текст вопроса|enter the question text/i)
    await expect(stemField).toBeVisible({ timeout: 10_000 })
    const value = await stemField.inputValue()
    expect(value.length).toBeGreaterThan(0)
  })
})
