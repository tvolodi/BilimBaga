import { test, expect } from '@playwright/test'
import { mockRefreshSuccess } from './fixtures/helpers'
import { mockMe, mockTenantConfig, adminUser } from './fixtures/api'

test.describe('Branding settings page', () => {
  test.beforeEach(async ({ page }) => {
    await mockRefreshSuccess(page)
    await mockMe(page, adminUser)
    await mockTenantConfig(page)
  })

  test('displays the branding settings form', async ({ page }) => {
    await page.goto('/admin/settings/branding')
    await expect(page.locator('input[value="BilimBaga"]')).toBeVisible()
  })

  test('shows primary color field', async ({ page }) => {
    await page.goto('/admin/settings/branding')
    await expect(page.locator('input[value="#4F46E5"]')).toBeVisible()
  })

  test('save button is present', async ({ page }) => {
    await page.goto('/admin/settings/branding')
    await expect(page.getByRole('button', { name: /save/i })).toBeVisible()
  })
})
