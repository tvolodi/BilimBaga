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

// FR-BB321 AC-6 and AC-13: the admin top bar, with its toggle and locale switcher, fits 375 px.
test.describe('Admin top bar at 375 px (FR-BB321 AC-6, AC-13)', () => {
  test.use({ viewport: { width: 375, height: 812 } })

  for (const locale of ['kk', 'ru'] as const) {
    test(`fits without horizontal scroll in ${locale}`, async ({ page }) => {
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

      // Restore the admin's locale so later specs see the seeded default.
      await localeSelect(page).selectOption('en')
    })
  }
})
