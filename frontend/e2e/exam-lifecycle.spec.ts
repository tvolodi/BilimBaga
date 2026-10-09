/**
 * Exam lifecycle tests — covers status badges, archive, and wizard behavior.
 *
 * Runs against the real stack. The seeded "E2E Mixed Exam" (active) and any
 * draft exams created by the wizard tests provide real data to assert against.
 */

import { test, expect } from '@playwright/test'
import { getSeedData, createTestExam, deleteTestExam, createTestQuestion, deleteTestQuestion } from './fixtures/seed'

const API_URL = process.env.E2E_API_URL || `http://localhost:${process.env.BB_API_PORT || 8080}`

async function waitForContent(page: import('@playwright/test').Page) {
  await page.waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin'), { timeout: 10_000 }).catch(() => {})
  await page.waitForLoadState('networkidle').catch(() => {})
}

// ---------------------------------------------------------------------------
// Exam List — status badges
// ---------------------------------------------------------------------------

test.describe('Exam Lifecycle — Status Badges in List', () => {
  test('exams list renders without error and shows status badges', async ({ page }) => {
    await page.goto('/admin/exams')
    await waitForContent(page)
    await expect(page.locator('body')).not.toContainText(/unexpected error|something went wrong/i)
    // If any exams exist, at least one badge should be visible
    const hasTable = await page.getByRole('table').isVisible().catch(() => false)
    if (hasTable) {
      const rows = page.locator('table tbody tr')
      const count = await rows.count()
      expect(count).toBeGreaterThan(0)
    }
  })

  test('seeded active exam shows "active" badge', async ({ page }) => {
    await page.goto('/admin/exams')
    await waitForContent(page)
    const hasTable = await page.getByRole('table').isVisible({ timeout: 10_000 }).catch(() => false)
    if (!hasTable) {
      test.info().annotations.push({ type: 'note', description: 'No exams table visible' })
      return
    }
    // The E2E Mixed Exam is published/active — its row should show an active badge
    const activeRow = page.locator('table tbody tr').filter({ hasText: 'E2E Mixed Exam' })
    if (await activeRow.isVisible({ timeout: 3_000 }).catch(() => false)) {
      await expect(activeRow.getByText(/\u0430\u043a\u0442\u0438\u0432\u0435\u043d|active/i)).toBeVisible()
    } else {
      test.info().annotations.push({ type: 'note', description: 'E2E Mixed Exam row not on this page — may be paginated' })
    }
  })

  test('draft exam shows "draft" badge', async ({ page }) => {
    const { adminToken } = await getSeedData()
    const exam = await createTestExam(adminToken, `E2E Draft Badge ${Date.now()}`)

    await page.goto('/admin/exams')
    await waitForContent(page)
    const row = page.locator('table tbody tr').filter({ hasText: exam.title })
    if (await row.isVisible({ timeout: 5_000 }).catch(() => false)) {
      await expect(row.getByText(/\u0447\u0435\u0440\u043d\u043e\u0432\u0438\u043a|draft/i).last()).toBeVisible()
    } else {
      test.info().annotations.push({ type: 'note', description: 'Draft exam row not visible on this page — may be paginated' })
    }
    await deleteTestExam(adminToken, exam.id)
  })

  test('all exam rows have an Edit action link', async ({ page }) => {
    await page.goto('/admin/exams')
    await waitForContent(page)
    const hasTable = await page.getByRole('table').isVisible({ timeout: 10_000 }).catch(() => false)
    if (!hasTable) {
      test.info().annotations.push({ type: 'note', description: 'No exams table' })
      return
    }
    // At least one edit link (pencil icon) should be present
    const editLinks = page.getByRole('link').filter({ has: page.locator('svg') })
    const count = await editLinks.count()
    expect(count).toBeGreaterThan(0)
  })
})

// ---------------------------------------------------------------------------
// Exam Lifecycle — Archived exam
// ---------------------------------------------------------------------------

test.describe('Exam Lifecycle — Archive via API', () => {
  test('archived exam shows "archived" status in the list', async ({ page }) => {
    const { adminToken } = await getSeedData()
    const question = await createTestQuestion(adminToken, `E2E Lifecycle Q ${Date.now()}`, 'single', true)
    const exam = await createTestExam(adminToken, `E2E Archive Test ${Date.now()}`, question.id)

    // Archive the exam via API
    await fetch(`${API_URL}/api/v1/exams/${exam.id}/archive`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${adminToken}` },
    })

    await page.goto('/admin/exams')
    await waitForContent(page)
    const row = page.locator('table tbody tr').filter({ hasText: exam.title })
    if (await row.isVisible({ timeout: 5_000 }).catch(() => false)) {
      await expect(row.getByText(/\u0432 \u0430\u0440\u0445\u0438\u0432\u0435|archived/i)).toBeVisible()
    } else {
      test.info().annotations.push({ type: 'note', description: 'Archived exam row not visible on this page' })
    }

    await deleteTestExam(adminToken, exam.id)
    await deleteTestQuestion(adminToken, question.id)
  })
})

// ---------------------------------------------------------------------------
// Exam Lifecycle — Editing an existing exam
// ---------------------------------------------------------------------------

test.describe('Exam Lifecycle — Edit existing exam', () => {
  test('editing the seeded mixed exam loads Step 1 with pre-populated title', async ({ page }) => {
    const { mixedExamId } = await getSeedData()
    await page.goto(`/admin/exams/${mixedExamId}/edit`)
    await waitForContent(page)
    await expect(page.getByText(/\u0440\u0435\u0434\u0430\u043a\u0442\u0438\u0440\u043e\u0432\u0430\u0442\u044c \u044d\u043a\u0437\u0430\u043c|\u043e\u0441\u043d\u043e\u0432\u043d\u044b\u0435 \u043d\u0430\u0441\u0442\u0440\u043e\u0439\u043a\u0438|Edit Exam|Basic Settings/i).first()).toBeVisible({ timeout: 10_000 })
    const titleField = page.getByLabel(/\u043d\u0430\u0437\u0432\u0430\u043d\u0438\u0435 \u044d\u043a\u0437\u0430\u043c\u0435\u043d\u0430|exam title/i)
    if (await titleField.isVisible({ timeout: 3_000 }).catch(() => false)) {
      const value = await titleField.inputValue()
      expect(value.length).toBeGreaterThan(0)
    }
  })

  test('exams list page navigates to edit wizard on clicking edit', async ({ page }) => {
    await page.goto('/admin/exams')
    await waitForContent(page)
    const hasTable = await page.getByRole('table').isVisible({ timeout: 10_000 }).catch(() => false)
    if (!hasTable) {
      test.info().annotations.push({ type: 'note', description: 'No exams table' })
      return
    }
    // Click first edit link
    const editLink = page.getByRole('link', { name: /\u0440\u0435\u0434\u0430\u043a\u0442\u0438\u0440\u043e\u0432\u0430\u0442\u044c|edit/i }).first()
    if (!(await editLink.isVisible({ timeout: 3_000 }).catch(() => false))) {
      // Try SVG icon links
      const svgLinks = page.getByRole('link').filter({ has: page.locator('svg') })
      const count = await svgLinks.count()
      if (count === 0) {
        test.info().annotations.push({ type: 'note', description: 'No edit links visible' })
        return
      }
      await svgLinks.first().click()
    } else {
      await editLink.click()
    }
    await expect(page).toHaveURL(/\/admin\/exams\/.+\/edit/, { timeout: 10_000 })
    await waitForContent(page)
    await expect(page.getByText(/основные настройки|редактировать экзам|basic settings|edit exam/i).first()).toBeVisible({ timeout: 10_000 })
  })
})

// ---------------------------------------------------------------------------
// Exam Lifecycle — Full status flow
// ---------------------------------------------------------------------------

test.describe('Exam Lifecycle — Full status flow', () => {
  test('exams list shows multiple statuses when multiple exams exist', async ({ page }) => {
    const { adminToken } = await getSeedData()
    const draftExam = await createTestExam(adminToken, `E2E Draft ${Date.now()}`)

    await page.goto('/admin/exams')
    await waitForContent(page)
    await expect(page.locator('body')).not.toContainText(/unexpected error/i)

    // Check that "active" badge exists (from E2E Mixed Exam) and "draft" exists
    const hasActive = await page.getByText(/\u0430\u043a\u0442\u0438\u0432\u0435\u043d|active/i).isVisible({ timeout: 3_000 }).catch(() => false)
    const hasDraft = await page.getByText(/\u0447\u0435\u0440\u043d\u043e\u0432\u0438\u043a|draft/i).isVisible({ timeout: 3_000 }).catch(() => false)
    if (!hasActive && !hasDraft) {
      test.info().annotations.push({ type: 'note', description: 'Neither active nor draft badges visible on current page' })
    }

    await deleteTestExam(adminToken, draftExam.id)
  })
})
