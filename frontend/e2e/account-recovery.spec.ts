/**
 * FR-BB115 live E2E: lockout -> admin unlock -> login, and forgot -> reset -> login.
 *
 * Needs the Docker stack: the api container sends mail to Mailhog (SMTP_HOST=mailhog), and the
 * reset link is read back from the Mailhog HTTP API (HOST_MAILHOG_UI_PORT, default 8025).
 */
import { test, expect } from '@playwright/test'
import { getSeedData, createTestUser, deleteTestUser } from './fixtures/seed'

const API = process.env.E2E_API_URL || `http://localhost:${process.env.BB_API_PORT || 8080}`
const MAILHOG = process.env.E2E_MAILHOG_URL || `http://localhost:${process.env.HOST_MAILHOG_UI_PORT || 8025}`
const NEW_PASSWORD = 'Recovered1234!'

async function postJson(path: string, body: unknown, token?: string) {
  const res = await fetch(`${API}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const json = (await res.json().catch(() => ({}))) as { data?: any; error?: { code: string } | null }
  return { status: res.status, data: json.data, error: json.error ?? null }
}

/** Decode a quoted-printable body (the SMTP client encodes both parts). */
function decodeQuotedPrintable(s: string): string {
  return s
    .replace(/=\r?\n/g, '')
    .replace(/=([0-9A-F]{2})/g, (_m, h: string) => String.fromCharCode(parseInt(h, 16)))
}

/** Poll Mailhog until a message to `to` arrives and return the reset token from its link. */
async function readResetTokenFromMailhog(to: string): Promise<string> {
  const deadline = Date.now() + 20_000
  while (Date.now() < deadline) {
    const res = await fetch(`${MAILHOG}/api/v2/search?kind=to&query=${encodeURIComponent(to)}`)
    if (res.ok) {
      const json = (await res.json()) as { items?: Array<{ Content: { Body: string } }> }
      for (const item of json.items ?? []) {
        const match = decodeQuotedPrintable(item.Content.Body).match(/reset-password\?token=([A-Za-z0-9_-]+)/)
        if (match) return match[1]
      }
    }
    await new Promise((r) => setTimeout(r, 500))
  }
  throw new Error(`no password reset email for ${to} found in Mailhog at ${MAILHOG}`)
}

test.describe('Account recovery (FR-BB115)', () => {
  test('lock -> admin unlock -> login succeeds without waiting for expiry', async () => {
    const { adminToken } = await getSeedData()
    const user = await createTestUser(adminToken, 'lockout')
    try {
      for (let i = 0; i < 5; i++) {
        await postJson('/api/v1/auth/login', { email: user.email, password: 'Wrong-password-1' })
      }
      const locked = await postJson('/api/v1/auth/login', { email: user.email, password: user.password })
      expect(locked.status).toBe(423)
      expect(locked.error?.code).toBe('ACCOUNT_LOCKED')

      const unlocked = await postJson(`/api/v1/users/${user.id}/unlock`, undefined, adminToken)
      expect(unlocked.status).toBe(200)
      expect(unlocked.data.is_locked).toBe(false)

      const ok = await postJson('/api/v1/auth/login', { email: user.email, password: user.password })
      expect(ok.status).toBe(200)

      const missing = await postJson('/api/v1/users/00000000-0000-4000-8000-000000000000/unlock', undefined, adminToken)
      expect(missing.status).toBe(404)
    } finally {
      await deleteTestUser(adminToken, user.id)
    }
  })

  test('forgot password -> email link -> reset -> login with the new password', async ({ page }) => {
    const { adminToken } = await getSeedData()
    const user = await createTestUser(adminToken, 'recovery')
    try {
      // The shared storage state (global-setup/seed) forces the Russian UI; these selectors are English.
      // i18n.ts reads localStorage 'i18n-lang' at load, so set it before any page script runs.
      await page.addInitScript(() => {
        try {
          localStorage.setItem('i18n-lang', 'en')
        } catch {
          /* storage unavailable */
        }
      })
      await page.goto('/forgot-password')
      await page.getByLabel(/email/i).fill(user.email)
      await page.getByRole('button', { name: /send reset link/i }).click()
      await expect(page.getByRole('status')).toContainText(/if an account exists/i)

      const token = await readResetTokenFromMailhog(user.email)

      // A weak password is rejected and does not consume the token.
      await page.goto(`/reset-password?token=${token}`)
      await page.getByLabel(/^new password/i).fill('weak')
      await page.getByLabel(/confirm new password/i).fill('weak')
      await page.getByRole('button', { name: /^reset password$/i }).click()
      await expect(page.getByRole('alert')).toContainText(/does not meet requirements/i)

      await page.getByLabel(/^new password/i).fill(NEW_PASSWORD)
      await page.getByLabel(/confirm new password/i).fill(NEW_PASSWORD)
      await page.getByRole('button', { name: /^reset password$/i }).click()
      await expect(page).toHaveURL(/\/login/)
      await expect(page.getByRole('status')).toContainText(/password has been reset/i)

      const ok = await postJson('/api/v1/auth/login', { email: user.email, password: NEW_PASSWORD })
      expect(ok.status).toBe(200)
      expect(ok.data.user.force_password_change).toBe(false)
      const old = await postJson('/api/v1/auth/login', { email: user.email, password: user.password })
      expect(old.status).toBe(401)

      // Single use: replaying the link is rejected.
      const replay = await postJson('/api/v1/auth/reset-password', { token, new_password: 'Another1234!' })
      expect(replay.status).toBe(400)
      expect(replay.error?.code).toBe('INVALID_TOKEN')
    } finally {
      await deleteTestUser(adminToken, user.id)
    }
  })

  test('forgot-password answers identically for an unknown email', async () => {
    const known = await postJson('/api/v1/auth/forgot-password', { email: 'admin@bilimbaga.local' })
    const unknown = await postJson('/api/v1/auth/forgot-password', { email: `nobody-${Date.now()}@e2e-test.local` })
    expect(unknown.status).toBe(known.status)
    expect(unknown.data).toEqual(known.data)
  })
})
