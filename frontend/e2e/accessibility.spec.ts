/**
 * FR-BB63 — Accessibility (WCAG 2.1 AA)
 * E2E tests for skip link and keyboard navigation against the real stack.
 */

import { test, expect } from '@playwright/test'
import type { Page } from '@playwright/test'
import { colorContrastViolations } from './fixtures/axe'
import {
  EMPLOYEE_STORAGE_STATE,
  createPassedEmployeeSession,
  createTestExam,
  createTestQuestion,
  deleteTestExam,
  deleteTestQuestion,
  getSeedData,
  readTenantPrimary,
  startEmployeeSession,
  writeTenantPrimary,
} from './fixtures/seed'

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

// FR-BB320 AC-5 and FR-BB321 AC-11: colour contrast (WCAG AA, color-contrast rule only) on the key screens,
// in the light and in the dark theme. The theme comes from the stored preference, set before the first navigation.
type ThemeName = 'light' | 'dark'
const THEMES: ThemeName[] = ['light', 'dark']

async function openInTheme(page: Page, theme: ThemeName, path: string) {
  await page.addInitScript((value) => localStorage.setItem('bb-theme', value), theme)
  await page.goto(path)
  await waitForContent(page)
  if (theme === 'dark') {
    await expect(page.locator('html')).toHaveClass(/\bdark\b/)
  } else {
    await expect(page.locator('html')).not.toHaveClass(/\bdark\b/)
  }
}

test.describe('Accessibility — colour contrast, both themes (FR-BB320 AC-5, FR-BB321 AC-11)', () => {
  let questionId = ''
  let takingExamId = ''
  let resultExamId = ''
  let takingSessionId = ''
  let resultSessionId = ''

  test.beforeAll(async () => {
    const seed = await getSeedData()
    const question = await createTestQuestion(seed.adminToken, 'Contrast run question', 'single', true)
    questionId = question.id
    const taking = await createTestExam(seed.adminToken, 'Contrast run exam (take)', question.id, {
      assignToUserId: seed.employeeId,
    })
    takingExamId = taking.id
    const result = await createTestExam(seed.adminToken, 'Contrast run exam (result)', question.id, {
      assignToUserId: seed.employeeId,
    })
    resultExamId = result.id
    takingSessionId = await startEmployeeSession(takingExamId)
    resultSessionId = await createPassedEmployeeSession(resultExamId)
  })

  test.afterAll(async () => {
    const seed = await getSeedData()
    for (const examId of [takingExamId, resultExamId]) {
      if (examId) await deleteTestExam(seed.adminToken, examId).catch(() => {})
    }
    if (questionId) await deleteTestQuestion(seed.adminToken, questionId).catch(() => {})
  })

  for (const theme of THEMES) {
    test.describe(`${theme} theme`, () => {
      test.describe('unauthenticated', () => {
        test.use({ storageState: { cookies: [], origins: [] } })

        test('login page has zero color-contrast violations', async ({ page }) => {
          await openInTheme(page, theme, '/login')
          expect(await colorContrastViolations(page)).toEqual([])
        })
      })

      test('admin dashboard has zero color-contrast violations', async ({ page }) => {
        await openInTheme(page, theme, '/admin')
        expect(await colorContrastViolations(page)).toEqual([])
      })

      test('users list has zero color-contrast violations', async ({ page }) => {
        await openInTheme(page, theme, '/admin/users')
        expect(await colorContrastViolations(page)).toEqual([])
      })

      test('question bank list has zero color-contrast violations', async ({ page }) => {
        await openInTheme(page, theme, '/admin/questions')
        expect(await colorContrastViolations(page)).toEqual([])
      })

      test.describe('employee', () => {
        test.use({ storageState: EMPLOYEE_STORAGE_STATE })

        test('employee portal has zero color-contrast violations', async ({ page }) => {
          await openInTheme(page, theme, '/portal')
          expect(await colorContrastViolations(page)).toEqual([])
        })

        test('exam taking has zero color-contrast violations', async ({ page }) => {
          await openInTheme(page, theme, `/portal/sessions/${takingSessionId}`)
          expect(await colorContrastViolations(page)).toEqual([])
        })

        test('result screen has zero color-contrast violations', async ({ page }) => {
          await openInTheme(page, theme, `/portal/sessions/${resultSessionId}/result`)
          expect(await colorContrastViolations(page)).toEqual([])
        })
      })
    })
  }
})

// FR-BB321 AC-11: the dark theme with a non-default tenant primary, so the derived colour is tested in the browser.
// The primary is set through the tenant config API and restored afterwards; the runs are serial because the
// setting is shared by the whole stack.
test.describe('Accessibility — dark theme with a non-default tenant primary (FR-BB321 AC-11)', () => {
  test.describe.configure({ mode: 'serial' })
  const TEST_PRIMARY = '#1B3A6B'
  let originalPrimary = ''

  test.beforeAll(async () => {
    const seed = await getSeedData()
    originalPrimary = await readTenantPrimary()
    await writeTenantPrimary(seed.adminToken, TEST_PRIMARY)
  })

  test.afterAll(async () => {
    const seed = await getSeedData()
    if (originalPrimary) await writeTenantPrimary(seed.adminToken, originalPrimary)
  })

  test('admin dashboard in the dark theme has zero color-contrast violations', async ({ page }) => {
    await openInTheme(page, 'dark', '/admin')
    expect(await colorContrastViolations(page)).toEqual([])
  })

  test.describe('employee portal', () => {
    test.use({ storageState: EMPLOYEE_STORAGE_STATE })

    test('employee portal in the dark theme has zero color-contrast violations', async ({ page }) => {
      await openInTheme(page, 'dark', '/portal')
      expect(await colorContrastViolations(page)).toEqual([])
    })
  })
})
