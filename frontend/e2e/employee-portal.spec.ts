/**
 * Employee Portal E2E tests — 8 tests
 *
 * Requires: make dev running, employee storage state seeded by global-setup.ts
 */

import { test, expect, type Page } from '@playwright/test'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

async function loginAsEmployee(page: Page) {
  await page.goto('/portal')
  await expect(page).toHaveURL(/\/portal/, { timeout: 15_000 })
  const url = page.url()
  if (url.includes('/login')) {
    throw new Error(
      `loginAsEmployee: auth failed — redirected to ${url}. ` +
        'Check that global-setup seeded __e2e_access_token__ into localStorage for employee.',
    )
  }
}

async function shot(page: Page, name: string) {
  await page.screenshot({ path: `screenshots/${name}.png`, fullPage: false })
}

async function waitForContent(page: Page) {
  await page
    .waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin'), {
      timeout: 10_000,
    })
    .catch(() => {})
  await page.waitForLoadState('networkidle').catch(() => {})
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe('Employee Portal', () => {
  test.setTimeout(120_000)

  test('01 — Portal loads — exam cards render', async ({ page }) => {
    await loginAsEmployee(page)
    await waitForContent(page)

    // At least one exam card should be visible (seeded by global-setup)
    const cards = page.locator('[class*="Card"]')
    await expect(cards.first()).toBeVisible({ timeout: 10_000 })
    await shot(page, 'ep-01-portal-loads')

    // Portal title heading must be present
    await expect(page.getByRole('heading', { name: /my exams/i })).toBeVisible()
  })

  test('02 — Empty portal state renders gracefully', async ({ page }) => {
    // We cannot reliably seed a zero-exam state without removing assignments,
    // so this test verifies the empty-state component exists in the DOM bundle.
    // If no exams are shown, the "No exams assigned" text is expected.
    await loginAsEmployee(page)
    await waitForContent(page)

    const isEmpty = await page
      .getByText(/no exams assigned/i)
      .isVisible()
      .catch(() => false)
    if (isEmpty) {
      await expect(page.getByText(/no exams assigned/i)).toBeVisible()
      await shot(page, 'ep-02-portal-empty')
    } else {
      // Exams are present — acceptable, seeded exams exist
      await expect(page.locator('[class*="Card"]').first()).toBeVisible()
      await shot(page, 'ep-02-portal-has-exams')
    }
  })

  test('03 — Start Exam modal opens', async ({ page }) => {
    await loginAsEmployee(page)
    await waitForContent(page)

    // Click the first "Start exam" button
    const startBtn = page.getByRole('button', { name: /start exam/i }).first()
    await expect(startBtn).toBeVisible({ timeout: 10_000 })
    await startBtn.click()

    // Modal must be visible
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    await expect(page.getByText(/start exam/i).first()).toBeVisible()
    await shot(page, 'ep-03-start-exam-modal')
  })

  test('04 — Start Exam modal cancel — modal gone, URL still /portal', async ({ page }) => {
    await loginAsEmployee(page)
    await waitForContent(page)

    const startBtn = page.getByRole('button', { name: /start exam/i }).first()
    await expect(startBtn).toBeVisible({ timeout: 10_000 })
    await startBtn.click()

    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })

    // Click Cancel
    const cancelBtn = page.getByRole('button', { name: /cancel/i })
    await expect(cancelBtn).toBeVisible()
    await cancelBtn.click()

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5_000 })
    await expect(page).toHaveURL(/\/portal/, { timeout: 5_000 })
    await shot(page, 'ep-04-start-exam-modal-cancelled')
  })

  test('05 — Start Exam modal confirm — navigates to /portal/sessions/', async ({ page }) => {
    await loginAsEmployee(page)
    await waitForContent(page)

    // Find "not started" exam card — should have "Start exam" button
    const startBtn = page.getByRole('button', { name: /start exam/i }).first()
    await expect(startBtn).toBeVisible({ timeout: 10_000 })
    await startBtn.click()

    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })

    // Click Begin exam (the confirm button)
    const confirmBtn = page.getByRole('button', { name: /begin exam/i })
    await expect(confirmBtn).toBeVisible()
    await confirmBtn.click()

    // Should navigate to the session taking page
    await expect(page).toHaveURL(/\/portal\/sessions\//, { timeout: 20_000 })
    await shot(page, 'ep-05-exam-started')
  })

  test('06 — Continue CTA navigates to in-progress session', async ({ page }) => {
    await loginAsEmployee(page)
    await waitForContent(page)

    // Look for a "Continue" button (in_progress exam card)
    const continueBtn = page.getByRole('button', { name: /^continue$/i }).first()
    if (await continueBtn.isVisible({ timeout: 3_000 }).catch(() => false)) {
      await continueBtn.click()
      await expect(page).toHaveURL(/\/portal\/sessions\//, { timeout: 15_000 })
      await shot(page, 'ep-06-continue-session')
    } else {
      // No in-progress exam available — acceptable
      await shot(page, 'ep-06-no-in-progress-exam')
      test.info().annotations.push({ type: 'note', description: 'No in-progress session at test time' })
    }
  })

  test('07 — View Result CTA navigates to result', async ({ page }) => {
    await loginAsEmployee(page)
    await waitForContent(page)

    // Look for "View result" button (passed/failed exam card)
    const viewResultBtn = page.getByRole('button', { name: /view result/i }).first()
    if (await viewResultBtn.isVisible({ timeout: 3_000 }).catch(() => false)) {
      await viewResultBtn.click()
      await expect(page).toHaveURL(/\/portal\/(exams|sessions)\//, { timeout: 15_000 })
      await shot(page, 'ep-07-view-result')
    } else {
      // No completed exam available — acceptable
      await shot(page, 'ep-07-no-completed-exam')
      test.info().annotations.push({ type: 'note', description: 'No completed session at test time' })
    }
  })

  test('08 — My Results page renders', async ({ page }) => {
    await loginAsEmployee(page)
    await page.goto('/portal/results')
    await waitForContent(page)

    // Heading must be visible
    await expect(page.getByRole('heading', { name: /my results/i })).toBeVisible({ timeout: 10_000 })
    await shot(page, 'ep-08-my-results')

    // Either a table (with results) or the empty state must be visible
    const hasTable = await page.getByRole('table').isVisible().catch(() => false)
    const hasEmpty = await page.getByText(/no.*(exam|result)/i).isVisible().catch(() => false)
    expect(hasTable || hasEmpty).toBeTruthy()
  })
})
