import { test, expect } from '@playwright/test'
import { mockLogin, mockMe, mockTenantConfig } from './fixtures/api'
import { mockRefreshExpired, mockRefreshSuccess } from './fixtures/helpers'

/**
 * FR-BB63 — Accessibility (WCAG 2.1 AA)
 * E2E tests for skip link and keyboard navigation.
 */

test.describe('Accessibility — Skip Link', () => {
  test.beforeEach(async ({ page }) => {
    await mockRefreshExpired(page)
    await mockTenantConfig(page)
  })

  test('skip link is the first focusable element on login page', async ({ page }) => {
    await page.goto('/login')
    // Tab once from the page body
    await page.keyboard.press('Tab')
    const focusedHref = await page.evaluate(() => (document.activeElement as HTMLAnchorElement)?.href)
    expect(focusedHref).toMatch(/#main-content$/)
  })

  test('skip link is visible when focused', async ({ page }) => {
    await page.goto('/login')
    await page.keyboard.press('Tab')
    const skipLink = page.locator('a[href="#main-content"]')
    await expect(skipLink).toBeVisible()
  })

  test('#main-content exists and has tabindex=-1 (required for skip link target)', async ({ page }) => {
    await page.goto('/login')
    const mainContent = page.locator('#main-content')
    await expect(mainContent).toBeAttached()
    const tabindex = await mainContent.getAttribute('tabindex')
    expect(tabindex).toBe('-1')
  })
})

test.describe('Accessibility — Admin Layout', () => {
  test.beforeEach(async ({ page }) => {
    await mockRefreshSuccess(page)
    await mockTenantConfig(page)
    await mockMe(page)
    await mockLogin(page, true)
    // Stub common endpoints to prevent network errors
    await page.route('**/api/v1/users*', (route) =>
      route.fulfill({ json: { data: { items: [], meta: { page: 1, per_page: 20, total: 0 } }, error: null } }),
    )
    await page.route('**/api/v1/audit*', (route) =>
      route.fulfill({ json: { data: { items: [], meta: { page: 1, per_page: 20, total: 0 } }, error: null } }),
    )
    await page.route('**/api/v1/dashboard*', (route) =>
      route.fulfill({ json: { data: {}, error: null } }),
    )
  })

  test('admin layout has #main-content landmark', async ({ page }) => {
    await page.goto('/admin')
    const main = page.locator('#main-content')
    await expect(main).toBeAttached()
  })

  test('skip link is the first focusable element on admin page', async ({ page }) => {
    await page.goto('/admin')
    await page.keyboard.press('Tab')
    const focusedHref = await page.evaluate(() => (document.activeElement as HTMLAnchorElement)?.href)
    expect(focusedHref).toMatch(/#main-content$/)
  })
})

test.describe('Accessibility — Error states', () => {
  test.beforeEach(async ({ page }) => {
    await mockRefreshExpired(page)
    await mockTenantConfig(page)
  })

  test('login page #main-content exists', async ({ page }) => {
    await page.goto('/login')
    const main = page.locator('#main-content')
    await expect(main).toBeAttached()
  })
})
