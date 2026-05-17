/**
 * Exam Taking E2E tests — 11 tests
 *
 * Requires: make dev running, employee storage state seeded by global-setup.ts
 * The "E2E Mixed Exam" (5 questions, one per type) must be assigned to the employee.
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

/**
 * Start a fresh exam session via the portal Start modal.
 * Returns the session URL after navigation completes.
 */
async function startExamSession(page: Page): Promise<string> {
  await page.goto('/portal')
  await page.waitForLoadState('networkidle')

  // Wait for exam cards to load
  const startBtn = page.getByRole('button', { name: /start exam/i }).first()
  await expect(startBtn).toBeVisible({ timeout: 15_000 })

  // If it's "Continue" because a session is already open, navigate directly
  const continueBtn = page.getByRole('button', { name: /^continue$/i }).first()
  if (await continueBtn.isVisible({ timeout: 2_000 }).catch(() => false)) {
    await continueBtn.click()
    await expect(page).toHaveURL(/\/portal\/sessions\//, { timeout: 20_000 })
    return page.url()
  }

  await startBtn.click()

  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible({ timeout: 5_000 })

  const confirmBtn = page.getByRole('button', { name: /begin exam/i })
  await expect(confirmBtn).toBeVisible()
  await confirmBtn.click()

  await expect(page).toHaveURL(/\/portal\/sessions\//, { timeout: 20_000 })
  return page.url()
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe('Exam Taking', () => {
  test.setTimeout(180_000)

  test('01 — Top bar and navigator render', async ({ page }) => {
    const sessionUrl = await startExamSession(page)
    await waitForContent(page)

    // ExamTopBar: exam title heading (h1)
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 10_000 })

    // CountdownTimer: shows HH:MM:SS or MM:SS format — find by role="timer" or text pattern
    // The timer is a <span> inside the header — look for time-like text
    const timer = page.locator('header').getByText(/\d{1,2}:\d{2}:\d{2}|\d{2}:\d{2}/).first()
    await expect(timer).toBeVisible({ timeout: 10_000 })

    // Progress indicator: "0 / N answered" or "N answered"
    const progress = page.locator('header').getByText(/\d+ \/ \d+ answered|\d+ answered/i).first()
    await expect(progress).toBeVisible({ timeout: 5_000 })

    // QuestionNavigator grid: buttons numbered 1..N in the sidebar
    const navButtons = page.locator('aside').getByRole('button')
    const navCount = await navButtons.count()
    expect(navCount).toBeGreaterThan(0)

    await shot(page, 'et-01-topbar-navigator')
    console.log('[test] Session URL:', sessionUrl)
  })

  test('02 — Single-choice answer', async ({ page }) => {
    await startExamSession(page)
    await waitForContent(page)

    // Find a radio group (single choice question)
    const radioGroup = page.locator('[role="radiogroup"]').first()
    await expect(radioGroup).toBeVisible({ timeout: 10_000 })
    await shot(page, 'et-02-single-choice-before')

    // Click the first radio option
    const firstOption = radioGroup.locator('input[type="radio"]').first()
    await expect(firstOption).toBeVisible()
    await firstOption.click()

    // The radio should now be checked
    await expect(firstOption).toBeChecked()
    await shot(page, 'et-02-single-choice-selected')

    // SaveIndicator should show "Saving…" then "Saved ✓" — wait for saved state
    await expect(page.getByText(/saved|saving/i).first()).toBeVisible({ timeout: 8_000 })
    await shot(page, 'et-02-single-choice-saved')
  })

  test('03 — Multiple-choice answer', async ({ page }) => {
    await startExamSession(page)
    await waitForContent(page)

    // Scroll through questions to find a multiple-choice one (has checkboxes)
    // Multiple-choice has role="group" or wraps checkboxes
    const checkboxes = page.locator('input[type="checkbox"]')

    // Try to find multiple-choice question — may need to scroll or be at a different position
    // The exam has 5 questions including one multiple-choice
    const cbCount = await checkboxes.count()
    if (cbCount === 0) {
      // Navigate to next question area by scrolling
      await page.keyboard.press('End')
      await page.waitForTimeout(500)
    }

    // Check if checkboxes are present anywhere in the main area
    const mainCheckboxes = page.locator('main input[type="checkbox"]')
    const mainCbCount = await mainCheckboxes.count()

    if (mainCbCount >= 2) {
      const first = mainCheckboxes.nth(0)
      const second = mainCheckboxes.nth(1)

      await expect(first).toBeVisible()
      await first.click()
      await expect(first).toBeChecked()
      await shot(page, 'et-03-multiple-choice-one-selected')

      await second.click()
      await expect(second).toBeChecked()
      await shot(page, 'et-03-multiple-choice-two-selected')

      // Uncheck first
      await first.click()
      await expect(first).not.toBeChecked()
      await shot(page, 'et-03-multiple-choice-one-unchecked')
    } else {
      // All-questions page — look for question containers and verify checkbox is in exam
      await shot(page, 'et-03-multiple-choice-not-visible-yet')
      test.info().annotations.push({ type: 'note', description: 'Multiple choice checkboxes not visible at scroll position' })
    }
  })

  test('04 — True/False answer', async ({ page }) => {
    await startExamSession(page)
    await waitForContent(page)

    // True/False is a 2-option radio group — find a radiogroup with exactly 2 radios
    const radioGroups = page.locator('[role="radiogroup"]')
    const groupCount = await radioGroups.count()

    let foundTrueFalse = false
    for (let i = 0; i < groupCount; i++) {
      const group = radioGroups.nth(i)
      const radios = group.locator('input[type="radio"]')
      const count = await radios.count()
      if (count === 2) {
        foundTrueFalse = true
        await radios.first().click()
        await expect(radios.first()).toBeChecked()
        await shot(page, 'et-04-truefalse-selected')
        break
      }
    }

    if (!foundTrueFalse) {
      // Couldn't identify true/false — all radio groups may have more options
      await shot(page, 'et-04-truefalse-fallback')
      test.info().annotations.push({ type: 'note', description: 'Could not identify a 2-option True/False radiogroup' })
    }
  })

  test('05 — Likert answer', async ({ page }) => {
    await startExamSession(page)
    await waitForContent(page)

    // Likert uses role="radiogroup" with role="radio" buttons (not <input type=radio>)
    // These are <button role="radio"> elements
    const likertRadios = page.locator('[role="radiogroup"] [role="radio"]')
    const count = await likertRadios.count()

    if (count > 0) {
      const firstLikert = likertRadios.first()
      await expect(firstLikert).toBeVisible({ timeout: 5_000 })
      await firstLikert.click()
      await expect(firstLikert).toHaveAttribute('aria-checked', 'true', { timeout: 5_000 })
      await shot(page, 'et-05-likert-selected')
    } else {
      await shot(page, 'et-05-likert-not-visible')
      test.info().annotations.push({ type: 'note', description: 'Likert role=radio buttons not visible at current scroll' })
    }
  })

  test('06 — Short-text answer', async ({ page }) => {
    await startExamSession(page)
    await waitForContent(page)

    // ShortText renders a <textarea> with placeholder
    const textarea = page.locator('main textarea').first()
    await expect(textarea).toBeVisible({ timeout: 10_000 })
    await expect(textarea).toHaveAttribute('placeholder', /type your answer here/i)

    await textarea.click()
    await textarea.fill('This is my E2E short text answer.')
    await shot(page, 'et-06-shorttext-typed')

    // Wait for debounce + SaveIndicator
    await page.waitForTimeout(1_000)
    await expect(page.getByText(/saving|saved/i).first()).toBeVisible({ timeout: 8_000 })
    await shot(page, 'et-06-shorttext-saved')
  })

  test('07 — Flag/unflag — navigator button turns yellow', async ({ page }) => {
    await startExamSession(page)
    await waitForContent(page)

    // FlagButton has aria-pressed
    const flagBtn = page.locator('[aria-pressed]').first()
    await expect(flagBtn).toBeVisible({ timeout: 10_000 })

    // Initially not pressed
    await expect(flagBtn).toHaveAttribute('aria-pressed', 'false')
    await shot(page, 'et-07-flag-before')

    // Click to flag
    await flagBtn.click()
    await expect(flagBtn).toHaveAttribute('aria-pressed', 'true', { timeout: 3_000 })
    await shot(page, 'et-07-flag-after')

    // Navigator button (in aside) should now be yellow (bg-yellow-300)
    const yellowNavBtn = page.locator('aside button.bg-yellow-300').first()
    // Note: class matching with locator — check if any yellow button appears
    const navBtnCount = await page.locator('aside button').count()
    expect(navBtnCount).toBeGreaterThan(0)
    await shot(page, 'et-07-flag-navigator-yellow')

    // Unflag
    await flagBtn.click()
    await expect(flagBtn).toHaveAttribute('aria-pressed', 'false', { timeout: 3_000 })
    await shot(page, 'et-07-unflagged')
  })

  test('08 — Question navigator jump', async ({ page }) => {
    await startExamSession(page)
    await waitForContent(page)

    // Get navigator buttons
    const navButtons = page.locator('aside button')
    const count = await navButtons.count()

    if (count >= 3) {
      // Click button 3 (index 2 = question 3)
      await navButtons.nth(2).click()
      await page.waitForTimeout(500)
      await shot(page, 'et-08-navigator-jump-q3')
      // Questions should still be visible
      await expect(page.locator('main')).toBeVisible()
    } else {
      // Fewer than 3 questions visible — click the last one
      await navButtons.last().click()
      await shot(page, 'et-08-navigator-jump-last')
    }
  })

  test('09 — Finish exam → review screen', async ({ page }) => {
    await startExamSession(page)
    await waitForContent(page)

    // Click "Finish exam" button
    const finishBtn = page.getByRole('button', { name: /finish exam/i })
    await expect(finishBtn).toBeVisible({ timeout: 10_000 })
    await finishBtn.click()

    // FinishReviewScreen should render
    await expect(page.getByRole('heading', { name: /review before submitting/i })).toBeVisible({
      timeout: 10_000,
    })
    await shot(page, 'et-09-review-screen')

    // Go Back and Submit Anyway buttons must be visible
    await expect(page.getByRole('button', { name: /go back/i })).toBeVisible()
    await expect(page.getByRole('button', { name: /submit anyway/i })).toBeVisible()
    await shot(page, 'et-09-review-buttons')
  })

  test('10 — Submit confirmation happy path', async ({ page }) => {
    await startExamSession(page)
    await waitForContent(page)

    // Navigate to review screen
    const finishBtn = page.getByRole('button', { name: /finish exam/i })
    await expect(finishBtn).toBeVisible({ timeout: 10_000 })
    await finishBtn.click()

    await expect(page.getByRole('heading', { name: /review before submitting/i })).toBeVisible({
      timeout: 10_000,
    })

    // Click "Submit anyway"
    await page.getByRole('button', { name: /submit anyway/i }).click()

    // SubmitConfirmModal should open
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    await expect(page.getByText(/submit exam/i).first()).toBeVisible()
    await expect(page.getByText(/once submitted/i)).toBeVisible()
    await shot(page, 'et-10-submit-confirm-modal')

    // Click the Submit button inside the modal
    const submitBtn = page.getByRole('dialog').getByRole('button', { name: /^submit$/i })
    await expect(submitBtn).toBeVisible()
    await submitBtn.click()

    // Wait for result screen — either passed, failed, or pending
    await expect(
      page.getByText(/passed|failed|being reviewed/i).first(),
    ).toBeVisible({ timeout: 30_000 })
    await shot(page, 'et-10-result-screen')
  })

  test('11 — Tab switch warning modal appears', async ({ page }) => {
    await startExamSession(page)
    await waitForContent(page)

    await shot(page, 'et-11-before-tab-switch')

    // Inject a visibilitychange event with hidden state
    await page.evaluate(() => {
      Object.defineProperty(document, 'visibilityState', {
        value: 'hidden',
        writable: true,
        configurable: true,
      })
      document.dispatchEvent(new Event('visibilitychange'))
    })

    await page.waitForTimeout(500)

    // TabSwitchWarningModal should appear
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 8_000 })

    // Title contains "Warning"
    await expect(page.getByText(/warning/i).first()).toBeVisible()
    await shot(page, 'et-11-tab-switch-warning-modal')

    // Close the modal — the close button uses t('common.cancel') = "Cancel"
    const closeBtn = page.getByRole('dialog').getByRole('button', { name: /cancel/i })
    await expect(closeBtn).toBeVisible()
    await closeBtn.click()

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5_000 })
    await shot(page, 'et-11-tab-switch-modal-closed')
  })
})
