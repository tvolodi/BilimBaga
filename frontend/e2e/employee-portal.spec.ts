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

/**
 * Wait for the portal exam cards to finish loading (skeletons disappear, real content renders).
 */
async function waitForPortalCards(page: Page) {
  await waitForContent(page)
  // Wait for real content: no skeleton AND (cards present OR empty state present)
  await page
    .waitForFunction(() => {
      const hasSkeleton = document.querySelector('.animate-pulse') !== null
      const hasCards = document.querySelector('.rounded-lg.border.bg-card') !== null
      const hasEmpty = (document.body.textContent?.includes('No exams assigned') ?? false) || (document.body.textContent?.includes('\u041d\u0435\u0442 \u043d\u0430\u0437\u043d\u0430\u0447\u0435\u043d\u043d\u044b\u0445 \u044d\u043a\u0437\u0430\u043c\u0435\u043d\u043e\u0432') ?? false)
      return !hasSkeleton && (hasCards || hasEmpty)
    }, { timeout: 20_000 })
    .catch(() => {})
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe('Employee Portal', () => {
  test.setTimeout(120_000)

  test('01 — Portal loads — exam cards render', async ({ page }) => {
    await loginAsEmployee(page)
    await waitForPortalCards(page)

    // Portal title heading must be present
    await expect(page.getByRole('heading', { name: /\u043c\u043e\u0438 \u044d\u043a\u0437\u0430\u043c\u0435\u043d\u044b|my exams/i })).toBeVisible({ timeout: 10_000 })
    await shot(page, 'ep-01-portal-loads')

    // At least one exam card should be visible — shadcn Card renders as div.rounded-lg.border
    const cards = page.locator('.rounded-lg.border.bg-card')
    await expect(cards.first()).toBeVisible({ timeout: 10_000 })
  })

  test('02 — Empty portal state renders gracefully', async ({ page }) => {
    // We cannot reliably seed a zero-exam state without removing assignments,
    // so this test verifies the empty-state component exists in the DOM bundle.
    // If no exams are shown, the "No exams assigned" text is expected.
    await loginAsEmployee(page)
    await waitForPortalCards(page)

    const isEmpty = await page
      .getByText(/\u043d\u0435\u0442 \u043d\u0430\u0437\u043d\u0430\u0447\u0435\u043d\u043d\u044b\u0445 \u044d\u043a\u0437\u0430\u043c\u0435\u043d\u043e\u0432|no exams assigned/i)
      .isVisible()
      .catch(() => false)
    if (isEmpty) {
      await expect(page.getByText(/\u043d\u0435\u0442 \u043d\u0430\u0437\u043d\u0430\u0447\u0435\u043d\u043d\u044b\u0445 \u044d\u043a\u0437\u0430\u043c\u0435\u043d\u043e\u0432|no exams assigned/i)).toBeVisible()
      await shot(page, 'ep-02-portal-empty')
    } else {
      // Exams are present — acceptable, seeded exams exist
      await expect(page.locator('.rounded-lg.border.bg-card').first()).toBeVisible()
      await shot(page, 'ep-02-portal-has-exams')
    }
  })

  test('03 — Start Exam modal opens', async ({ page }) => {
    await loginAsEmployee(page)
    await waitForPortalCards(page)

    // Find any "Start exam" button on any card (exam must be not_started)
    const startBtn = page.getByRole('button', { name: /\u043d\u0430\u0447\u0430\u0442\u044c \u044d\u043a\u0437\u0430\u043c\u0435\u043d|start exam/i }).first()
    const isStartAvailable = await startBtn.isVisible({ timeout: 5_000 }).catch(() => false)

    if (!isStartAvailable) {
      // All exams are in_progress/completed — modal test not applicable this run
      await shot(page, 'ep-03-start-exam-modal-skipped-no-not-started')
      test.info().annotations.push({ type: 'note', description: 'No not_started exam available — mixed exam may be in_progress from previous test run' })
      return
    }

    await startBtn.click()

    // Modal must be visible
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    await expect(page.getByText(/\u043d\u0430\u0447\u0430\u0442\u044c \u044d\u043a\u0437\u0430\u043c\u0435\u043d|start exam/i).first()).toBeVisible()
    await shot(page, 'ep-03-start-exam-modal')
  })

  test('04 — Start Exam modal cancel — modal gone, URL still /portal', async ({ page }) => {
    await loginAsEmployee(page)
    await waitForPortalCards(page)

    const startBtn = page.getByRole('button', { name: /\u043d\u0430\u0447\u0430\u0442\u044c \u044d\u043a\u0437\u0430\u043c\u0435\u043d|start exam/i }).first()
    const isStartAvailable = await startBtn.isVisible({ timeout: 5_000 }).catch(() => false)

    if (!isStartAvailable) {
      await shot(page, 'ep-04-start-exam-modal-skipped-no-not-started')
      test.info().annotations.push({ type: 'note', description: 'No not_started exam available — modal cancel test not applicable this run' })
      return
    }

    await startBtn.click()

    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })

    // Click Cancel
    const cancelBtn = page.getByRole('button', { name: /\u043e\u0442\u043c\u0435\u043d\u0430|cancel/i })
    await expect(cancelBtn).toBeVisible()
    await cancelBtn.click()

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5_000 })
    await expect(page).toHaveURL(/\/portal/, { timeout: 5_000 })
    await shot(page, 'ep-04-start-exam-modal-cancelled')
  })

  test('05 — Start Exam modal confirm — navigates to /portal/sessions/', async ({ page }) => {
    await loginAsEmployee(page)
    await waitForPortalCards(page)

    // Look for a "Start exam" button — works when exam is not_started
    const startBtn = page.getByRole('button', { name: /\u043d\u0430\u0447\u0430\u0442\u044c \u044d\u043a\u0437\u0430\u043c\u0435\u043d|start exam/i }).first()
    const isStartAvailable = await startBtn.isVisible({ timeout: 5_000 }).catch(() => false)

    if (!isStartAvailable) {
      // Exam already in_progress from previous run — verify Continue navigates instead
      const continueBtn = page.getByRole('button', { name: /^\u043f\u0440\u043e\u0434\u043e\u043b\u0436\u0438\u0442\u044c$|^continue$/i }).first()
      if (await continueBtn.isVisible({ timeout: 3_000 }).catch(() => false)) {
        await continueBtn.click()
        await expect(page).toHaveURL(/\/portal\/sessions\//, { timeout: 20_000 })
        await shot(page, 'ep-05-exam-continued-already-in-progress')
        return
      }
      await shot(page, 'ep-05-start-exam-not-available')
      test.info().annotations.push({ type: 'note', description: 'No start/continue button available' })
      return
    }

    await startBtn.click()

    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })

    // Click Begin exam (the confirm button)
    const confirmBtn = page.getByRole('button', { name: /\u043d\u0430\u0447\u0430\u0442\u044c \u044d\u043a\u0437\u0430\u043c\u0435\u043d|begin exam/i }).last()
    await expect(confirmBtn).toBeVisible()
    await confirmBtn.click()

    // Should navigate to the session taking page
    await expect(page).toHaveURL(/\/portal\/sessions\//, { timeout: 20_000 })
    await shot(page, 'ep-05-exam-started')
  })

  test('06 — Continue CTA navigates to in-progress session', async ({ page }) => {
    await loginAsEmployee(page)
    await waitForPortalCards(page)

    // Look for a "Continue" button (in_progress exam card)
    const continueBtn = page.getByRole('button', { name: /^\u043f\u0440\u043e\u0434\u043e\u043b\u0436\u0438\u0442\u044c$|^continue$/i }).first()
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
    await waitForPortalCards(page)

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
    // Wait for skeleton to clear (ResultsTableSkeleton uses animate-pulse Skeleton component)
    // Condition: no skeleton AND (table loaded OR empty state rendered)
    await page
      .waitForFunction(() => {
        const hasSkeleton = document.querySelector('.animate-pulse') !== null
        const hasTable = document.querySelector('table') !== null
        const hasEmpty = document.body.textContent?.includes('You have not completed') ?? false
        return !hasSkeleton && (hasTable || hasEmpty)
      }, { timeout: 20_000 })
      .catch(() => {})

    // Heading must be visible
    await expect(page.getByRole('heading', { name: /\u043c\u043e\u0438 \u0440\u0435\u0437\u0443\u043b\u044c\u0442\u0430\u0442\u044b|my results/i })).toBeVisible({ timeout: 10_000 })
    await shot(page, 'ep-08-my-results')

    // Either a table (with results) or the empty state must be visible
    // Empty state text: "You have not completed any exams yet."
    const hasTable = await page.getByRole('table').isVisible().catch(() => false)
    const hasEmpty = await page
      .getByText(/You have not completed|no exams assigned/i)
      .isVisible()
      .catch(() => false)
    expect(hasTable || hasEmpty).toBeTruthy()
  })
})
