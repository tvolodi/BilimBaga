import { test, expect } from '@playwright/test'
import { mockRefreshSuccess } from './fixtures/helpers'
import { mockMe, mockTenantConfig, mockCategories, mockTags, mockQuestions, adminUser } from './fixtures/api'

test.describe('Question Bank page', () => {
  test.beforeEach(async ({ page }) => {
    await mockRefreshSuccess(page)
    await mockMe(page, adminUser)
    await mockTenantConfig(page)
    await mockCategories(page)
    await mockTags(page)
    await mockQuestions(page)
  })

  test('displays question rows', async ({ page }) => {
    await page.goto('/admin/questions')
    await expect(page.getByText('What is H2O?')).toBeVisible()
  })

  test('shows the difficulty badge', async ({ page }) => {
    await page.goto('/admin/questions')
    await expect(page.getByText('What is H2O?')).toBeVisible()
    await expect(page.locator('text=/easy/i').first()).toBeVisible()
  })

  test('shows the New Question button', async ({ page }) => {
    await page.goto('/admin/questions')
    await expect(page.getByRole('button', { name: /new question/i })).toBeVisible()
  })

  test('navigates to question editor on New Question click', async ({ page }) => {
    await page.goto('/admin/questions')
    await page.getByRole('button', { name: /new question/i }).click()
    await expect(page).toHaveURL(/\/admin\/questions\/new/)
  })

  test('search input updates without crash', async ({ page }) => {
    await page.goto('/admin/questions')
    await page.getByPlaceholder(/search/i).fill('biology')
    await expect(page.getByPlaceholder(/search/i)).toHaveValue('biology')
  })

  test('shows empty state when no questions returned', async ({ page }) => {
    await mockQuestions(page, [])
    await page.goto('/admin/questions')
    await expect(page.getByText('What is H2O?')).not.toBeVisible()
  })
})
