/**
 * FR-BB63 — Accessibility (WCAG 2.1 AA)
 * E2E tests for skip link and keyboard navigation against the real stack.
 */

import { test, expect } from '@playwright/test'

async function waitForContent(page: import('@playwright/test').Page) {
  await page.waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin'), { timeout: 10_000 }).catch(() => {})
  await page.waitForLoadState('networkidle').catch(() => {})
}

test.describe('Accessibility — Skip Link (Login Page)', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test('skip link is the first focusable element on login page', async ({ page }) => {
    await page.goto('/login')
    await waitForContent(page)
    await page.keyboard.press('Tab')
    const focusedHref = await page.evaluate(() => (document.activeElement as HTMLAnchorElement)?.href)
    expect(focusedHref).toMatch(/#main-content$/)
  })

  test('skip link is visible when focused', async ({ page }) => {
    await page.goto('/login')
    await waitForContent(page)
    await page.keyboard.press('Tab')
    const skipLink = page.locator('a[href="#main-content"]')
    await expect(skipLink).toBeVisible()
  })

  test('#main-content exists and has tabindex=-1 (required for skip link target)', async ({ page }) => {
    await page.goto('/login')
    await waitForContent(page)
    const mainContent = page.locator('#main-content')
    await expect(mainContent).toBeAttached()
    const tabindex = await mainContent.getAttribute('tabindex')
    expect(tabindex).toBe('-1')
  })

  test('login page #main-content exists', async ({ page }) => {
    await page.goto('/login')
    await waitForContent(page)
    const main = page.locator('#main-content')
    await expect(main).toBeAttached()
  })
})

test.describe('Accessibility — Admin Layout', () => {
  test('admin layout has #main-content landmark', async ({ page }) => {
    await page.goto('/admin')
    await waitForContent(page)
    const main = page.locator('#main-content')
    await expect(main).toBeAttached()
  })

  test('skip link is the first focusable element on admin page', async ({ page }) => {
    await page.goto('/admin')
    await waitForContent(page)
    await page.keyboard.press('Tab')
    const focusedHref = await page.evaluate(() => (document.activeElement as HTMLAnchorElement)?.href)
    expect(focusedHref).toMatch(/#main-content$/)
  })

  test('admin pages render without accessibility-breaking crashes', async ({ page }) => {
    for (const path of ['/admin/users', '/admin/questions', '/admin/exams', '/admin/tags']) {
      await page.goto(path)
      await waitForContent(page)
      await expect(page.locator('body')).not.toContainText(/unexpected error|something went wrong/i)
      const main = page.locator('#main-content')
      await expect(main).toBeAttached()
    }
  })
})
