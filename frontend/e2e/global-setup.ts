import { chromium } from '@playwright/test'
import path from 'path'
import { fileURLToPath } from 'url'
import fs from 'fs'
import { seedEmployeeFixtures } from './fixtures/seed'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const AUTH_DIR = path.join(__dirname, '..', '.auth')
export const STORAGE_STATE_PATH = path.join(AUTH_DIR, 'admin.json')
const TOKEN_PATH = path.join(AUTH_DIR, 'token.txt')

/**
 * Global setup: log in once as admin.
 * Saves the access token to .auth/token.txt so each test can seed it into
 * sessionStorage (via storageState) without needing to call the rate-limited
 * login endpoint again.
 */
export default async function globalSetup() {
  if (!fs.existsSync(AUTH_DIR)) fs.mkdirSync(AUTH_DIR, { recursive: true })

  const browser = await chromium.launch()
  const context = await browser.newContext({ baseURL: 'http://localhost:5173' })
  const page = await context.newPage()

  // Navigate first so relative URLs resolve correctly.
  await page.goto('http://localhost:5173/login')

  // Call the login API directly (bypasses rate limit — only 1 call in global setup).
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

  const accessToken = res.data.access_token as string

  // Seed the access token into localStorage so each test page can read it.
  // sessionStorage is NOT reliably captured by Playwright's storageState — use localStorage.
  await page.evaluate((token: string) => {
    localStorage.setItem('__e2e_access_token__', token)
  }, accessToken)

  // Verify localStorage was set.
  const stored = await page.evaluate(() => localStorage.getItem('__e2e_access_token__'))
  console.log('[global-setup] localStorage token set:', stored ? 'YES (length=' + stored.length + ')' : 'NO')

  // Save storage state (includes the sessionStorage entry + refresh cookie).
  await context.storageState({ path: STORAGE_STATE_PATH })

  // Verify the saved state.
  const savedState = JSON.parse(fs.readFileSync(STORAGE_STATE_PATH, 'utf8'))
  console.log('[global-setup] Saved origins:', JSON.stringify(savedState.origins?.length ?? 0))

  // Also save the raw token so tests can re-seed if needed.
  fs.writeFileSync(TOKEN_PATH, accessToken, 'utf8')

  await browser.close()

  // Seed employee fixtures (creates employee user, exams, sessions).
  await seedEmployeeFixtures(accessToken)
}
