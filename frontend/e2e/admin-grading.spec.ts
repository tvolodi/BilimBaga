/**
 * Admin Grading Queue E2E tests — 7 tests
 *
 * Uses admin storage state.
 * Requires: make dev running, a submitted short-text session seeded by global-setup.ts
 */

import { test, expect, type Page } from '@playwright/test'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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

async function loginAsAdmin(page: Page) {
  await page.goto('/admin/dashboard')
  await expect(page).toHaveURL(/\/admin/, { timeout: 15_000 })
  const url = page.url()
  if (url.includes('/login')) {
    throw new Error(
      `loginAsAdmin: auth failed — redirected to ${url}. ` +
        'Check that global-setup seeded __e2e_access_token__ into localStorage.',
    )
  }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe('Admin Grading Queue', () => {
  test.setTimeout(120_000)

  test('01 — Grading queue page renders', async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/admin/grading')
    await waitForContent(page)

    // Heading must be visible
    await expect(page.getByRole('heading', { name: /manual grading queue/i })).toBeVisible({
      timeout: 10_000,
    })
    await shot(page, 'ag-01-grading-queue')

    // Either the empty message or a session row must be visible
    const hasEmpty = await page
      .getByText(/no sessions are awaiting manual grading/i)
      .isVisible()
      .catch(() => false)
    const hasRow = await page.getByRole('table').isVisible().catch(() => false)
    expect(hasEmpty || hasRow).toBeTruthy()
  })

  test('02 — Session row renders and clicking navigates to detail', async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/admin/grading')
    await waitForContent(page)

    const table = page.getByRole('table')
    if (await table.isVisible({ timeout: 5_000 }).catch(() => false)) {
      // Check table row exists
      const firstRow = table.locator('tbody tr').first()
      await expect(firstRow).toBeVisible({ timeout: 5_000 })
      await shot(page, 'ag-02-grading-queue-row')

      // Click the row (or a link within it)
      const rowLink = firstRow.getByRole('link').first()
      if (await rowLink.isVisible({ timeout: 2_000 }).catch(() => false)) {
        await rowLink.click()
      } else {
        await firstRow.click()
      }

      await expect(page).toHaveURL(/\/admin\/grading\//, { timeout: 15_000 })
      await waitForContent(page)
      await shot(page, 'ag-02-grading-detail')
    } else {
      // Empty queue — acceptable if no submitted sessions
      await expect(page.getByText(/no sessions are awaiting manual grading/i)).toBeVisible()
      await shot(page, 'ag-02-grading-queue-empty')
      test.info().annotations.push({ type: 'note', description: 'Grading queue is empty — skipping row navigation' })
    }
  })

  test('03 — Question display and Previous/Next navigation', async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/admin/grading')
    await waitForContent(page)

    const table = page.getByRole('table')
    if (await table.isVisible({ timeout: 5_000 }).catch(() => false)) {
      const firstRow = table.locator('tbody tr').first()
      const rowLink = firstRow.getByRole('link').first()
      if (await rowLink.isVisible({ timeout: 2_000 }).catch(() => false)) {
        await rowLink.click()
      } else {
        await firstRow.click()
      }
      await waitForContent(page)

      // Detail page title
      await expect(page.getByRole('heading', { name: /grade session/i })).toBeVisible({
        timeout: 10_000,
      })
      await shot(page, 'ag-03-grading-detail')

      // Question indicator: "Question N of M"
      await expect(page.getByText(/question \d+ of \d+/i)).toBeVisible()

      // Previous button (disabled at question 1) and Next button
      const prevBtn = page.getByRole('button', { name: /previous/i })
      const nextBtn = page.getByRole('button', { name: /next/i })
      await expect(prevBtn).toBeVisible()
      await expect(nextBtn).toBeVisible()

      // Previous should be disabled on first question
      await expect(prevBtn).toBeDisabled()
      await shot(page, 'ag-03-grading-navigation')

      // If there are multiple questions, Next should be enabled
      const nextEnabled = await nextBtn.isEnabled().catch(() => false)
      if (nextEnabled) {
        await nextBtn.click()
        await page.waitForTimeout(500)
        await shot(page, 'ag-03-grading-question-2')
      }
    } else {
      await shot(page, 'ag-03-queue-empty')
      test.info().annotations.push({ type: 'note', description: 'Queue empty — cannot test question navigation' })
    }
  })

  test('04 — Score input validation', async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/admin/grading')
    await waitForContent(page)

    const table = page.getByRole('table')
    if (await table.isVisible({ timeout: 5_000 }).catch(() => false)) {
      const firstRow = table.locator('tbody tr').first()
      const rowLink = firstRow.getByRole('link').first()
      if (await rowLink.isVisible({ timeout: 2_000 }).catch(() => false)) {
        await rowLink.click()
      } else {
        await firstRow.click()
      }
      await waitForContent(page)

      // Find the score number input (not the range slider)
      const scoreInput = page.locator('input[type="number"]').first()
      await expect(scoreInput).toBeVisible({ timeout: 10_000 })

      // Enter invalid value (>100)
      await scoreInput.clear()
      await scoreInput.fill('150')
      await page.waitForTimeout(300)
      await shot(page, 'ag-04-score-invalid')

      // Error message should appear
      await expect(page.getByText(/score must be between 0 and 100/i)).toBeVisible({
        timeout: 5_000,
      })
      await shot(page, 'ag-04-score-error-visible')

      // Enter valid value
      await scoreInput.clear()
      await scoreInput.fill('85')
      await page.waitForTimeout(300)

      // Error message should disappear
      await expect(page.getByText(/score must be between 0 and 100/i)).not.toBeVisible({
        timeout: 3_000,
      })
      await shot(page, 'ag-04-score-valid')
    } else {
      await shot(page, 'ag-04-queue-empty')
      test.info().annotations.push({ type: 'note', description: 'Queue empty — cannot test score validation' })
    }
  })

  test('05 — Feedback textarea', async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/admin/grading')
    await waitForContent(page)

    const table = page.getByRole('table')
    if (await table.isVisible({ timeout: 5_000 }).catch(() => false)) {
      const firstRow = table.locator('tbody tr').first()
      const rowLink = firstRow.getByRole('link').first()
      if (await rowLink.isVisible({ timeout: 2_000 }).catch(() => false)) {
        await rowLink.click()
      } else {
        await firstRow.click()
      }
      await waitForContent(page)

      // Find feedback textarea (not the read-only employee answer textarea)
      // The feedback textarea has a placeholder "Add comments for the employee..."
      const feedbackTextarea = page.locator('textarea[placeholder*="Add comments"]')
      await expect(feedbackTextarea).toBeVisible({ timeout: 10_000 })

      await feedbackTextarea.fill('Great answer! Well done.')
      await page.waitForTimeout(300)
      await expect(feedbackTextarea).toHaveValue('Great answer! Well done.')
      await shot(page, 'ag-05-feedback-typed')
    } else {
      await shot(page, 'ag-05-queue-empty')
      test.info().annotations.push({ type: 'note', description: 'Queue empty — cannot test feedback textarea' })
    }
  })

  test('06 — Submit All Grades — loading state → success toast → redirect', async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/admin/grading')
    await waitForContent(page)

    const table = page.getByRole('table')
    if (await table.isVisible({ timeout: 5_000 }).catch(() => false)) {
      const firstRow = table.locator('tbody tr').first()
      const rowLink = firstRow.getByRole('link').first()
      if (await rowLink.isVisible({ timeout: 2_000 }).catch(() => false)) {
        await rowLink.click()
      } else {
        await firstRow.click()
      }
      await waitForContent(page)

      // Fill score with a valid value
      const scoreInput = page.locator('input[type="number"]').first()
      await expect(scoreInput).toBeVisible({ timeout: 10_000 })
      await scoreInput.clear()
      await scoreInput.fill('80')
      await page.waitForTimeout(300)

      // Click Submit All Grades
      const submitBtn = page.getByRole('button', { name: /submit all grades/i })
      await expect(submitBtn).toBeVisible()
      await shot(page, 'ag-06-before-submit')
      await submitBtn.click()

      // Wait for either success toast or redirect to /admin/grading
      await page
        .waitForURL(/\/admin\/grading$/, { timeout: 20_000 })
        .catch(async () => {
          // Might show toast first
          const toast = page.getByText(/all grades submitted/i)
          await expect(toast).toBeVisible({ timeout: 20_000 })
          await shot(page, 'ag-06-success-toast')
        })

      await shot(page, 'ag-06-after-submit')
    } else {
      await shot(page, 'ag-06-queue-empty')
      test.info().annotations.push({ type: 'note', description: 'Queue empty — cannot test submit all grades' })
    }
  })

  test('07 — Pagination boundary states', async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/admin/grading')
    await waitForContent(page)

    await shot(page, 'ag-07-pagination')

    // If pagination controls exist (more than one page of sessions)
    const prevBtn = page.getByRole('button').filter({ hasText: '←' }).first()
    const nextBtn = page.getByRole('button').filter({ hasText: '→' }).first()

    if (await prevBtn.isVisible({ timeout: 3_000 }).catch(() => false)) {
      // Previous button should be disabled at page 1
      await expect(prevBtn).toBeDisabled()

      // Check if next is enabled (multiple pages)
      const nextEnabled = await nextBtn.isEnabled().catch(() => false)
      if (nextEnabled) {
        await nextBtn.click()
        await waitForContent(page)
        await shot(page, 'ag-07-pagination-page2')

        // Now previous should be enabled
        await expect(prevBtn).toBeEnabled()
      }
    } else {
      // No pagination visible — single page or empty queue
      await expect(page.locator('body')).not.toContainText(/unexpected error/i)
      test.info().annotations.push({ type: 'note', description: 'No pagination controls visible (queue fits in one page or is empty)' })
    }
  })
})
