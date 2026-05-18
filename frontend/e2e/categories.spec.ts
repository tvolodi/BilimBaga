import { test, expect } from '@playwright/test'

async function waitForContent(page: import('@playwright/test').Page) {
  await page.waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin'), { timeout: 10_000 }).catch(() => {})
  await page.waitForLoadState('networkidle').catch(() => {})
}

test.describe('Categories page', () => {
  test('displays the categories list', async ({ page }) => {
    await page.goto('/admin/categories')
    await waitForContent(page)
    // At least one heading or category item must render — page must not crash
    await expect(page.getByRole('heading')).toBeVisible()
    await expect(page.locator('body')).not.toContainText(/unexpected error|something went wrong/i)
  })

  test('shows the New Category button for admin', async ({ page }) => {
    await page.goto('/admin/categories')
    await waitForContent(page)
    await expect(page.getByRole('button', { name: /new category|add category|create category/i })).toBeVisible()
  })

  test('opens the create modal when New Category is clicked', async ({ page }) => {
    await page.goto('/admin/categories')
    await waitForContent(page)
    await page.getByRole('button', { name: /new category|add category|create category/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
  })

  test('create modal has a name input field', async ({ page }) => {
    await page.goto('/admin/categories')
    await waitForContent(page)
    await page.getByRole('button', { name: /new category|add category|create category/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    const nameInput = page.getByRole('dialog').getByRole('textbox').first()
    await expect(nameInput).toBeVisible()
    await nameInput.fill('E2E Test Category')
    await expect(nameInput).toHaveValue('E2E Test Category')
    // Close without saving
    await page.keyboard.press('Escape')
  })
})
