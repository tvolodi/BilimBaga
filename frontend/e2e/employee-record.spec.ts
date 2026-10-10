/**
 * Employee record page E2E (FR-BB58, issue #16) + certificate download from the record (FR-BB43/44).
 *
 * Requires: make dev running; admin storage state + seeded employee from global-setup.ts.
 * Runs in the chromium-live-admin project.
 */
import { test, expect, type Page } from '@playwright/test'
import { getSeedData } from './fixtures/seed'

const RECORD_API = /\/api\/v1\/admin\/users\/[^/]+\/record(\?.*)?$/
const PROGRESS_API = /\/api\/v1\/admin\/users\/[^/]+\/progress$/
const CERT_API = /\/api\/v1\/admin\/sessions\/[^/]+\/certificate$/

const TITLE = /карточка сотрудника|employee record/i
const HISTORY = /история сессий|session history/i
const PROGRESS = /прогресс по направлениям|track progress/i

let employeeId = ''

test.beforeAll(async () => {
  employeeId = (await getSeedData()).employeeId
})

async function openRecord(page: Page) {
  await page.goto(`/admin/users/${employeeId}/record`)
  await expect(page.getByRole('heading', { level: 1, name: TITLE })).toBeVisible({ timeout: 20_000 })
}

test.describe('Employee record (FR-BB58)', () => {
  test('renders info header, session history and track progress sections', async ({ page }) => {
    await openRecord(page)

    // AC-2: header shows the e-mail as a mailto link.
    const mail = page.getByRole('link', { name: 'employee@bilimbaga.local' })
    await expect(mail).toBeVisible()
    await expect(mail).toHaveAttribute('href', 'mailto:employee@bilimbaga.local')

    // AC-3: history section is a landmark; it holds either the table or the empty message.
    const history = page.getByRole('region', { name: HISTORY })
    await expect(history).toBeVisible()
    const table = history.getByRole('table')
    const empty = history.getByText(/нет записей об экзаменах|no exam sessions recorded/i)
    await expect(table.or(empty)).toBeVisible()
    if (await table.isVisible()) {
      for (const col of [/экзамен|exam/i, /балл|score/i, /статус|status/i, /сертификат|certificate/i]) {
        await expect(table.getByRole('columnheader', { name: col }).first()).toBeVisible()
      }
    }

    // AC-7: three track cards.
    const progress = page.getByRole('region', { name: PROGRESS })
    await expect(progress).toBeVisible()
    for (const track of [/безопасность|security/i, /охрана труда|safety/i, /лояльность|loyalty/i]) {
      await expect(progress.getByRole('heading', { level: 3, name: track })).toBeVisible()
    }
  })

  test('"View Record" link in the users list opens the record page', async ({ page }) => {
    await page.goto('/admin/users')
    const link = page.getByRole('link', { name: /просмотр записи|view record/i }).first()
    await expect(link).toBeVisible({ timeout: 20_000 })
    await expect(link).toHaveAttribute('href', /\/admin\/users\/[0-9a-f-]{36}\/record$/)
    await link.click()
    await expect(page).toHaveURL(/\/admin\/users\/[0-9a-f-]{36}\/record$/)
    await expect(page.getByRole('heading', { level: 1, name: TITLE })).toBeVisible({ timeout: 20_000 })
  })

  test('Export CSV request carries a Bearer token', async ({ page }) => {
    await openRecord(page)
    const exportBtn = page.getByRole('button', { name: /экспорт csv|export csv/i })
    await expect(exportBtn).toBeVisible()
    const [request] = await Promise.all([
      page.waitForRequest((r) => /\/api\/v1\/admin\/users\/[^/]+\/record\/export$/.test(r.url())),
      exportBtn.click(),
    ])
    expect(request.headers()['authorization']).toMatch(/^Bearer /)
  })

  test('shows an error with a working Retry when progress fails to load (AC-11)', async ({ page }) => {
    let failing = true
    await page.route(PROGRESS_API, (route) =>
      failing
        ? route.fulfill({
            status: 500,
            contentType: 'application/json',
            body: JSON.stringify({ data: null, error: { code: 'INTERNAL', message: 'boom' } }),
          })
        : route.continue(),
    )
    await page.goto(`/admin/users/${employeeId}/record`)
    // react-query retries a failed query a few times (exponential backoff) before surfacing the error.
    await expect(page.getByText(/не удалось загрузить карточку|failed to load employee record/i)).toBeVisible({
      timeout: 30_000,
    })
    failing = false
    await page.getByRole('button', { name: /повторить|retry/i }).click()
    await expect(page.getByRole('heading', { level: 1, name: TITLE })).toBeVisible({ timeout: 20_000 })
  })

  test('certificate download appears only for rows with a certificate and sends Bearer (AC-4)', async ({ page }) => {
    const certId = 'CERT-E2E-0001'
    const sessions = [
      {
        session_id: '11111111-1111-4111-8111-111111111111',
        exam_id: '22222222-2222-4222-8222-222222222222',
        exam_title: 'E2E Stub Passed Exam',
        started_at: '2026-01-01T10:00:00Z',
        submitted_at: '2026-01-01T10:20:00Z',
        score_pct: 90,
        passed: true,
        time_taken_seconds: 1200,
        status: 'graded',
        certificate_id: certId,
        exam_category_track: 'security',
      },
      {
        session_id: '33333333-3333-4333-8333-333333333333',
        exam_id: '44444444-4444-4444-8444-444444444444',
        exam_title: 'E2E Stub Failed Exam',
        started_at: '2026-01-02T10:00:00Z',
        submitted_at: '2026-01-02T10:20:00Z',
        score_pct: 20,
        passed: false,
        time_taken_seconds: 600,
        status: 'graded',
        certificate_id: null,
        exam_category_track: 'security',
      },
    ]
    await page.route(RECORD_API, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { sessions }, meta: { page: 1, per_page: 20, total: 2 }, error: null }),
      }),
    )
    await page.route(CERT_API, (route) =>
      route.fulfill({ status: 200, contentType: 'application/pdf', body: '%PDF-1.4\n%%EOF\n' }),
    )

    await openRecord(page)
    const history = page.getByRole('region', { name: HISTORY })
    const passedRow = history.getByRole('row').filter({ hasText: 'E2E Stub Passed Exam' })
    const failedRow = history.getByRole('row').filter({ hasText: 'E2E Stub Failed Exam' })
    await expect(passedRow).toBeVisible()
    await expect(failedRow).toBeVisible()
    await expect(failedRow.getByRole('button')).toHaveCount(0)
    await expect(history.getByText(/показано 1–2 из 2|showing 1–2 of 2/i)).toBeVisible()

    const [request, download] = await Promise.all([
      page.waitForRequest((r) => CERT_API.test(r.url())),
      page.waitForEvent('download'),
      passedRow.getByRole('button', { name: /скачать|download/i }).click(),
    ])
    expect(request.url()).toContain('/sessions/11111111-1111-4111-8111-111111111111/certificate')
    expect(request.headers()['authorization']).toMatch(/^Bearer /)
    expect(download.suggestedFilename()).toBe(`certificate-${certId}.pdf`)
  })

  // Stubs the record (one row with a certificate) and the certificate endpoint with a failure.
  async function stubCertificateFailure(page: Page, status: number, error: { code: string; message: string }) {
    await page.route(RECORD_API, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            sessions: [
              {
                session_id: '55555555-5555-4555-8555-555555555555',
                exam_id: '66666666-6666-4666-8666-666666666666',
                exam_title: 'E2E Stub Cert Error Exam',
                started_at: '2026-01-01T10:00:00Z',
                submitted_at: '2026-01-01T10:20:00Z',
                score_pct: 80,
                passed: true,
                time_taken_seconds: 900,
                status: 'graded',
                certificate_id: 'CERT-E2E-ERR',
                exam_category_track: null,
              },
            ],
          },
          meta: { page: 1, per_page: 20, total: 1 },
          error: null,
        }),
      }),
    )
    await page.route(CERT_API, (route) =>
      route.fulfill({
        status,
        contentType: 'application/json',
        body: JSON.stringify({ data: null, error }),
      }),
    )
  }

  test('certificate download failure shows the exam-not-certifiable message (AC-4)', async ({ page }) => {
    await stubCertificateFailure(page, 422, { code: 'EXAM_NOT_CERTIFIABLE', message: 'not certifiable' })
    await openRecord(page)
    await page.getByRole('button', { name: /скачать|download/i }).click()
    await expect(
      page.getByRole('alert').filter({ hasText: /does not issue certificates|сертификаты не выдаются|сертификат берілмейді/i }),
    ).toBeVisible()
  })

  test('certificate download failure with an unmapped error shows the generic message (AC-4)', async ({ page }) => {
    await stubCertificateFailure(page, 500, { code: 'INTERNAL_ERROR', message: 'boom' })
    await openRecord(page)
    await page.getByRole('button', { name: /скачать|download/i }).click()
    await expect(page.getByRole('alert').filter({ hasText: /download failed|не удалось скачать/i })).toBeVisible()
  })
})
