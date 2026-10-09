import { test, expect } from '@playwright/test'

/**
 * Regression for issue #37: authenticated file downloads must carry the Bearer token.
 * Requires the live stack (make dev; BB_API_PORT / E2E_API_URL as for the other specs).
 * Uses the default admin storage state from the live config.
 */

const API_URL = process.env.E2E_API_URL || `http://localhost:${process.env.BB_API_PORT || 8080}`

test.describe('Authenticated downloads send Authorization (#37)', () => {
  test('dashboard PDF export carries a Bearer token', async ({ page }) => {
    await page.goto('/admin/reports')
    const exportBtn = page.getByRole('button', { name: /export pdf|экспорт pdf|pdf/i }).first()
    await expect(exportBtn).toBeVisible({ timeout: 10_000 })
    const [request] = await Promise.all([
      page.waitForRequest(req => req.url().includes('/api/v1/admin/dashboard/export')),
      exportBtn.click(),
    ])
    expect(request.headers()['authorization']).toMatch(/^Bearer /)
  })

  test('exam results CSV export carries a Bearer token', async ({ page }) => {
    await page.goto('/admin/reports')
    const csvBtn = page.getByRole('button', { name: /csv/i }).first()
    await expect(csvBtn).toBeVisible({ timeout: 10_000 })
    const [request] = await Promise.all([
      page.waitForRequest(req => /\/api\/v1\/admin\/exams\/[^/]+\/results\/export/.test(req.url())),
      csvBtn.click(),
    ])
    expect(request.headers()['authorization']).toMatch(/^Bearer /)
  })

  test('API rejects the export endpoint without a token (sanity check)', async ({ request }) => {
    const res = await request.get(`${API_URL}/api/v1/admin/dashboard/export?from=2026-01-01&to=2026-01-31`)
    expect(res.status()).toBe(401)
  })
})
