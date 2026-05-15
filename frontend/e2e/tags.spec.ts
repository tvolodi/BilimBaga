import { test, expect } from '@playwright/test'
import { mockRefreshSuccess } from './fixtures/helpers'
import { mockMe, mockTenantConfig, mockTags, adminUser } from './fixtures/api'

test.describe('Tags page', () => {
  test.beforeEach(async ({ page }) => {
    await mockRefreshSuccess(page)
    await mockMe(page, adminUser)
    await mockTenantConfig(page)
    await mockTags(page)
  })

  test('displays tags from the API', async ({ page }) => {
    await page.goto('/admin/tags')
    await expect(page.getByText('Biology')).toBeVisible()
    await expect(page.getByText('Chemistry')).toBeVisible()
  })

  test('shows usage counts', async ({ page }) => {
    await page.goto('/admin/tags')
    // Usage count cells in the table
    await expect(page.getByRole('cell', { name: '5' })).toBeVisible()
    await expect(page.getByRole('cell', { name: '3' })).toBeVisible()
  })

  test('shows New Tag button for super_admin', async ({ page }) => {
    await page.goto('/admin/tags')
    await expect(page.getByRole('button', { name: /new tag/i })).toBeVisible()
  })

  test('filters tags by search input', async ({ page }) => {
    await page.goto('/admin/tags')
    await expect(page.getByText('Biology')).toBeVisible()
    await page.getByRole('textbox').fill('Bio')
    await expect(page.getByText('Biology')).toBeVisible()
    await expect(page.getByText('Chemistry')).not.toBeVisible()
  })
})
