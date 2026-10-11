/**
 * FR-BB320 AC-13: page baselines for login, the admin dashboard, the users list, the question bank list, the employee
 * portal, exam taking and the result screen, at 1280 and 375 px in en and kk, light theme only (28 baselines). The
 * fixtures come from the setup project (visual-fixtures.setup.ts). Dynamic nodes (dates, ids, durations, counts from the
 * clock) carry data-visual-mask and are passed to toHaveScreenshot as mask. Runs only through scripts/visual/run.sh.
 */

import { test, expect, type Browser, type Page } from '@playwright/test'
import {
  ADMIN_STORAGE_STATE,
  EMPLOYEE_STORAGE_STATE,
  employeeFetch,
  loadAppFonts,
  readVisualFixtures,
} from './fixtures/visual'

// The visual suite runs only through scripts/visual/run.sh, which sets VISUAL_MODE (FR-BB320 AC-11).
test.skip(!process.env.VISUAL_MODE, 'visual suite: run through scripts/visual/run.sh')

const WIDTHS = [1280, 375] as const
const LOCALES = ['en', 'kk'] as const

/** The employee's saved locale overrides the browser's on portal pages, so it is set per test (FR-BB116). */
async function setEmployeeLocale(locale: string | null) {
  const res = await employeeFetch('/api/v1/users/me', 'PATCH', { preferred_locale: locale })
  if (!res.ok) throw new Error(`PATCH /users/me preferred_locale failed: ${res.status}`)
}

async function waitForContent(page: Page) {
  // Skeletons (animate-pulse) and spinners are the loading states; the data is on screen once they are gone.
  await page
    .waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin, .animate-pulse'), null, { timeout: 15_000 })
    .catch(() => {})
  await page.waitForLoadState('networkidle').catch(() => {})
  await loadAppFonts(page)
}

type Auth = 'none' | 'admin' | 'employee'

interface VisualPage {
  name: string
  auth: Auth
  path: () => string
}

const PAGES: VisualPage[] = [
  { name: 'login', auth: 'none', path: () => '/login' },
  { name: 'admin-dashboard', auth: 'admin', path: () => '/admin/dashboard' },
  { name: 'users-list', auth: 'admin', path: () => '/admin/users' },
  { name: 'question-bank', auth: 'admin', path: () => '/admin/questions' },
  { name: 'employee-portal', auth: 'employee', path: () => '/portal' },
  { name: 'exam-taking', auth: 'employee', path: () => `/portal/sessions/${readVisualFixtures().takingSessionId}` },
  { name: 'result', auth: 'employee', path: () => `/portal/sessions/${readVisualFixtures().resultSessionId}/result` },
]

const STORAGE: Record<Auth, string | undefined> = {
  none: undefined,
  admin: ADMIN_STORAGE_STATE,
  employee: EMPLOYEE_STORAGE_STATE,
}

test.describe('visual pages (FR-BB320 AC-13)', () => {
  test.afterAll(async () => {
    await setEmployeeLocale(null).catch(() => {})
  })

  for (const visualPage of PAGES) {
    for (const locale of LOCALES) {
      for (const width of WIDTHS) {
        test(`${visualPage.name}, ${width} px, ${locale}`, async ({ browser }: { browser: Browser }) => {
          if (visualPage.auth === 'employee') await setEmployeeLocale(locale)
          const context = await browser.newContext({
            storageState: STORAGE[visualPage.auth],
            viewport: { width, height: 900 },
          })
          try {
            await context.addInitScript((lang: string) => localStorage.setItem('i18n-lang', lang), locale)
            const page = await context.newPage()
            await page.goto(visualPage.path())
            await waitForContent(page)
            await expect(page).toHaveScreenshot(`${visualPage.name}-${width}-${locale}.png`, {
              fullPage: true,
              mask: [page.locator('[data-visual-mask]')],
            })
          } finally {
            await context.close()
          }
        })
      }
    }
  }
})
