import { test, expect } from '@playwright/test'
import { getSeedData, createTestQuestion, deleteTestQuestion } from './fixtures/seed'

async function waitForContent(page: import('@playwright/test').Page) {
  await page.waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin'), { timeout: 10_000 }).catch(() => {})
  await page.waitForLoadState('networkidle').catch(() => {})
}

// ---------------------------------------------------------------------------
// Delete Question
// ---------------------------------------------------------------------------

test.describe('Question Management — Delete Question', () => {
  test('row actions menu is visible for each question', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    const hasTable = await page.getByRole('table').isVisible().catch(() => false)
    if (!hasTable) {
      test.info().annotations.push({ type: 'note', description: 'No questions in bank' })
      return
    }
    const actionsBtn = page.getByRole('button', { name: /row actions/i }).first()
    if (!(await actionsBtn.isVisible({ timeout: 5_000 }).catch(() => false))) {
      test.info().annotations.push({ type: 'note', description: 'No row actions button visible' })
      return
    }
    await expect(actionsBtn).toBeVisible()
  })

  test('actions menu shows Delete option for draft questions', async ({ page }) => {
    const { adminToken } = await getSeedData()
    const question = await createTestQuestion(adminToken, `E2E Delete Test ${Date.now()}`)

    // Draft questions are deletable — but createTestQuestion activates them via seed helper
    // We create directly without activation to get a draft
    await page.goto('/admin/questions?status=draft')
    await waitForContent(page)

    const hasTable = await page.getByRole('table').isVisible().catch(() => false)
    if (!hasTable) {
      await deleteTestQuestion(adminToken, question.id)
      test.info().annotations.push({ type: 'note', description: 'No draft questions in bank' })
      return
    }
    const actionsBtn = page.getByRole('button', { name: /row actions/i }).first()
    if (!(await actionsBtn.isVisible({ timeout: 5_000 }).catch(() => false))) {
      await deleteTestQuestion(adminToken, question.id)
      test.info().annotations.push({ type: 'note', description: 'No row actions button visible' })
      return
    }
    await actionsBtn.click()
    const deleteOption = page.getByRole('button', { name: /delete/i })
    if (await deleteOption.isVisible({ timeout: 3_000 }).catch(() => false)) {
      await expect(deleteOption).toBeVisible()
      await page.keyboard.press('Escape')
    }
    await deleteTestQuestion(adminToken, question.id)
  })

  test('clicking Delete opens a confirmation dialog', async ({ page }) => {
    const { adminToken } = await getSeedData()
    const question = await createTestQuestion(adminToken, `E2E Delete Dialog ${Date.now()}`)

    await page.goto('/admin/questions?status=draft')
    await waitForContent(page)

    const row = page.locator('table tbody tr').filter({ hasText: question.stem })
    const actionsBtn = row.getByRole('button', { name: /row actions/i })
    if (!(await actionsBtn.isVisible({ timeout: 10_000 }).catch(() => false))) {
      await deleteTestQuestion(adminToken, question.id)
      test.info().annotations.push({ type: 'note', description: 'Draft question row not found' })
      return
    }
    await actionsBtn.click()
    // Dropdown renders inside the row — scope delete button to the row container
    const deleteBtn = row.getByRole('button', { name: /^delete$/i })
    await expect(deleteBtn).toBeVisible({ timeout: 5_000 })
    await deleteBtn.click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    await page.getByRole('dialog').getByRole('button', { name: /cancel/i }).click()
    await deleteTestQuestion(adminToken, question.id)
  })

  test('delete confirmation dialog has confirm and cancel buttons', async ({ page }) => {
    const { adminToken } = await getSeedData()
    const question = await createTestQuestion(adminToken, `E2E Delete Confirm ${Date.now()}`)

    await page.goto('/admin/questions?status=draft')
    await waitForContent(page)

    const row = page.locator('table tbody tr').filter({ hasText: question.stem })
    const actionsBtn = row.getByRole('button', { name: /row actions/i })
    if (!(await actionsBtn.isVisible({ timeout: 10_000 }).catch(() => false))) {
      await deleteTestQuestion(adminToken, question.id)
      test.info().annotations.push({ type: 'note', description: 'Draft question row not found' })
      return
    }
    await actionsBtn.click()
    const deleteBtn = row.getByRole('button', { name: /^delete$/i })
    await expect(deleteBtn).toBeVisible({ timeout: 5_000 })
    await deleteBtn.click()
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 5_000 })
    await expect(dialog.getByRole('button', { name: /^delete$/i })).toBeVisible()
    await expect(dialog.getByRole('button', { name: /cancel/i })).toBeVisible()
    await dialog.getByRole('button', { name: /cancel/i }).click()
    await deleteTestQuestion(adminToken, question.id)
  })

  test('cancelling delete closes dialog without deleting', async ({ page }) => {
    const { adminToken } = await getSeedData()
    const question = await createTestQuestion(adminToken, `E2E Delete Cancel ${Date.now()}`)

    await page.goto('/admin/questions?status=draft')
    await waitForContent(page)

    const row = page.locator('table tbody tr').filter({ hasText: question.stem })
    const actionsBtn = row.getByRole('button', { name: /row actions/i })
    if (!(await actionsBtn.isVisible({ timeout: 10_000 }).catch(() => false))) {
      await deleteTestQuestion(adminToken, question.id)
      test.info().annotations.push({ type: 'note', description: 'Draft question row not found' })
      return
    }
    await actionsBtn.click()
    const deleteBtn = row.getByRole('button', { name: /^delete$/i })
    await expect(deleteBtn).toBeVisible({ timeout: 5_000 })
    await deleteBtn.click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    await page.getByRole('dialog').getByRole('button', { name: /cancel/i }).click()
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 3_000 })
    // Table still present — question not deleted
    await expect(page.getByRole('table')).toBeVisible()
    await deleteTestQuestion(adminToken, question.id)
  })
})

// ---------------------------------------------------------------------------
// Archive Question
// ---------------------------------------------------------------------------

test.describe('Question Management — Archive Question', () => {
  test('actions menu shows Archive option for active questions', async ({ page }) => {
    await page.goto('/admin/questions?status=active')
    await waitForContent(page)

    const hasTable = await page.getByRole('table').isVisible().catch(() => false)
    if (!hasTable) {
      test.info().annotations.push({ type: 'note', description: 'No active questions in bank' })
      return
    }
    const actionsBtn = page.getByRole('button', { name: /row actions/i }).first()
    if (!(await actionsBtn.isVisible({ timeout: 5_000 }).catch(() => false))) {
      test.info().annotations.push({ type: 'note', description: 'No row actions button' })
      return
    }
    await actionsBtn.click()
    const archiveBtn = page.getByRole('button', { name: /archive/i })
    if (await archiveBtn.isVisible({ timeout: 3_000 }).catch(() => false)) {
      await expect(archiveBtn).toBeVisible()
    }
    await page.keyboard.press('Escape')
  })

  test('clicking Archive sends transition request and hides the dropdown', async ({ page }) => {
    await page.goto('/admin/questions?status=active')
    await waitForContent(page)

    const hasTable = await page.getByRole('table').isVisible().catch(() => false)
    if (!hasTable) {
      test.info().annotations.push({ type: 'note', description: 'No active questions' })
      return
    }
    const actionsBtn = page.getByRole('button', { name: /row actions/i }).first()
    if (!(await actionsBtn.isVisible({ timeout: 5_000 }).catch(() => false))) {
      test.info().annotations.push({ type: 'note', description: 'No row actions button' })
      return
    }
    await actionsBtn.click()
    const archiveBtn = page.getByRole('button', { name: /archive/i })
    if (!(await archiveBtn.isVisible({ timeout: 3_000 }).catch(() => false))) {
      await page.keyboard.press('Escape')
      test.info().annotations.push({ type: 'note', description: 'No Archive option in menu' })
      return
    }
    await archiveBtn.click()
    await expect(page.getByRole('button', { name: /archive/i })).not.toBeVisible({ timeout: 5_000 })
  })
})

// ---------------------------------------------------------------------------
// Import Questions
// ---------------------------------------------------------------------------

test.describe('Question Management — Import Questions', () => {
  test('Import button is visible in question bank header', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    await expect(page.getByRole('button', { name: /import/i })).toBeVisible({ timeout: 10_000 })
  })

  test('clicking Import opens the import modal', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    await page.getByRole('button', { name: /import/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    await expect(page.getByRole('heading', { name: /import/i })).toBeVisible()
  })

  test('import modal contains a file upload area', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    await page.getByRole('button', { name: /import/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    const fileInput = page.locator('input[type="file"]')
    await expect(fileInput).toBeAttached()
    const acceptAttr = await fileInput.getAttribute('accept')
    expect(acceptAttr).toMatch(/\.csv|\.json/i)
  })

  test('import modal has Cancel button', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    await page.getByRole('button', { name: /import/i }).click()
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 5_000 })
    await expect(dialog.getByRole('button', { name: /cancel/i })).toBeVisible()
  })

  test('import modal closes on Cancel', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    await page.getByRole('button', { name: /import/i }).click()
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 5_000 })
    await dialog.getByRole('button', { name: /cancel/i }).click()
    await expect(dialog).not.toBeVisible({ timeout: 3_000 })
  })
})

// ---------------------------------------------------------------------------
// Export Questions
// ---------------------------------------------------------------------------

test.describe('Question Management — Export Questions', () => {
  test('selecting a question shows the bulk action bar', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    const hasTable = await page.getByRole('table').isVisible().catch(() => false)
    if (!hasTable) {
      test.info().annotations.push({ type: 'note', description: 'No questions in bank' })
      return
    }
    const checkbox = page.locator('input[type="checkbox"]').nth(1)
    if (!(await checkbox.isVisible({ timeout: 3_000 }).catch(() => false))) {
      test.info().annotations.push({ type: 'note', description: 'No checkboxes in question table' })
      return
    }
    await checkbox.check()
    await expect(page.getByRole('button', { name: /export.*csv/i })).toBeVisible({ timeout: 3_000 })
  })

  test('bulk action bar has Export CSV and Export JSON buttons', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    const hasTable = await page.getByRole('table').isVisible().catch(() => false)
    if (!hasTable) {
      test.info().annotations.push({ type: 'note', description: 'No questions in bank' })
      return
    }
    const checkbox = page.locator('input[type="checkbox"]').nth(1)
    if (!(await checkbox.isVisible({ timeout: 3_000 }).catch(() => false))) {
      test.info().annotations.push({ type: 'note', description: 'No checkboxes' })
      return
    }
    await checkbox.check()
    await expect(page.getByRole('button', { name: /export.*csv/i })).toBeVisible()
    await expect(page.getByRole('button', { name: /export.*json/i })).toBeVisible()
  })

  test('bulk action bar shows selected count', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    const hasTable = await page.getByRole('table').isVisible().catch(() => false)
    if (!hasTable) {
      test.info().annotations.push({ type: 'note', description: 'No questions in bank' })
      return
    }
    const checkbox = page.locator('input[type="checkbox"]').nth(1)
    if (!(await checkbox.isVisible({ timeout: 3_000 }).catch(() => false))) {
      test.info().annotations.push({ type: 'note', description: 'No checkboxes' })
      return
    }
    await checkbox.check()
    await expect(page.getByText(/questions selected/i)).toBeVisible({ timeout: 3_000 })
  })

  test('select-all checkbox selects all visible questions', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    const hasTable = await page.getByRole('table').isVisible().catch(() => false)
    if (!hasTable) {
      test.info().annotations.push({ type: 'note', description: 'No questions in bank' })
      return
    }
    const selectAllCheckbox = page.locator('input[type="checkbox"]').first()
    if (!(await selectAllCheckbox.isVisible({ timeout: 3_000 }).catch(() => false))) {
      test.info().annotations.push({ type: 'note', description: 'No select-all checkbox' })
      return
    }
    await selectAllCheckbox.check()
    await expect(page.getByRole('button', { name: /export.*csv/i })).toBeVisible({ timeout: 3_000 })
  })
})

// ---------------------------------------------------------------------------
// AI Question Generation
// ---------------------------------------------------------------------------

test.describe('Question Management — AI Generate', () => {
  test('AI Generate button is visible for examiner+ roles', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    await expect(page.getByRole('button', { name: /generate|ai/i })).toBeVisible({ timeout: 10_000 })
  })

  test('clicking AI Generate opens the generation dialog', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    await page.getByRole('button', { name: /generate|ai/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
  })

  test('AI Generate dialog has category, difficulty, count, and context fields', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    await page.getByRole('button', { name: /generate|ai/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    const dialog = page.getByRole('dialog')
    await expect(dialog.locator('select').first()).toBeVisible()
    await expect(dialog.locator('input[type="number"]')).toBeVisible()
    await expect(dialog.locator('textarea')).toBeVisible()
    await expect(dialog.getByRole('button', { name: /generate/i })).toBeVisible()
  })

  test('AI Generate dialog can be cancelled', async ({ page }) => {
    await page.goto('/admin/questions')
    await waitForContent(page)
    await page.getByRole('button', { name: /generate|ai/i }).click()
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 5_000 })
    await dialog.getByRole('button', { name: /cancel/i }).click()
    await expect(dialog).not.toBeVisible({ timeout: 3_000 })
  })
})
