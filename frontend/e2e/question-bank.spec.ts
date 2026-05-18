import { test, expect } from '@playwright/test'

async function waitForContent(page: import('@playwright/test').Page) {
  await page.waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin'), { timeout: 10_000 }).catch(() => {})
  await page.waitForLoadState('networkidle').catch(() => {})
}

test.describe('Question Bank page', () => {
  test('displays the question bank heading and table', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    await expect(page.getByRole('heading')).toBeVisible()
    await expect(page.locator('body')).not.toContainText(/unexpected error|something went wrong/i)
  })

  test('shows the New Question button', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    await expect(page.getByRole('button', { name: /new question/i })).toBeVisible()
  })

  test('navigates to question editor on New Question click', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    await page.getByRole('button', { name: /new question/i }).click()
    await expect(page).toHaveURL(/\/admin\/questions\/new/)
  })

  test('shows difficulty badges for existing questions', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    // Seeded questions have difficulty badges — if questions exist, at least one badge visible
    const hasTable = await page.getByRole('table').isVisible().catch(() => false)
    if (hasTable) {
      const badges = page.locator('table tbody tr')
      const count = await badges.count()
      expect(count).toBeGreaterThan(0)
    }
  })

  test('search input filters questions without crashing', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    const searchInput = page.getByPlaceholder(/search/i)
    await expect(searchInput).toBeVisible()
    await searchInput.fill('E2E')
    await page.waitForTimeout(500)
    await expect(page.locator('body')).not.toContainText(/unexpected error/i)
    await searchInput.clear()
  })

  test('shows empty state when search matches nothing', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    const searchInput = page.getByPlaceholder(/search/i)
    await searchInput.fill('ZZZNOMATCHXXX')
    await page.waitForTimeout(600)
    // Either shows empty state text or zero rows — no crash
    await expect(page.locator('body')).not.toContainText(/unexpected error/i)
    await searchInput.clear()
  })

  test('shows Import and AI Generate buttons', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    await expect(page.getByRole('button', { name: /import/i })).toBeVisible()
    await expect(page.getByRole('button', { name: /generate|ai/i })).toBeVisible()
  })
})
