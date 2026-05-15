import { test, expect } from '@playwright/test'
import { mockRefreshSuccess } from './fixtures/helpers'
import { mockMe, mockTenantConfig, mockCategories, adminUser, sampleCategory } from './fixtures/api'

test.describe('Categories page', () => {
  test.beforeEach(async ({ page }) => {
    await mockRefreshSuccess(page)
    await mockMe(page, adminUser)
    await mockTenantConfig(page)
    await mockCategories(page)
  })

  test('displays the categories list', async ({ page }) => {
    await page.goto('/admin/categories')
    await expect(page.getByText('Science')).toBeVisible()
  })

  test('shows the New Category button for super_admin', async ({ page }) => {
    await page.goto('/admin/categories')
    await expect(page.getByRole('button', { name: /new category/i })).toBeVisible()
  })

  test('opens the create modal when New Category is clicked', async ({ page }) => {
    await page.route('**/api/v1/categories', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({ json: { data: { ...sampleCategory, id: 'cat-new', name: 'New Cat' }, error: null } })
      } else {
        route.fulfill({ json: { data: [sampleCategory], error: null } })
      }
    })
    await page.goto('/admin/categories')
    await expect(page.getByText('Science')).toBeVisible()
    await page.getByRole('button', { name: /new category/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible()
  })

  test('shows empty state when no categories exist', async ({ page }) => {
    await mockCategories(page, [])
    await page.goto('/admin/categories')
    await expect(page.locator('body')).toBeVisible()
    await expect(page.getByText('Science')).not.toBeVisible()
  })
})
