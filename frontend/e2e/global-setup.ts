import { chromium } from '@playwright/test'
import path from 'path'
import { fileURLToPath } from 'url'
import fs from 'fs'
import { seedEmployeeFixtures } from './fixtures/seed'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const AUTH_DIR = path.join(__dirname, '..', '.auth')
export const STORAGE_STATE_PATH = path.join(AUTH_DIR, 'admin.json')
const TOKEN_PATH = path.join(AUTH_DIR, 'token.txt')

/** Decode the exp claim from a JWT without verifying the signature. */
function jwtExpiresAt(token: string): number {
  try {
    const payload = JSON.parse(Buffer.from(token.split('.')[1], 'base64url').toString())
    return typeof payload.exp === 'number' ? payload.exp * 1000 : 0
  } catch {
    return 0
  }
}

/** Return a cached access token if it is still valid for at least 5 minutes. */
function cachedToken(): string | null {
  if (!fs.existsSync(TOKEN_PATH)) return null
  const token = fs.readFileSync(TOKEN_PATH, 'utf8').trim()
  if (!token) return null
  const exp = jwtExpiresAt(token)
  if (exp === 0 || exp - Date.now() < 45 * 60 * 1000) return null
  return token
}

/**
 * Global setup: log in once as admin.
 * Saves the access token to .auth/token.txt and reuses it on repeated runs
 * as long as it hasn't expired — this prevents hitting the auth rate limiter
 * when the test suite is run multiple times in quick succession.
 */
export default async function globalSetup() {
  if (!fs.existsSync(AUTH_DIR)) fs.mkdirSync(AUTH_DIR, { recursive: true })

  // ── Token reuse: skip login if the cached token is still fresh ────────────
  let accessToken = cachedToken()
  let storageStateExists = fs.existsSync(STORAGE_STATE_PATH)

  if (accessToken && storageStateExists) {
    console.log('[global-setup] Reusing cached token (expires in',
      Math.round((jwtExpiresAt(accessToken) - Date.now()) / 60_000), 'min)')
  } else {
    // ── Fresh login ───────────────────────────────────────────────────────────
    const browser = await chromium.launch()
    const context = await browser.newContext({ baseURL: 'http://localhost:5173' })
    const page = await context.newPage()

    // Navigate first so relative URLs resolve correctly.
    await page.goto('http://localhost:5173/login')

    // Call the login API directly (1 request per full test run — keeps us under the rate limit).
    const res = await page.evaluate(async ({ email, pass }: { email: string; pass: string }) => {
      const r = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password: pass }),
        credentials: 'include',
      })
      const json = await r.json()
      return { ok: r.ok, data: json.data, error: json.error }
    }, {
      email: process.env.E2E_ADMIN_EMAIL ?? 'admin@bilimbaga.local',
      pass: process.env.E2E_ADMIN_PASS ?? 'Admin1234!',
    })

    if (!res.ok || !res.data?.access_token) {
      await browser.close()
      throw new Error(`Global setup login failed: ${JSON.stringify(res.error)}`)
    }

    accessToken = res.data.access_token as string

    // Seed locale + auth token into localStorage.
    await page.evaluate((token: string) => {
      localStorage.setItem('__e2e_access_token__', token)
      localStorage.setItem('i18n-lang', 'ru')
    }, accessToken)

    const stored = await page.evaluate(() => localStorage.getItem('__e2e_access_token__'))
    console.log('[global-setup] localStorage token set:', stored ? 'YES (length=' + stored.length + ')' : 'NO')

    // Save storage state (captures localStorage entries + refresh cookie).
    await context.storageState({ path: STORAGE_STATE_PATH })
    storageStateExists = true

    const savedState = JSON.parse(fs.readFileSync(STORAGE_STATE_PATH, 'utf8'))
    console.log('[global-setup] Saved origins:', JSON.stringify(savedState.origins?.length ?? 0))

    fs.writeFileSync(TOKEN_PATH, accessToken, 'utf8')
    await browser.close()
  }

  // Seed employee fixtures (creates employee user, exams, sessions).
  await seedEmployeeFixtures(accessToken!)
}
