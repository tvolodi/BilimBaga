/**
 * Tab-switch / anti-cheat events E2E (FR-BB38, issue #16).
 *
 * Requires: make dev running, employee storage state seeded by global-setup.ts
 * ("E2E Mixed Exam" assigned to the employee). Runs in the chromium-live-employee project.
 *
 * The client reports `tab_switch` (document visibilitychange -> hidden) and `blur`
 * (window blur) to POST /api/v1/portal/sessions/:id/events (see useTabSwitchDetection).
 * The server-side policy (log / warn / auto-submit) is exercised by stubbing the response
 * so the UI reaction is deterministic regardless of how the seeded exam is configured;
 * one test additionally checks the real endpoint contract.
 */
import { test, expect, type Page, type Route } from '@playwright/test'

const EVENTS_URL = /\/api\/v1\/portal\/sessions\/[^/]+\/events$/

async function startMixedExamSession(page: Page): Promise<string> {
  await page.goto('/portal')
  const card = page.locator('.rounded-lg.border.bg-card').filter({
    has: page.locator('h3', { hasText: 'E2E Mixed Exam' }),
  })
  await expect(card).toBeVisible({ timeout: 20_000 })

  const continueBtn = card.getByRole('button', { name: /^продолжить$|^continue$/i })
  const startBtn = card.getByRole('button', { name: /начать экзамен|start exam/i })
  if (await continueBtn.or(startBtn).waitFor({ state: 'visible', timeout: 5_000 }).then(() => true).catch(() => false)) {
    if (await continueBtn.isVisible()) {
      await continueBtn.click()
    } else {
      await startBtn.click()
      const dialog = page.getByRole('dialog')
      await expect(dialog).toBeVisible()
      await dialog.getByRole('button', { name: /начать экзамен|begin exam/i }).click()
    }
  } else {
    // Card shows "View result" (e.g. exam-taking.spec.ts already submitted it): open a fresh
    // session through the API, as exam-taking.spec.ts does.
    const created = await page.evaluate(async () => {
      const tok = localStorage.getItem('__e2e_access_token__')
      const headers = { Authorization: `Bearer ${tok}` }
      const list = await fetch('/api/v1/portal/exams', { headers }).then((r) => r.json())
      const exam = ((list.data ?? []) as Array<{ id: string; title: string }>).find((e) => e.title === 'E2E Mixed Exam')
      if (!exam) return null
      const res = await fetch(`/api/v1/portal/exams/${exam.id}/sessions`, { method: 'POST', headers, credentials: 'include' })
      const json = await res.json()
      return (json.data as { session_id?: string } | null)?.session_id ?? null
    })
    test.skip(!created, 'Cannot open a session for E2E Mixed Exam (attempts exhausted or exam missing)')
    await page.goto(`/portal/sessions/${created}`)
  }
  await expect(page).toHaveURL(/\/portal\/sessions\/[^/]+$/, { timeout: 20_000 })
  // Exam layout is mounted once the header timer/progress is rendered.
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 })
  return page.url()
}

async function stubEvents(page: Page, body: Record<string, unknown>) {
  await page.route(EVENTS_URL, (route: Route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: body, error: null }),
    }),
  )
}

/** Simulate the browser tab becoming hidden (what switching tabs does). */
async function fireTabHidden(page: Page) {
  await page.evaluate(() => {
    Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'hidden' })
    document.dispatchEvent(new Event('visibilitychange'))
  })
}

const WARNING_TITLE = /предупреждение|warning/i

test.describe('Tab-switch events (FR-BB38)', () => {
  test.setTimeout(120_000)

  test('tab hidden reports a tab_switch event and the endpoint contract holds', async ({ page }) => {
    await startMixedExamSession(page)
    const [request, response] = await Promise.all([
      page.waitForRequest((r) => EVENTS_URL.test(r.url()) && r.method() === 'POST'),
      page.waitForResponse((r) => EVENTS_URL.test(r.url())),
      fireTabHidden(page),
    ])
    expect(request.postDataJSON()).toEqual({ type: 'tab_switch' })
    expect(request.headers()['authorization']).toMatch(/^Bearer /)
    expect(response.status()).toBe(200)
    const body = (await response.json()) as { data: { warn: boolean; event_count?: number } }
    expect(typeof body.data.warn).toBe('boolean')
  })

  test('window blur reports a blur event', async ({ page }) => {
    await startMixedExamSession(page)
    await stubEvents(page, { warn: false, event_count: 1 })
    const [request] = await Promise.all([
      page.waitForRequest((r) => EVENTS_URL.test(r.url()) && r.method() === 'POST'),
      page.evaluate(() => window.dispatchEvent(new Event('blur'))),
    ])
    expect(request.postDataJSON()).toEqual({ type: 'blur' })
  })

  test('warn policy shows the warning modal, which can be dismissed', async ({ page }) => {
    await startMixedExamSession(page)
    await stubEvents(page, { warn: true, event_count: 1 })
    await fireTabHidden(page)

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 10_000 })
    await expect(dialog.getByRole('heading', { name: WARNING_TITLE })).toBeVisible()
    await expect(dialog).toContainText(/покинули окно экзамена|left the exam window/i)

    await dialog.getByRole('button', { name: /отмена|cancel/i }).click()
    await expect(dialog).toBeHidden()
    // Still on the exam page, not kicked out.
    await expect(page).toHaveURL(/\/portal\/sessions\/[^/]+$/)
  })

  test('log policy (warn:false) does not interrupt the exam', async ({ page }) => {
    await startMixedExamSession(page)
    await stubEvents(page, { warn: false, event_count: 1 })
    const [response] = await Promise.all([
      page.waitForResponse((r) => EVENTS_URL.test(r.url())),
      fireTabHidden(page),
    ])
    expect(response.status()).toBe(200)
    await expect(page.getByRole('dialog')).toHaveCount(0)
    await expect(page).toHaveURL(/\/portal\/sessions\/[^/]+$/)
  })

  test('auto-submit policy navigates to the session result page', async ({ page }) => {
    await startMixedExamSession(page)
    const sessionId = page.url().split('/').pop() as string
    await stubEvents(page, {
      warn: false,
      event_count: 3,
      session_id: sessionId,
      status: 'auto_submitted',
      score_pct: null,
      passed: null,
    })
    await fireTabHidden(page)
    await expect(page).toHaveURL(new RegExp(`/portal/sessions/${sessionId}/result$`), { timeout: 15_000 })
  })
})
