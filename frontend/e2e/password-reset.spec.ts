/**
 * Forgot / reset password UI E2E (FR-BB115 AC-7, issues #16 and #131).
 *
 * Complements account-recovery.spec.ts, which drives the full live flow (lockout/unlock and
 * forgot -> Mailhog link -> reset -> login). This file only covers what that spec does not, with
 * the API stubbed so no mail server is needed: the login-page entry link, the request contract
 * of the forgot form (no Authorization header), and the invalid / missing token error states.
 * The happy-path reset is NOT repeated here (covered live in account-recovery.spec.ts).
 *
 * Runs unauthenticated (fresh contexts). The shared storage state forces the Russian UI, so each
 * context pins i18n-lang=en before any page script runs (i18n.ts reads it at load).
 */
import { test, expect, type Browser, type Page } from '@playwright/test'

const FORGOT_API = /\/api\/v1\/auth\/forgot-password$/
const RESET_API = /\/api\/v1\/auth\/reset-password$/

async function newPublicPage(browser: Browser) {
  const ctx = await browser.newContext({ storageState: { cookies: [], origins: [] } })
  await ctx.addInitScript(() => {
    try {
      localStorage.setItem('i18n-lang', 'en')
    } catch {
      /* storage unavailable */
    }
  })
  const page = await ctx.newPage()
  return { ctx, page }
}

async function submitReset(page: Page, password: string) {
  await page.getByLabel(/^new password/i).fill(password)
  await page.getByLabel(/confirm new password/i).fill(password)
  await page.getByRole('button', { name: /^reset password$/i }).click()
}

test.describe('Account recovery UI states (FR-BB115)', () => {
  test('login page links to /forgot-password', async ({ browser }) => {
    const { ctx, page } = await newPublicPage(browser)
    try {
      await page.goto('/login')
      await page.getByRole('link', { name: /forgot password\?/i }).click()
      await expect(page).toHaveURL(/\/forgot-password$/)
    } finally {
      await ctx.close()
    }
  })

  test('forgot-password sends only the email, unauthenticated, and shows the neutral confirmation', async ({ browser }) => {
    const { ctx, page } = await newPublicPage(browser)
    try {
      await page.route(FORGOT_API, (route) =>
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: { message: 'ok' }, error: null }),
        }),
      )
      await page.goto('/forgot-password')
      await page.getByLabel(/email/i).fill('nobody@example.com')
      const [request] = await Promise.all([
        page.waitForRequest((r) => FORGOT_API.test(r.url())),
        page.getByRole('button', { name: /send reset link/i }).click(),
      ])
      expect(request.postDataJSON()).toEqual({ email: 'nobody@example.com' })
      expect(request.headers()['authorization']).toBeUndefined()
      await expect(page.getByRole('status')).toContainText(/if an account exists/i)
    } finally {
      await ctx.close()
    }
  })

  test('reset-password with an invalid token shows an error and a link to request a new one', async ({ browser }) => {
    const { ctx, page } = await newPublicPage(browser)
    try {
      await page.route(RESET_API, (route) =>
        route.fulfill({
          status: 400,
          contentType: 'application/json',
          body: JSON.stringify({ data: null, error: { code: 'INVALID_TOKEN', message: 'invalid' } }),
        }),
      )
      await page.goto('/reset-password?token=bad')
      await submitReset(page, 'NewPassw0rd!x')
      await expect(page.getByRole('alert')).toContainText(/invalid or has expired/i)
      await expect(page.getByRole('link', { name: /request a new reset link/i })).toHaveAttribute(
        'href',
        /\/forgot-password$/,
      )
    } finally {
      await ctx.close()
    }
  })

  test('reset-password without a token shows the invalid-link state immediately', async ({ browser }) => {
    const { ctx, page } = await newPublicPage(browser)
    try {
      await page.goto('/reset-password')
      await expect(page.getByRole('alert')).toContainText(/invalid or has expired/i)
      await expect(page.getByRole('link', { name: /request a new reset link/i })).toHaveAttribute(
        'href',
        /\/forgot-password$/,
      )
    } finally {
      await ctx.close()
    }
  })
})
