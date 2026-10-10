/**
 * FR-BB63 — Accessibility (WCAG 2.1 AA)
 * E2E tests for skip link and keyboard navigation against the real stack.
 */

import { test, expect } from '@playwright/test'
import type { Page } from '@playwright/test'
import { colorContrastViolations } from './fixtures/axe'
import { EMPLOYEE_STORAGE_STATE } from './fixtures/seed'

async function waitForContent(page: import('@playwright/test').Page) {
  await page.waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin'), { timeout: 10_000 }).catch(() => {})
  await page.waitForLoadState('networkidle').catch(() => {})
}

/** Light theme is the default; remove the class explicitly so the run cannot inherit a dark preference. */
async function useLightTheme(page: Page) {
  await page.evaluate(() => document.documentElement.classList.remove('dark'))
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

// FR-BB320 AC-5: colour contrast (WCAG AA, color-contrast rule only) on the key screens.
test.describe('Accessibility — colour contrast, light theme (FR-BB320 AC-5)', () => {
  test.describe('unauthenticated', () => {
    test.use({ storageState: { cookies: [], origins: [] } })

    test('login page has zero color-contrast violations', async ({ page }) => {
      await page.goto('/login')
      await waitForContent(page)
      await useLightTheme(page)
      expect(await colorContrastViolations(page)).toEqual([])
    })
  })

  test('admin dashboard has zero color-contrast violations', async ({ page }) => {
    await page.goto('/admin')
    await waitForContent(page)
    await useLightTheme(page)
    expect(await colorContrastViolations(page)).toEqual([])
  })

  test.describe('employee portal', () => {
    test.use({ storageState: EMPLOYEE_STORAGE_STATE })

    test('employee portal has zero color-contrast violations', async ({ page }) => {
      await page.goto('/portal')
      await waitForContent(page)
      await useLightTheme(page)
      expect(await colorContrastViolations(page)).toEqual([])
    })
  })
})

test.describe('Accessibility — colour contrast, dark theme (FR-BB320 AC-5)', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test('login page in the dark theme has zero color-contrast violations', async ({ page }) => {
    await page.goto('/login')
    await waitForContent(page)
    await page.evaluate(() => document.documentElement.classList.add('dark'))
    expect(await colorContrastViolations(page)).toEqual([])
  })
})
