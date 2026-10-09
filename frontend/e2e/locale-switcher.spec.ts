/**
 * Portal locale switcher E2E (FR-BB316, issue #16).
 *
 * Requires: make dev running, employee storage state from global-setup.ts (seeded with
 * localStorage i18n-lang = 'ru'). Runs in the chromium-live-employee project.
 * The login-screen selector test uses a fresh unauthenticated context.
 */
import { test, expect, type Page } from '@playwright/test'
import { EMPLOYEE_STORAGE_STATE } from './fixtures/seed'

// Native <select> rendered by LocaleSwitcher (options kk / ru / en).
function localeSelect(page: Page) {
  return page.locator('select').filter({ has: page.locator('option[value="kk"]') })
}

const TABS = {
  ru: { exams: 'Мои экзамены', results: 'Мои результаты' },
  en: { exams: 'My Exams', results: 'My Results' },
  kk: { exams: 'Менің емтихандарым', results: 'Менің нәтижелерім' },
}

function portalNav(page: Page) {
  return page.getByRole('navigation', { name: /portal navigation/i })
}

test.describe('Portal locale switcher (FR-BB316)', () => {
  test.use({ storageState: EMPLOYEE_STORAGE_STATE })

  test('AC1: switcher is visible in the portal header and offers kk/ru/en', async ({ page }) => {
    await page.goto('/portal')
    const select = localeSelect(page)
    await expect(select).toBeVisible({ timeout: 20_000 })
    await expect(select.locator('option')).toHaveText(['Қазақша', 'Русский', 'English'])
    // Seeded storage state starts the portal in Russian.
    await expect(select).toHaveValue('ru')
  })

  test('AC2: selecting a locale changes visible strings without a page reload', async ({ page }) => {
    await page.goto('/portal')
    const nav = portalNav(page)
    await expect(nav.getByRole('link', { name: TABS.ru.exams })).toBeVisible({ timeout: 20_000 })

    // Marker that disappears on a full reload.
    await page.evaluate(() => {
      ;(window as unknown as { __noReload: boolean }).__noReload = true
    })

    await localeSelect(page).selectOption('en')
    await expect(nav.getByRole('link', { name: TABS.en.exams })).toBeVisible()
    await expect(nav.getByRole('link', { name: TABS.en.results })).toBeVisible()
    await expect(page.locator('html')).toHaveAttribute('lang', 'en')

    await localeSelect(page).selectOption('kk')
    await expect(nav.getByRole('link', { name: TABS.kk.exams })).toBeVisible()
    await expect(page.locator('html')).toHaveAttribute('lang', 'kk')

    expect(await page.evaluate(() => (window as unknown as { __noReload?: boolean }).__noReload)).toBe(true)
  })

  test('AC3: locale is retained when navigating between portal tabs', async ({ page }) => {
    await page.goto('/portal')
    await localeSelect(page).selectOption('en')
    const nav = portalNav(page)
    await nav.getByRole('link', { name: TABS.en.results }).click()
    await expect(page).toHaveURL(/\/portal\/results$/)
    await expect(localeSelect(page)).toHaveValue('en')
    await expect(nav.getByRole('link', { name: TABS.en.exams })).toBeVisible()
    await nav.getByRole('link', { name: TABS.en.exams }).click()
    await expect(page).toHaveURL(/\/portal$/)
    await expect(localeSelect(page)).toHaveValue('en')
  })

  test('AC4: selection is stored in localStorage and survives a reload', async ({ page }) => {
    await page.goto('/portal')
    await localeSelect(page).selectOption('en')
    await expect.poll(() => page.evaluate(() => localStorage.getItem('i18n-lang'))).toBe('en')

    await page.reload()
    await expect(localeSelect(page)).toHaveValue('en', { timeout: 20_000 })
    await expect(portalNav(page).getByRole('link', { name: TABS.en.exams })).toBeVisible()
  })

  test('AC6: switcher does not cause horizontal overflow at 375px width', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 800 })
    await page.goto('/portal')
    await expect(localeSelect(page)).toBeVisible({ timeout: 20_000 })
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    )
    expect(overflow).toBeLessThanOrEqual(0)
  })
})

test.describe('Login screen language selector (FR-BB316 AC5: own switcher)', () => {
  test('login screen has its own language selector and no portal switcher', async ({ browser }) => {
    const ctx = await browser.newContext()
    const page = await ctx.newPage()
    try {
      await page.goto('/login')
      const select = page.getByRole('combobox', { name: /язык|language|тіл/i })
      await expect(select).toBeVisible({ timeout: 20_000 })
      await expect(page.getByRole('navigation', { name: /portal navigation/i })).toHaveCount(0)

      await select.selectOption('ru')
      await expect(page.getByRole('button', { name: /войти/i })).toBeVisible()
      await select.selectOption('en')
      await expect(page.getByRole('button', { name: /sign in/i })).toBeVisible()
      expect(await page.evaluate(() => localStorage.getItem('i18n-lang'))).toBe('en')
    } finally {
      await ctx.close()
    }
  })
})
