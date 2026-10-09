/**
 * Certificate verification E2E (FR-BB43 / FR-BB44 / FR-BB48, issue #16).
 *
 * Covers the public /verify/:code page (the target of the certificate QR code).
 * Download of the certificate PDF from the employee record is covered in
 * employee-record.spec.ts.
 *
 * Requires: make dev running (only for the "unknown code" test, which hits the real API).
 * Runs in the chromium-live-admin project; the page itself is public (rendered outside the
 * auth bootstrap), which the first test asserts.
 */
import { test, expect } from '@playwright/test'

const VERIFY_API = /\/api\/v1\/verify\/[^/]+$/

const VALID_PAYLOAD = {
  valid: true,
  employee_name: 'E2E Certificate Holder',
  exam_title: 'E2E Certified Exam',
  score_pct: 87.5,
  issued_at: '2026-03-15T12:00:00Z',
}

test.describe('Certificate verification page (FR-BB48)', () => {
  test('unknown / malformed code renders "not found or invalid" (valid:false, ISS-092)', async ({ page }) => {
    await page.goto('/verify/not-a-real-code')
    await expect(page.getByRole('heading', { level: 1, name: /проверка сертификата|certificate verification/i })).toBeVisible({
      timeout: 20_000,
    })
    await expect(page.getByRole('main').getByRole('alert')).toContainText(/не найден или недействителен|not found or invalid/i)
    await expect(page.getByRole('status')).toHaveCount(0)
  })

  test('is public: sends no Authorization header and no token refresh', async ({ page }) => {
    const apiRequests: Array<{ url: string; auth: string | undefined }> = []
    page.on('request', (r) => {
      if (r.url().includes('/api/v1/')) apiRequests.push({ url: r.url(), auth: r.headers()['authorization'] })
    })
    await page.route(VERIFY_API, (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: VALID_PAYLOAD, error: null }) }),
    )
    await page.goto('/verify/E2E-PUBLIC-CODE')
    await expect(page.getByRole('status')).toBeVisible({ timeout: 20_000 })

    expect(apiRequests.some((r) => /\/api\/v1\/verify\/E2E-PUBLIC-CODE$/.test(r.url))).toBe(true)
    expect(apiRequests.filter((r) => r.auth)).toEqual([])
    expect(apiRequests.some((r) => r.url.includes('/auth/refresh'))).toBe(false)
  })

  test('valid certificate shows holder, exam, score and issue date', async ({ page }) => {
    await page.route(VERIFY_API, (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: VALID_PAYLOAD, error: null }) }),
    )
    await page.goto('/verify/E2E-VALID-CODE')

    const status = page.getByRole('status')
    await expect(status).toContainText(/сертификат действителен|certificate is valid/i, { timeout: 20_000 })
    const main = page.getByRole('main')
    await expect(main.getByText('E2E Certificate Holder')).toBeVisible()
    await expect(main.getByText('E2E Certified Exam')).toBeVisible()
    await expect(main.getByText('87.5%')).toBeVisible()
    // Issue date is formatted per locale (UTC); only the year is locale independent.
    await expect(main.getByText(/2026/)).toBeVisible()
  })

  test('adds a robots noindex meta tag while mounted (AC-7)', async ({ page }) => {
    await page.route(VERIFY_API, (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: { valid: false }, error: null }) }),
    )
    await page.goto('/verify/E2E-NOINDEX')
    await expect(page.locator('meta[name="robots"][content="noindex"]')).toHaveCount(1)
  })

  test('transient failure shows "unavailable" and Retry recovers', async ({ page }) => {
    let failing = true
    await page.route(VERIFY_API, (route) =>
      failing
        ? route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ data: null, error: { code: 'INTERNAL', message: 'boom' } }) })
        : route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: VALID_PAYLOAD, error: null }) }),
    )
    await page.goto('/verify/E2E-RETRY')
    await expect(page.getByText(/проверка временно недоступна|verification temporarily unavailable/i)).toBeVisible({
      timeout: 20_000,
    })
    failing = false
    await page.getByRole('button', { name: /повторить|retry/i }).click()
    await expect(page.getByRole('status')).toContainText(/сертификат действителен|certificate is valid/i)
  })

  test('locale switcher on the verification page translates the result', async ({ page }) => {
    await page.route(VERIFY_API, (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: VALID_PAYLOAD, error: null }) }),
    )
    await page.goto('/verify/E2E-LOCALE')
    await expect(page.getByRole('status')).toBeVisible({ timeout: 20_000 })
    const select = page.locator('select').filter({ has: page.locator('option[value="kk"]') })
    await select.selectOption('en')
    await expect(page.getByRole('status')).toContainText(/certificate is valid/i)
    await select.selectOption('ru')
    await expect(page.getByRole('status')).toContainText(/сертификат действителен/i)
  })

  // seed.ts creates no exam that is certificate-enabled with a passed employee session (the
  // seeded exams are not configured for certificates), so the real issue -> PDF -> verify round
  // trip cannot be driven without extending the seed. Tracked as a follow-up in the PR description.
  test.fixme('passed certifiable exam: result page offers a PDF whose code verifies (FR-BB43/44)', async () => {
    // Blocked: needs a seeded exam with certificates enabled + a passed employee session.
  })
})
