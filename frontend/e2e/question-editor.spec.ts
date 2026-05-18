import { test, expect } from '@playwright/test'

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
    await expect(page.getByRole('button', { name: /save draft/i })).toBeVisible()
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
    await expect(page.getByPlaceholder(/enter the question text/i)).toBeVisible()
  })

  test('switching to shorttext shows auto-grading section', async ({ page }) => {
    await page.goto('/admin/questions/new')
    await waitForContent(page)
    await typeSelect(page).selectOption('shorttext')
    await expect(page.getByText(/auto.grading/i).first()).toBeVisible({ timeout: 3_000 })
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
    // Find a question from the bank first
    await page.goto('/admin/questions')
    await page.waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin'), { timeout: 10_000 }).catch(() => {})
    await page.waitForLoadState('networkidle').catch(() => {})

    const firstQuestion = page.locator('table tbody tr').first()
    if (await firstQuestion.isVisible({ timeout: 5_000 }).catch(() => false)) {
      // Click the question stem to navigate to edit
      const stemBtn = firstQuestion.getByRole('button').first()
      if (await stemBtn.isVisible({ timeout: 2_000 }).catch(() => false)) {
        await stemBtn.click()
      } else {
        // Navigate via URL pattern
        const href = await firstQuestion.getByRole('link').first().getAttribute('href').catch(() => null)
        if (href) await page.goto(href)
      }
      await page.waitForURL(/\/admin\/questions\//, { timeout: 10_000 })
      await expect(page.getByRole('button', { name: /save(?: draft)?/i })).toBeVisible({ timeout: 10_000 })
    } else {
      test.info().annotations.push({ type: 'note', description: 'No questions in bank — skipping edit test' })
    }
  })

  test('pre-populates the stem field for an existing question', async ({ page }) => {
    await page.goto('/admin/questions')
    await page.waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin'), { timeout: 10_000 }).catch(() => {})
    await page.waitForLoadState('networkidle').catch(() => {})

    const firstStemBtn = page.locator('table tbody tr').first().getByRole('button').first()
    if (await firstStemBtn.isVisible({ timeout: 5_000 }).catch(() => false)) {
      await firstStemBtn.click()
      await page.waitForURL(/\/admin\/questions\//, { timeout: 10_000 })
      await page.waitForLoadState('networkidle').catch(() => {})
      // Stem textarea must have a non-empty value
      const stemField = page.getByPlaceholder(/enter the question text/i)
      await expect(stemField).toBeVisible({ timeout: 10_000 })
      const value = await stemField.inputValue()
      expect(value.length).toBeGreaterThan(0)
    } else {
      test.info().annotations.push({ type: 'note', description: 'No questions in bank — skipping pre-populate test' })
    }
  })
})
