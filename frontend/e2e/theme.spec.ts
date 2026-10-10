/**
 * FR-BB321 AC-12 — theme switch end to end: the choice persists, the stored theme is applied before
 * React mounts, the system choice follows the OS, the exam screen has no toggle, and the admin top bar
 * fits 375 px in kk and ru (AC-6, AC-13).
 */

import { test, expect, type Page } from '@playwright/test'
import {
  EMPLOYEE_STORAGE_STATE,
  createTestExam,
  createTestQuestion,
  deleteTestExam,
  deleteTestQuestion,
  getSeedData,
  startEmployeeSession,
} from './fixtures/seed'

async function waitForContent(page: Page) {
  await page.waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin'), { timeout: 10_000 }).catch(() => {})
  await page.waitForLoadState('networkidle').catch(() => {})
}

// Native <select> rendered by LocaleSwitcher (options kk / ru / en).
function localeSelect(page: Page) {
  return page.locator('select').filter({ has: page.locator('option[value="kk"]') })
}

test.describe('Theme switch — admin top bar (FR-BB321 AC-5, AC-12)', () => {
  test('choosing Dark persists across a reload and sets color-scheme', async ({ page }) => {
    await page.goto('/admin')
    await waitForContent(page)

    await page.getByRole('radio', { name: 'Dark', exact: true }).click()
    await expect(page.locator('html')).toHaveClass(/\bdark\b/)
    await expect.poll(() => page.evaluate(() => document.documentElement.style.colorScheme)).toBe('dark')

    await page.reload()
    await waitForContent(page)
    await expect(page.locator('html')).toHaveClass(/\bdark\b/)
    await expect.poll(() => page.evaluate(() => document.documentElement.style.colorScheme)).toBe('dark')
  })

  test('the stored dark theme is on the page before React mounts (no flash)', async ({ page }) => {
    await page.goto('/admin')
    await page.evaluate(() => localStorage.setItem('bb-theme', 'dark'))

    // Records the theme at the first insertion into #root. The init script runs before any page script.
    await page.addInitScript(() => {
      const probe: { darkAtFirstRootChild: boolean | null } = { darkAtFirstRootChild: null }
      ;(window as unknown as { __themeProbe: typeof probe }).__themeProbe = probe
      const observer = new MutationObserver(() => {
        const root = document.getElementById('root')
        if (root && root.childNodes.length > 0 && probe.darkAtFirstRootChild === null) {
          probe.darkAtFirstRootChild = document.documentElement.classList.contains('dark')
          observer.disconnect()
        }
      })
      observer.observe(document, { childList: true, subtree: true })
    })

    await page.reload()
    await waitForContent(page)
    const darkAtFirstChild = await page.evaluate(
      () => (window as unknown as { __themeProbe?: { darkAtFirstRootChild: boolean | null } }).__themeProbe?.darkAtFirstRootChild ?? null,
    )
    expect(darkAtFirstChild).toBe(true)
  })

  test('System follows the OS, including a change while the page is open', async ({ page }) => {
    await page.emulateMedia({ colorScheme: 'dark' })
    await page.goto('/admin')
    await waitForContent(page)

    await page.getByRole('radio', { name: 'System', exact: true }).click()
    await expect(page.locator('html')).toHaveClass(/\bdark\b/)

    await page.emulateMedia({ colorScheme: 'light' })
    await expect(page.locator('html')).not.toHaveClass(/\bdark\b/)
  })
})

test.describe('Theme switch — exam screen (FR-BB321 D-5, AC-12)', () => {
  test.use({ storageState: EMPLOYEE_STORAGE_STATE })

  let questionId = ''
  let examId = ''
  let sessionId = ''

  test.beforeAll(async () => {
    const seed = await getSeedData()
    const question = await createTestQuestion(seed.adminToken, 'Theme run question', 'single', true)
    questionId = question.id
    const exam = await createTestExam(seed.adminToken, 'Theme run exam', question.id, {
      assignToUserId: seed.employeeId,
    })
    examId = exam.id
    sessionId = await startEmployeeSession(examId)
  })

  test.afterAll(async () => {
    const seed = await getSeedData()
    if (examId) await deleteTestExam(seed.adminToken, examId).catch(() => {})
    if (questionId) await deleteTestQuestion(seed.adminToken, questionId).catch(() => {})
  })

  test('the exam screen has no theme toggle and applies the stored theme', async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem('bb-theme', 'dark'))
    await page.goto(`/portal/sessions/${sessionId}`)
    await waitForContent(page)

    await expect(page.locator('html')).toHaveClass(/\bdark\b/)
    await expect(page.getByRole('radiogroup', { name: 'Theme' })).toHaveCount(0)
  })
})

// FR-BB321 AC-6, AC-13 and #472: at 375 px the admin top bar fits, and the sidebar starts as a rail.
test.describe('Admin layout at 375 px (FR-BB321 AC-6, AC-13, #472)', () => {
  test.use({ viewport: { width: 375, height: 812 } })

  // Restores the admin's locale to the seeded default even when a test fails, so later specs are not affected.
  test.afterEach(async ({ page }) => {
    await page.goto('/admin')
    await waitForContent(page)
    await localeSelect(page).selectOption('en')
  })

  for (const locale of ['kk', 'ru'] as const) {
    test(`top bar fits without horizontal scroll in ${locale}`, async ({ page }) => {
      await page.goto('/admin')
      await waitForContent(page)
      await localeSelect(page).selectOption(locale)
      await waitForContent(page)

      const overflow = await page.evaluate(() => {
        const header = document.querySelector('header')
        return {
          page: document.documentElement.scrollWidth - window.innerWidth,
          header: header ? header.scrollWidth - header.clientWidth : -1,
        }
      })
      expect(overflow.page).toBeLessThanOrEqual(0)
      expect(overflow.header).toBeLessThanOrEqual(0)
      await expect(page.locator('header').getByRole('radiogroup', { name: 'Theme' })).toBeInViewport()
    })
  }

  test('the sidebar starts as a rail, opens over the page, and closes after a link is followed', async ({ page }) => {
    await page.goto('/admin')
    await waitForContent(page)
    const toggle = page.getByRole('button', { name: /sidebar/i })
    await expect(toggle).toHaveAttribute('aria-expanded', 'false')

    await toggle.click()
    await expect(toggle).toHaveAttribute('aria-expanded', 'true')
    const box = await toggle.boundingBox()
    expect(box?.width ?? 0).toBeGreaterThanOrEqual(44)
    expect(box?.height ?? 0).toBeGreaterThanOrEqual(44)

    await page.getByRole('link', { name: 'Users', exact: true }).click()
    await expect(page).toHaveURL(new RegExp('/admin/users'))
    await expect(page.getByRole('button', { name: /sidebar/i })).toHaveAttribute('aria-expanded', 'false')
  })
})
