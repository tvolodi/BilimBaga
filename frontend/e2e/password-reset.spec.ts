/**
 * Forgot / reset password E2E (FR-BB110 follow-up, FR-BB115 AC-7, issue #16).
 *
 * STATUS: all tests are test.fixme. The self-service recovery flow (issue #33, FR-BB115) is not
 * on main yet: App.tsx has no /forgot-password or /reset-password route, the login page has no
 * "Forgot password?" link, and src/api/recovery.ts does not exist. The selectors below follow
 * the FR-BB115 spec text and are UNCONFIRMED against real components; when #33 merges, remove
 * the fixme markers and align names/labels with the shipped i18n keys (auth.recovery.*).
 *
 * The admin-initiated temporary-password reset that does exist today is covered by
 * user-management.spec.ts ("User Management - Reset Password").
 *
 * Runs unauthenticated (fresh contexts); the API is stubbed so no mail server is needed.
 */
import { test, expect } from '@playwright/test'

const FORGOT_API = /\/api\/v1\/auth\/forgot-password$/
const RESET_API = /\/api\/v1\/auth\/reset-password$/

test.describe('Account recovery (FR-BB115) - blocked on #33', () => {
  test.fixme(true, 'FR-BB115 / issue #33 not merged: no /forgot-password, /reset-password routes on main')

  test('login page links to /forgot-password', async ({ browser }) => {
    const ctx = await browser.newContext({ storageState: { cookies: [], origins: [] } })
    const page = await ctx.newPage()
    await page.goto('/login')
    await page.getByRole('link', { name: /забыли пароль|forgot password/i }).click()
    await expect(page).toHaveURL(/\/forgot-password$/)
    await ctx.close()
  })

  test('forgot-password shows the same neutral confirmation for any email', async ({ browser }) => {
    const ctx = await browser.newContext({ storageState: { cookies: [], origins: [] } })
    const page = await ctx.newPage()
    await page.route(FORGOT_API, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { message: 'ok' }, error: null }),
      }),
    )
    await page.goto('/forgot-password')
    await page.getByRole('textbox', { name: /email/i }).fill('nobody@example.com')
    const [request] = await Promise.all([
      page.waitForRequest((r) => FORGOT_API.test(r.url())),
      page.getByRole('button', { name: /отправить|send|submit/i }).click(),
    ])
    expect(request.postDataJSON()).toEqual({ email: 'nobody@example.com' })
    expect(request.headers()['authorization']).toBeUndefined()
    await expect(page.getByRole('status').or(page.getByRole('alert'))).toBeVisible()
    await ctx.close()
  })

  test('reset-password with a valid token submits and redirects to /login', async ({ browser }) => {
    const ctx = await browser.newContext({ storageState: { cookies: [], origins: [] } })
    const page = await ctx.newPage()
    await page.route(RESET_API, (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { message: 'ok' }, error: null }),
      }),
    )
    await page.goto('/reset-password?token=e2e-token')
    const fields = page.locator('input[type="password"]')
    await fields.nth(0).fill('NewPassw0rd!x')
    await fields.nth(1).fill('NewPassw0rd!x')
    const [request] = await Promise.all([
      page.waitForRequest((r) => RESET_API.test(r.url())),
      page.getByRole('button', { name: /сменить|сохранить|reset|save|submit/i }).click(),
    ])
    expect(request.postDataJSON()).toMatchObject({ token: 'e2e-token', new_password: 'NewPassw0rd!x' })
    await expect(page).toHaveURL(/\/login/)
    await ctx.close()
  })

  test('reset-password with an invalid token shows an error and a link back to /forgot-password', async ({ browser }) => {
    const ctx = await browser.newContext({ storageState: { cookies: [], origins: [] } })
    const page = await ctx.newPage()
    await page.route(RESET_API, (route) =>
      route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({ data: null, error: { code: 'INVALID_TOKEN', message: 'invalid' } }),
      }),
    )
    await page.goto('/reset-password?token=bad')
    const fields = page.locator('input[type="password"]')
    await fields.nth(0).fill('NewPassw0rd!x')
    await fields.nth(1).fill('NewPassw0rd!x')
    await page.getByRole('button', { name: /сменить|сохранить|reset|save|submit/i }).click()
    await expect(page.getByRole('alert')).toBeVisible()
    await expect(page.getByRole('link', { name: /forgot|забыли/i })).toHaveAttribute('href', /\/forgot-password/)
    await ctx.close()
  })
})
