/**
 * Tab-switch / anti-cheat events E2E (FR-BB38, FR-BB319, issue #16).
 *
 * Requires: make dev running, employee storage state seeded by global-setup.ts. Runs in the
 * chromium-live-employee project. Creates its own exams per run (see below).
 *
 * The client reports `tab_switch` (document visibilitychange -> hidden) and `blur` (window blur,
 * after a 300 ms window) to POST /api/v1/portal/sessions/:id/events (see useTabSwitchDetection).
 * The server-side policy (log / warn / auto-submit) is exercised by stubbing the response
 * so the UI reaction is deterministic regardless of the exam's on_tab_switch policy;
 * the FR-BB319 tests use a real exam with on_tab_switch = 'submit' and check the event type
 * on the client request (the type is not stored, so the database is checked by the backend tests).
 */
import { test, expect, type Page, type Route, type Request as PwRequest } from '@playwright/test'
import {
  getSeedData,
  getEmployeeApiToken,
  createTestQuestion,
  createTestExam,
  deleteTestExam,
  deleteTestQuestion,
  startEmployeeSession,
  type TestExam,
} from './fixtures/seed'

const EVENTS_URL = /\/api\/v1\/portal\/sessions\/[^/]+\/events$/

// Dedicated exams are created per run (max_attempts 500, assigned to the seed employee) so the
// shared "E2E Mixed Exam" attempt counter, which repeated suite runs exhaust, cannot make these
// tests disappear. Session creation failures FAIL the test; they never skip it.
let adminToken = ''
let exam: TestExam
let submitExam: TestExam
let questionId = ''

/** Sessions started by the FR-BB319 tests on the submit exam; submitted again after each test. */
const submitExamSessions: string[] = []

test.beforeAll(async () => {
  const seed = await getSeedData()
  adminToken = seed.adminToken
  const question = await createTestQuestion(adminToken, `E2E TabSwitch Q ${Date.now()}`, 'single', true)
  questionId = question.id
  exam = await createTestExam(adminToken, `E2E TabSwitch Exam ${Date.now()}`, question.id, {
    maxAttempts: 500,
    onTabSwitch: 'log',
    assignToUserId: seed.employeeId,
  })
  submitExam = await createTestExam(adminToken, `E2E TabSwitch Submit Exam ${Date.now()}`, question.id, {
    maxAttempts: 500,
    onTabSwitch: 'submit',
    assignToUserId: seed.employeeId,
  })
})

test.afterAll(async () => {
  // Best effort: an exam with sessions may be undeletable; the title is unique per run.
  if (submitExam) await deleteTestExam(adminToken, submitExam.id).catch(() => undefined)
  if (exam) await deleteTestExam(adminToken, exam.id).catch(() => undefined)
  if (questionId) await deleteTestQuestion(adminToken, questionId).catch(() => undefined)
})

test.afterEach(async ({ request }) => {
  // Close the sessions this test opened on the submit exam, so the next test starts a fresh session
  // (startEmployeeSession reuses an open one). Errors are ignored: a session may already be submitted.
  if (submitExamSessions.length === 0) return
  const token = await getEmployeeApiToken()
  for (const id of submitExamSessions.splice(0)) {
    await request
      .post(`/api/v1/portal/sessions/${id}/submit`, { headers: { Authorization: `Bearer ${token}` } })
      .catch(() => undefined)
  }
})

/** Open (or resume) a session on the given exam through the API and load the exam page. */
async function startExamSession(page: Page, examId: string = exam.id): Promise<string> {
  const sessionId = await startEmployeeSession(examId)
  await page.goto(`/portal/sessions/${sessionId}`)
  await expect(page).toHaveURL(/\/portal\/sessions\/[^/]+$/, { timeout: 20_000 })
  // Exam layout is mounted once the header timer/progress is rendered.
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 })
  return page.url()
}

/** A fresh session on the submit exam (on_tab_switch = 'submit'), closed again after the test. */
async function startFreshSubmitSession(page: Page): Promise<string> {
  const sessionId = await startEmployeeSession(submitExam.id)
  submitExamSessions.push(sessionId)
  await page.goto(`/portal/sessions/${sessionId}`)
  await expect(page).toHaveURL(/\/portal\/sessions\/[^/]+$/, { timeout: 20_000 })
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 })
  return sessionId
}

/** Collect every POST to the events URL the page sends, in order. */
function trackEventRequests(page: Page): PwRequest[] {
  const sent: PwRequest[] = []
  page.on('request', (r) => {
    if (EVENTS_URL.test(r.url()) && r.method() === 'POST') sent.push(r)
  })
  return sent
}

function eventTypes(requests: PwRequest[]): unknown[] {
  return requests.map((r) => (r.postDataJSON() as { type: unknown }).type)
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
    await startExamSession(page)
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
    await startExamSession(page)
    await stubEvents(page, { warn: false, event_count: 1 })
    const [request] = await Promise.all([
      page.waitForRequest((r) => EVENTS_URL.test(r.url()) && r.method() === 'POST'),
      page.evaluate(() => window.dispatchEvent(new Event('blur'))),
    ])
    expect(request.postDataJSON()).toEqual({ type: 'blur' })
  })

  test('warn policy shows the warning modal, which can be dismissed', async ({ page }) => {
    await startExamSession(page)
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
    await startExamSession(page)
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
    await startExamSession(page)
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

test.describe('Anti-cheat event hygiene on an on_tab_switch = submit exam (FR-BB319)', () => {
  test.setTimeout(120_000)

  test('(a) a simulated tab switch under submit sends exactly one tab_switch and auto-submits', async ({ page }) => {
    const sessionId = await startFreshSubmitSession(page)
    const sent = trackEventRequests(page)

    await page.evaluate(() => {
      Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'hidden' })
      document.dispatchEvent(new Event('visibilitychange'))
      window.dispatchEvent(new Event('blur'))
    })

    await expect(page).toHaveURL(new RegExp(`/portal/sessions/${sessionId}/result$`), { timeout: 15_000 })
    // Wait past the 300 ms blur window: the blur that accompanies the tab switch must not be sent.
    await page.waitForTimeout(500)
    expect(eventTypes(sent)).toEqual(['tab_switch'])
  })

  test('(b) a bare blur under submit sends one blur, warns with event_count 1 and does not submit', async ({ page }) => {
    const sessionId = await startFreshSubmitSession(page)
    const sent = trackEventRequests(page)

    const [request, response] = await Promise.all([
      page.waitForRequest((r) => EVENTS_URL.test(r.url()) && r.method() === 'POST'),
      page.waitForResponse((r) => EVENTS_URL.test(r.url())),
      page.evaluate(() => window.dispatchEvent(new Event('blur'))),
    ])
    expect(request.postDataJSON()).toEqual({ type: 'blur' })
    expect(response.status()).toBe(200)
    const body = (await response.json()) as { data: Record<string, unknown> }
    expect(body.data).toMatchObject({ warn: true, event_count: 1 })
    expect(body.data).not.toHaveProperty('status')

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 10_000 })
    await expect(dialog).toContainText(/violations recorded: 1|зафиксировано нарушений: 1|тіркелген бұзушылықтар: 1/i)
    await expect(page).toHaveURL(new RegExp(`/portal/sessions/${sessionId}$`))

    // Nothing else is sent after the warning.
    await page.waitForTimeout(500)
    expect(eventTypes(sent)).toEqual(['blur'])
  })
})
