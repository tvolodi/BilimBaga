/**
 * FR-BB320 AC-10: exam-mode audit. On the exam-taking page every visible interactive element is at least
 * 44 by 44 px (design-system checklist item 8), and each one shows a visible focus indicator after Tab.
 *
 * A native radio or checkbox is measured by its label: the label is the element a learner taps, and the
 * native control inside it is small by design. The focus check still runs on the native control.
 */

import { test, expect, type Page } from '@playwright/test'
import {
  EMPLOYEE_STORAGE_STATE,
  createTestExam,
  createTestQuestion,
  deleteTestExam,
  deleteTestQuestion,
  getSeedData,
  startEmployeeSession,
} from './fixtures/seed'

const MIN_TARGET_PX = 44
const INTERACTIVE = 'button, a[href], textarea, select, input:not([type="hidden"]), [role="radio"], [role="checkbox"]'

interface Target {
  label: string
  width: number
  height: number
}

async function waitForContent(page: Page) {
  await page.waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin'), { timeout: 10_000 }).catch(() => {})
  await page.waitForLoadState('networkidle').catch(() => {})
}

/** Every visible interactive element, measured as the learner taps it. */
async function visibleTargets(page: Page): Promise<Target[]> {
  return page.evaluate((selector) => {
    return Array.from(document.querySelectorAll<HTMLElement>(selector))
      .map((el) => {
        const isNativeChoice = el instanceof HTMLInputElement && (el.type === 'radio' || el.type === 'checkbox')
        const target = isNativeChoice ? el.closest('label') ?? el : el
        const rect = target.getBoundingClientRect()
        const style = getComputedStyle(target)
        const visible = rect.width > 0 && rect.height > 0 && style.visibility !== 'hidden' && style.display !== 'none'
        const label = (target.textContent ?? '').trim().slice(0, 40) || target.getAttribute('aria-label') || target.tagName
        return { label, width: rect.width, height: rect.height, visible }
      })
      .filter((t) => t.visible)
      .map(({ label, width, height }) => ({ label, width, height }))
  }, INTERACTIVE)
}

test.describe('Exam-mode audit (FR-BB320 AC-10)', () => {
  test.use({ storageState: EMPLOYEE_STORAGE_STATE })

  let questionId = ''
  let examId = ''
  let sessionId = ''

  test.beforeAll(async () => {
    const seed = await getSeedData()
    const question = await createTestQuestion(seed.adminToken, 'Design audit question', 'single', true)
    questionId = question.id
    const exam = await createTestExam(seed.adminToken, 'Design audit exam', question.id, {
      assignToUserId: seed.employeeId,
    })
    examId = exam.id
    sessionId = await startEmployeeSession(examId)
  })

  test.afterAll(async () => {
    const seed = await getSeedData()
    if (examId) await deleteTestExam(seed.adminToken, examId).catch(() => {})
    if (questionId) await deleteTestQuestion(seed.adminToken, questionId).catch(() => {})
  })

  test('every visible interactive element is at least 44 by 44 px', async ({ page }) => {
    await page.goto(`/portal/sessions/${sessionId}`)
    await waitForContent(page)

    const targets = await visibleTargets(page)
    expect(targets.length).toBeGreaterThan(0)
    const small = targets.filter((t) => t.width < MIN_TARGET_PX || t.height < MIN_TARGET_PX)
    expect(small).toEqual([])
  })

  test('each visible interactive element shows a focus indicator after Tab', async ({ page }) => {
    await page.goto(`/portal/sessions/${sessionId}`)
    await waitForContent(page)

    const targets = await visibleTargets(page)
    const missing: string[] = []
    for (let step = 0; step < targets.length; step++) {
      await page.keyboard.press('Tab')
      // Read the focused element's style, then blur it to read the unfocused style, then focus it again so
      // the next Tab continues from here.
      const result = await page.evaluate(() => {
        const el = document.activeElement as HTMLElement | null
        if (!el || el === document.body) return null
        const focused = getComputedStyle(el)
        const focusedOutline = parseFloat(focused.outlineWidth)
        const focusedShadow = focused.boxShadow
        el.blur()
        const unfocused = getComputedStyle(el)
        const unfocusedShadow = unfocused.boxShadow
        el.focus()
        const label = (el.textContent ?? '').trim().slice(0, 40) || el.getAttribute('aria-label') || el.tagName
        return {
          label,
          indicator: focusedOutline > 0 || focusedShadow !== unfocusedShadow,
        }
      })
      if (result && !result.indicator) missing.push(result.label)
    }
    expect(missing).toEqual([])
  })
})
