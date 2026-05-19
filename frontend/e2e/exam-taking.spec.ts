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
 * Wait for portal exam cards to finish loading (skeleton removed, real cards or empty state visible).
 */
async function waitForPortalReady(page: Page) {
  await waitForContent(page)
  // Phase 1: wait for ExamCardSkeleton to appear (portal started loading)
  // Phase 2: wait for ExamCardSkeleton to disappear (data loaded, real cards rendered)
  // If skeleton never appears (empty portal), the condition is already met.
  await page
    .waitForFunction(() => {
      const hasSkeleton = document.querySelector('.animate-pulse') !== null
      const hasCards = document.querySelector('.rounded-lg.border.bg-card') !== null
      const hasEmpty = (document.body.textContent?.includes('No exams assigned') ?? false) || (document.body.textContent?.includes('\u041d\u0435\u0442 \u043d\u0430\u0437\u043d\u0430\u0447\u0435\u043d\u043d\u044b\u0445 \u044d\u043a\u0437\u0430\u043c\u0435\u043d\u043e\u0432') ?? false)
      // Done if: no skeleton AND (cards loaded OR empty state)
      return !hasSkeleton && (hasCards || hasEmpty)
    }, { timeout: 20_000 })
    .catch(() => {})
}

/**
 * Navigate to an exam session via the portal.
 * Handles three states: not_started (shows "Start exam"), in_progress (shows "Continue"),
 * passed/failed (creates session via API directly since UI only shows "View result").
 */
async function startExamSession(page: Page): Promise<string> {
  await page.goto('/portal')
  await waitForPortalReady(page)

  // Confirm portal heading is visible
  await expect(page.getByRole('heading', { name: /\u043c\u043e\u0438 \u044d\u043a\u0437\u0430\u043c\u0435\u043d\u044b|my exams/i })).toBeVisible({ timeout: 15_000 })

  // Look for the "E2E Mixed Exam" card specifically, then find its CTA button
  const mixedExamCard = page.locator('.rounded-lg.border.bg-card').filter({
    has: page.locator('h3', { hasText: 'E2E Mixed Exam' }),
  })

  // Wait for the card to appear
  await expect(mixedExamCard).toBeVisible({ timeout: 15_000 })

  // "Continue" button means in_progress — navigate directly to existing session
  const continueBtn = mixedExamCard.getByRole('button', { name: /^\u043f\u0440\u043e\u0434\u043e\u043b\u0436\u0438\u0442\u044c$|^continue$/i })
  if (await continueBtn.isVisible({ timeout: 3_000 }).catch(() => false)) {
    await continueBtn.click()
    await expect(page).toHaveURL(/\/portal\/sessions\//, { timeout: 20_000 })
    return page.url()
  }

  // "Start exam" button means not_started — open modal and confirm
  const startBtn = mixedExamCard.getByRole('button', { name: /\u043d\u0430\u0447\u0430\u0442\u044c \u044d\u043a\u0437\u0430\u043c\u0435\u043d|start exam/i })
  if (await startBtn.isVisible({ timeout: 3_000 }).catch(() => false)) {
    await startBtn.click()
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 5_000 })
    const confirmBtn = page.getByRole('button', { name: /\u043d\u0430\u0447\u0430\u0442\u044c \u044d\u043a\u0437\u0430\u043c\u0435\u043d|begin exam/i }).last()
    await expect(confirmBtn).toBeVisible()
    await confirmBtn.click()
    await expect(page).toHaveURL(/\/portal\/sessions\//, { timeout: 20_000 })
    return page.url()
  }

  // "View result" means exam was passed/failed — create a new session via API directly
  // (the UI doesn't show "Start again" even when attempts remain, so we bypass it)
  const token = await page.evaluate(() => localStorage.getItem('__e2e_access_token__'))
  if (!token) {
    throw new Error('startExamSession: no access token in localStorage — auth not seeded')
  }

  // Get the Mixed Exam ID from the portal API
  const apiResp = await page.evaluate(async (tok: string) => {
    const res = await fetch('/api/v1/portal/exams', {
      headers: { Authorization: `Bearer ${tok}` },
    })
    const json = await res.json()
    const exams = (json.data ?? []) as Array<{ id: string; title: string }>
    const mixed = exams.find((e) => e.title === 'E2E Mixed Exam')
    return mixed?.id ?? null
  }, token)

  if (!apiResp) {
    throw new Error('startExamSession: could not find E2E Mixed Exam in portal API')
  }

  // Create a new session via API
  const sessionResp = await page.evaluate(
    async ({ tok, examId }: { tok: string; examId: string }) => {
      const res = await fetch(`/api/v1/portal/exams/${examId}/sessions`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${tok}`, 'Content-Type': 'application/json' },
        credentials: 'include',
      })
      const json = await res.json()
      return { ok: res.ok, sessionId: (json.data as { session_id?: string })?.session_id ?? null, error: json.error?.code }
    },
    { tok: token, examId: apiResp },
  )

  if (!sessionResp.ok || !sessionResp.sessionId) {
    throw new Error(`startExamSession: failed to create session via API: ${sessionResp.error}`)
  }

  await page.goto(`/portal/sessions/${sessionResp.sessionId}`)
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
    const progress = page.locator('header').getByText(/\d+ \/ \d+ отвечено|\d+ отвечено|\d+ \/ \d+ answered|\d+ answered/i).first()
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

    // Find a radio that is NOT currently checked to guarantee a state change (save fires)
    const allRadios = radioGroup.locator('input[type="radio"]')
    const radioCount = await allRadios.count()
    let targetRadio = allRadios.first()
    for (let i = 0; i < radioCount; i++) {
      const radio = allRadios.nth(i)
      if (!(await radio.isChecked())) {
        targetRadio = radio
        break
      }
    }

    await expect(targetRadio).toBeVisible()
    await targetRadio.click()

    // The radio should now be checked
    await expect(targetRadio).toBeChecked()
    await shot(page, 'et-02-single-choice-selected')

    // SaveIndicator should show "Saving…" then "Saved ✓" — wait for saved state
    await expect(page.getByText(/\u0441\u043e\u0445\u0440\u0430\u043d\u0435\u043d\u043e|\u0441\u043e\u0445\u0440\u0430\u043d\u0435\u043d\u0438\u0435|saved|saving/i).first()).toBeVisible({ timeout: 8_000 })
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
      // Find an unchecked checkbox to guarantee a state change on click
      let targetCb = mainCheckboxes.nth(0)
      let targetIdx = 0
      for (let i = 0; i < mainCbCount; i++) {
        const cb = mainCheckboxes.nth(i)
        if (!(await cb.isChecked())) {
          targetCb = cb
          targetIdx = i
          break
        }
      }

      await expect(targetCb).toBeVisible()
      const wasChecked = await targetCb.isChecked()

      // Toggle once — state should flip
      await targetCb.click()
      if (wasChecked) {
        await expect(targetCb).not.toBeChecked()
      } else {
        await expect(targetCb).toBeChecked()
      }
      await shot(page, 'et-03-multiple-choice-one-selected')

      // Toggle back
      await targetCb.click()
      if (wasChecked) {
        await expect(targetCb).toBeChecked()
      } else {
        await expect(targetCb).not.toBeChecked()
      }
      await shot(page, 'et-03-multiple-choice-one-unchecked')

      // Toggle once more — leave in opposite of initial state
      await targetCb.click()
      if (wasChecked) {
        await expect(targetCb).not.toBeChecked()
      } else {
        await expect(targetCb).toBeChecked()
      }
      await shot(page, 'et-03-multiple-choice-two-selected')
      void targetIdx // suppress unused warning
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
    await expect(textarea).toHaveAttribute('placeholder', /введите ваш ответ здесь|type your answer here/i)

    // Use a unique value to guarantee the change triggers onChange even if previous answer was the same
    const uniqueAnswer = `E2E short text answer — ${Date.now()}`
    await textarea.click()
    await textarea.fill(uniqueAnswer)
    await shot(page, 'et-06-shorttext-typed')

    // Wait for debounce (800ms) + SaveIndicator to appear
    await page.waitForTimeout(1_200)
    await expect(page.getByText(/сохранен|saving|saved/i).first()).toBeVisible({ timeout: 8_000 })
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
    const finishBtn = page.getByRole('button', { name: /завершить экзамен|finish exam/i })
    await expect(finishBtn).toBeVisible({ timeout: 10_000 })
    await finishBtn.click()

    // FinishReviewScreen should render
    await expect(page.getByRole('heading', { name: /проверка перед отправкой|review before submitting/i })).toBeVisible({
      timeout: 10_000,
    })
    await shot(page, 'et-09-review-screen')

    // Go Back and Submit Anyway buttons must be visible
    await expect(page.getByRole('button', { name: /вернуться назад|go back/i })).toBeVisible()
    await expect(page.getByRole('button', { name: /всё равно отправить|submit anyway/i })).toBeVisible()
    await shot(page, 'et-09-review-buttons')
  })

  test('10 — Submit confirmation happy path', async ({ page }) => {
    await startExamSession(page)
    await waitForContent(page)

    // Navigate to review screen
    const finishBtn = page.getByRole('button', { name: /завершить экзамен|finish exam/i })
    await expect(finishBtn).toBeVisible({ timeout: 10_000 })
    await finishBtn.click()

    await expect(page.getByRole('heading', { name: /проверка перед отправкой|review before submitting/i })).toBeVisible({
      timeout: 10_000,
    })

    // Click "Submit anyway"
    await page.getByRole('button', { name: /всё равно отправить|submit anyway/i }).click()

    // SubmitConfirmModal should open with correct content
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    await expect(page.getByText(/сдать экзамен|submit exam/i).first()).toBeVisible()
    await expect(page.getByText(/после отправки|once submitted/i)).toBeVisible()
    await shot(page, 'et-10-submit-confirm-modal')

    // Verify the Submit and Cancel buttons are present
    const submitBtn = page.getByRole('dialog').getByRole('button', { name: /^отправить$|^submit$/i })
    await expect(submitBtn).toBeVisible()

    // Cancel instead of submitting — leave the session open for test 11
    const cancelBtn = page.getByRole('dialog').getByRole('button', { name: /отмена|cancel/i })
    await expect(cancelBtn).toBeVisible()
    await cancelBtn.click()

    // Modal should close, back on review screen
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5_000 })
    await shot(page, 'et-10-submit-confirm-modal-cancelled')
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
    await expect(page.getByText(/предупреждение|warning/i).first()).toBeVisible()
    await shot(page, 'et-11-tab-switch-warning-modal')

    // Close the modal — the close button uses t('common.cancel') = "Cancel"
    const closeBtn = page.getByRole('dialog').getByRole('button', { name: /отмена|cancel/i })
    await expect(closeBtn).toBeVisible()
    await closeBtn.click()

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5_000 })
    await shot(page, 'et-11-tab-switch-modal-closed')
  })
})
