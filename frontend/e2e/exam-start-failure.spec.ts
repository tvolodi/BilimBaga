/**
 * ISS-132 / GitHub #132 — "Start button is pressed but the exam does not start".
 *
 * A refused start must be visible to the employee (translated alert inside the modal) and a
 * successful start must navigate to the session page. The POST is intercepted so the failure
 * modes (outside window, gateway error) are deterministic; the portal list itself comes from
 * the real seeded stack.
 *
 * Requires: make dev running, employee storage state seeded by global-setup.ts
 */

import { test, expect, type Page } from '@playwright/test'

const START_ROUTE = '**/api/v1/portal/exams/*/sessions'
const START_BTN = /начать экзамен|start exam/i
const CONFIRM_BTN = /начать экзамен|begin exam/i

async function openStartModal(page: Page): Promise<boolean> {
  await page.goto('/portal')
  await page.waitForLoadState('networkidle').catch(() => {})
  const startBtn = page.getByRole('button', { name: START_BTN }).first()
  if (!(await startBtn.isVisible({ timeout: 10_000 }).catch(() => false))) return false
  await startBtn.click()
  await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
  return true
}

test.describe('Start exam failure visibility (ISS-132)', () => {
  test.setTimeout(60_000)

  test('refused start (422 EXAM_OUTSIDE_WINDOW) shows an alert and keeps the modal open', async ({ page }) => {
    await page.route(START_ROUTE, (route) =>
      route.request().method() === 'POST'
        ? route.fulfill({
            status: 422,
            contentType: 'application/json',
            body: JSON.stringify({ data: null, error: { code: 'EXAM_OUTSIDE_WINDOW', message: 'x' } }),
          })
        : route.continue(),
    )
    if (!(await openStartModal(page))) {
      test.info().annotations.push({ type: 'note', description: 'No not_started exam available' })
      return
    }
    await page.getByRole('dialog').getByRole('button', { name: CONFIRM_BTN }).click()
    await expect(page.getByRole('dialog').getByRole('alert')).toBeVisible({ timeout: 5_000 })
    await expect(page).toHaveURL(/\/portal$/)
  })

  test('gateway failure (502, non-JSON) shows the generic alert', async ({ page }) => {
    await page.route(START_ROUTE, (route) =>
      route.request().method() === 'POST'
        ? route.fulfill({ status: 502, contentType: 'text/html', body: '<html>Bad Gateway</html>' })
        : route.continue(),
    )
    if (!(await openStartModal(page))) return
    await page.getByRole('dialog').getByRole('button', { name: CONFIRM_BTN }).click()
    await expect(page.getByRole('dialog').getByRole('alert')).toBeVisible({ timeout: 5_000 })
  })

  test('successful start navigates to the session page', async ({ page }) => {
    // No interception: exercises the real backend, which must return data.session_id.
    if (!(await openStartModal(page))) return
    await page.getByRole('dialog').getByRole('button', { name: CONFIRM_BTN }).click()
    await expect(page).toHaveURL(/\/portal\/sessions\/[0-9a-f-]{36}/, { timeout: 20_000 })
  })

  test('Start is disabled with a reason when the availability window is closed', async ({ page }) => {
    await page.route('**/api/v1/portal/exams', async (route) => {
      if (route.request().method() !== 'GET') return route.continue()
      const res = await route.fetch()
      const body = await res.json()
      const data = (body.data ?? []).map((e: Record<string, unknown>) => ({
        ...e,
        user_status: 'not_started',
        open_session_id: null,
        available_until: '2020-01-01T00:00:00Z',
      }))
      return route.fulfill({ response: res, json: { ...body, data } })
    })
    await page.goto('/portal')
    const startBtn = page.getByRole('button', { name: START_BTN }).first()
    if (!(await startBtn.isVisible({ timeout: 10_000 }).catch(() => false))) return
    await expect(startBtn).toBeDisabled()
    await expect(page.getByRole('status').first()).toBeVisible()
  })
})
