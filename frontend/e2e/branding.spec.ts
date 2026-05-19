import { test, expect } from '@playwright/test'

async function waitForContent(page: import('@playwright/test').Page) {
  await page.waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin'), { timeout: 10_000 }).catch(() => {})
  await page.waitForLoadState('networkidle').catch(() => {})
}

test.describe('Branding settings page', () => {
  test('displays the branding settings form', async ({ page }) => {
    await page.goto('/admin/settings/branding')
    await page.waitForSelector('#branding-app-name', { timeout: 15_000 })
    await expect(page.locator('#branding-app-name')).toBeVisible()
  })

  test('shows primary color field', async ({ page }) => {
    await page.goto('/admin/settings/branding')
    await page.waitForSelector('#branding-app-name', { timeout: 15_000 })
    // Color input is present (either color or text type)
    const colorInput = page.locator('input[type="color"]').first()
    await expect(colorInput).toBeVisible()
  })

  test('save button is present', async ({ page }) => {
    await page.goto('/admin/settings/branding')
    await page.waitForSelector('#branding-app-name', { timeout: 15_000 })
    await expect(page.getByRole('button', { name: /сохранить|save/i })).toBeVisible()
  })

  test('can update app name and save', async ({ page }) => {
    await page.goto('/admin/settings/branding')
    await waitForContent(page)
    await page.waitForSelector('#branding-app-name', { timeout: 15_000 })
    const appName = page.locator('#branding-app-name')
    const originalValue = await appName.inputValue()
    // Use a value distinct from the current DB value to guarantee onChange fires
    const testValue = originalValue === 'BilimBaga Test' ? 'BilimBaga Alt' : 'BilimBaga Test'
    const saveBtn = page.getByRole('button', { name: /сохранить|save/i })
    await appName.fill(testValue)
    await page.waitForTimeout(300)
    await expect(saveBtn).toBeEnabled({ timeout: 5_000 })
    await saveBtn.click()
    await expect(page.locator('body')).not.toContainText(/unexpected error|save failed/i)
    // Wait for save mutation to complete before filling restore value
    await page.waitForLoadState('networkidle').catch(() => {})
    await page.waitForTimeout(300)
    // Restore to a known-clean value different from testValue
    const restoreValue = originalValue && originalValue !== testValue ? originalValue : 'BilimBaga'
    await appName.fill(restoreValue)
    await expect(saveBtn).toBeEnabled({ timeout: 5_000 })
    await saveBtn.click()
  })
})
