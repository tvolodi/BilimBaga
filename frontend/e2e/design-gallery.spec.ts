/**
 * FR-BB320 AC-12: the design gallery as one full-page screenshot per width and locale, in the light theme at 375, 768 and
 * 1280 px in kk, ru and en (9 baselines), plus one with every section in the .dark wrapper at 1280 px in en (1 baseline).
 * The gallery is the whole page, not one screenshot per component. Runs only through scripts/visual/run.sh.
 */

import { test, expect, type Page } from '@playwright/test'
import { loadAppFonts } from './fixtures/visual'

// The visual suite runs only through scripts/visual/run.sh, which sets VISUAL_MODE (FR-BB320 AC-11).
test.skip(!process.env.VISUAL_MODE, 'visual suite: run through scripts/visual/run.sh')

const WIDTHS = [375, 768, 1280] as const
const LOCALES = ['kk', 'ru', 'en'] as const

async function openGallery(page: Page, width: number, locale: string) {
  await page.setViewportSize({ width, height: 900 })
  await page.addInitScript((lang: string) => localStorage.setItem('i18n-lang', lang), locale)
  await page.goto('/__design')
  await page.waitForLoadState('networkidle')
  await loadAppFonts(page)
}

for (const locale of LOCALES) {
  for (const width of WIDTHS) {
    test(`gallery light, ${width} px, ${locale}`, async ({ page }) => {
      await openGallery(page, width, locale)
      await expect(page).toHaveScreenshot(`gallery-light-${width}-${locale}.png`, { fullPage: true })
    })
  }
}

test('gallery with every section dark, 1280 px, en', async ({ page }) => {
  await openGallery(page, 1280, 'en')
  // Each section has its own Theme group; its Dark button puts that section's states in the .dark wrapper.
  const darkButtons = page.locator('[role="group"][aria-label="Theme"] button', { hasText: /^Dark$/ })
  const sections = await darkButtons.count()
  expect(sections).toBeGreaterThan(0)
  for (let i = 0; i < sections; i++) {
    await darkButtons.nth(i).click()
  }
  await expect(page).toHaveScreenshot('gallery-dark-1280-en.png', { fullPage: true })
})
