/**
 * TOKEN_REVOKED session end, real browser (ISS-249, FR-BB110 AC-11). Two browser contexts.
 *
 * Requires: the live stack and global setup (run by playwright.live.config.ts, project
 * chromium-live-admin). Context A is the signed-in user under test. Context B changes that user's
 * account through the API, which revokes A's access token:
 *  - sequence (a): password changed (ISS-171). A's next protected call gets 401 TOKEN_REVOKED, the
 *    refresh gets 401, and the session ends: URL /login, a dismissible role="alert" notice, no
 *    further protected request, and the requested path is restored after signing in again.
 *  - sequence (b): department changed (ISS-240 claims). A's call gets 401 TOKEN_REVOKED, the refresh
 *    gets 200 with the new claims, the retry gets 200, and the user stays on the same page.
 *
 * The request is made by a sidebar click (or a portal tab click) after the revocation, so it uses
 * the token held in memory, the path AC-11 describes. A full reload would run the boot refresh path.
 *
 * Every context starts with empty storage state, so no seeded token from the project's storageState
 * is in play. Sign-in checks that the token the page received belongs to the account under test.
 */
import { test, expect, type Browser, type Locator, type Page } from '@playwright/test'
import { requireTarget } from '../../scripts/lib/target-guard'
import { loginClearingForceChange } from '../../scripts/lib/e2e-auth'
import { getSeedData, deleteTestUser } from './fixtures/seed'

const BASE = requireTarget(
  'E2E_API_URL',
  process.env.E2E_API_URL || `http://localhost:${process.env.BB_API_PORT || 8080}`,
  process.env,
)

const PASS_FIRST = 'E2eRevoke2024!'
const PASS_SECOND = 'E2eRevoke2025!'
/** Login-page notice (auth.login.sessionRevoked) in kk, ru and en. */
const NOTICE_TEXT = /Қайта кіріңіз|Войдите снова|sign in again/i
/** Dismiss control inside the notice (auth.login.dismissNotice) in kk, ru and en. */
const DISMISS_NAME = /Жабу|Закрыть|Dismiss/i
/**
 * Quiet period after the refresh 401 that covers React Query's default retry back-off (1s, 2s, 4s):
 * a query that kept retrying would send its requests inside this window.
 */
const QUIET_MS = 8_000
/**
 * A token issued in the same second as a password change is not treated as revoked, so context B
 * waits past the second boundary before each change.
 */
const SETTLE_MS = 1_100

let adminToken = ''
const accounts: Account[] = []
const departmentIds: string[] = []

interface ApiResult<T> {
  status: number
  data: T | null
  error: { code?: string; message?: string } | null
}

interface Account {
  id: string
  email: string
  fullName: string
  password: string
  token: string
}

interface Hit {
  method: string
  path: string
  status: number
}

test.beforeAll(async () => {
  adminToken = (await getSeedData()).adminToken
})

test.afterAll(async () => {
  for (const account of accounts) {
    await deleteTestUser(adminToken, account.id).catch(() => undefined)
  }
  for (const id of departmentIds) {
    await fetch(`${BASE}/api/v1/departments/${id}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${adminToken}` },
    }).catch(() => undefined)
  }
})

test.describe('TOKEN_REVOKED ends the session in a real browser', () => {
  test.setTimeout(150_000)

  test('(a) admin: password changed elsewhere ends the session at /login and restores the requested path', async ({ browser }) => {
    const admin = await createAccount('super_admin', 'e2e-revoke-admin')
    const ctxA = await isolatedContext(browser)
    const page = await ctxA.newPage()
    const hits = trackApi(page)
    try {
      await signIn(page, admin)
      await expect(page).toHaveURL(/\/admin/, { timeout: 20_000 })
      await page.goto('/admin/users')
      await expect(sidebar(page)).toBeVisible({ timeout: 20_000 })

      await settle()
      await changePasswordInContextB(browser, admin, PASS_SECOND)

      // Probe: a new query on a guarded admin route, made with the revoked token.
      await sidebarLink(page, '/admin/departments').click()

      await assertSessionEnded(page, hits)
      await page.getByRole('alert').getByRole('button', { name: DISMISS_NAME }).click()
      await expect(page.getByRole('alert')).toHaveCount(0)

      await submitLogin(page, admin.email, admin.password)
      await expect(page).toHaveURL(/\/admin\/departments$/, { timeout: 20_000 })
    } finally {
      await ctxA.close()
    }
  })

  test('(a) portal: password changed elsewhere ends the session at /login and restores the requested path', async ({ browser }) => {
    const employee = await createAccount('employee', 'e2e-revoke-emp')
    const ctxA = await isolatedContext(browser)
    const page = await ctxA.newPage()
    const hits = trackApi(page)
    try {
      await signIn(page, employee)
      await expect(page).toHaveURL(/\/portal/, { timeout: 20_000 })
      await page.goto('/portal')
      await expect(page).toHaveURL(/\/portal/, { timeout: 20_000 })

      await settle()
      await changePasswordInContextB(browser, employee, PASS_SECOND)

      // Probe: the results tab queries with the revoked token.
      await page.locator('a[href="/portal/results"]').first().click()

      await assertSessionEnded(page, hits)
      await page.getByRole('alert').getByRole('button', { name: DISMISS_NAME }).click()
      await expect(page.getByRole('alert')).toHaveCount(0)

      await submitLogin(page, employee.email, employee.password)
      await expect(page).toHaveURL(/\/portal\/results$/, { timeout: 20_000 })
    } finally {
      await ctxA.close()
    }
  })

  // A migrated module (#249 attempt 4): the admin dashboard queries through the shared apiFetch, which
  // before that change was a local copy with no revocation handling.
  test('(a) admin dashboard: password changed elsewhere ends the session from a migrated module', async ({ browser }) => {
    const admin = await createAccount('super_admin', 'e2e-revoke-dash')
    const ctxA = await isolatedContext(browser)
    const page = await ctxA.newPage()
    const hits = trackApi(page)
    try {
      await signIn(page, admin)
      await expect(page).toHaveURL(/\/admin/, { timeout: 20_000 })
      await page.goto('/admin/users')
      await expect(sidebar(page)).toBeVisible({ timeout: 20_000 })

      await settle()
      await changePasswordInContextB(browser, admin, PASS_SECOND)

      // Probe: the sidebar's dashboard link redirects to /admin/dashboard, which queries with the revoked token.
      await sidebarLink(page, '/admin').click()

      await assertSessionEnded(page, hits)
      await page.getByRole('alert').getByRole('button', { name: DISMISS_NAME }).click()
      await expect(page.getByRole('alert')).toHaveCount(0)

      await submitLogin(page, admin.email, admin.password)
      await expect(page).toHaveURL(/\/admin\/dashboard$/, { timeout: 20_000 })
    } finally {
      await ctxA.close()
    }
  })

  test('(b) admin: department changed elsewhere refreshes once and the user stays signed in', async ({ browser }) => {
    const admin = await createAccount('super_admin', 'e2e-revoke-claims')
    const department = await createDepartment()
    const ctxA = await isolatedContext(browser)
    const page = await ctxA.newPage()
    const hits = trackApi(page)
    try {
      await signIn(page, admin)
      await expect(page).toHaveURL(/\/admin/, { timeout: 20_000 })
      await page.goto('/admin/users')
      await expect(sidebar(page)).toBeVisible({ timeout: 20_000 })

      await settle()
      // Change the account's department in context B (the JWT department claim no longer matches).
      const ctxB = await isolatedContext(browser)
      try {
        const roleId = await roleIdByName('super_admin')
        const res = await ctxB.request.put(`${BASE}/api/v1/users/${admin.id}`, {
          headers: { Authorization: `Bearer ${adminToken}` },
          data: { full_name: admin.fullName, role_id: roleId, department_id: department },
        })
        expect(res.status(), 'department change in context B').toBe(200)
      } finally {
        await ctxB.close()
      }

      await sidebarLink(page, '/admin/departments').click()

      // 401 TOKEN_REVOKED, then one refresh that succeeds, then the retry that succeeds.
      await expect.poll(() => retryAfterRefresh(hits), { timeout: 20_000 }).toBe(true)
      await expect(page).toHaveURL(/\/admin\/departments$/)
      await expect(sidebar(page)).toBeVisible()
      await expect(page.getByRole('alert')).toHaveCount(0)
      // Exactly one refresh after the revoked request (the page's own boot refresh came earlier).
      const revoked = hits.findIndex((h) => isProtected(h) && h.status === 401)
      expect(
        hits.slice(revoked + 1).filter((h) => h.path === '/api/v1/auth/refresh' && h.status === 200),
      ).toHaveLength(1)
    } finally {
      await ctxA.close()
    }
  })
})

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

async function api<T>(method: string, path: string, token: string | null, body?: unknown): Promise<ApiResult<T>> {
  const headers: Record<string, string> = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const json = (await res.json().catch(() => ({}))) as { data?: T | null; error?: ApiResult<T>['error'] }
  return { status: res.status, data: json.data ?? null, error: json.error ?? null }
}

async function roleIdByName(name: string): Promise<string> {
  const roles = await api<{ id: string; name: string }[]>('GET', '/api/v1/users/roles', adminToken)
  const id = roles.data?.find((r) => r.name === name)?.id
  if (!id) throw new Error(`role "${name}" not found`)
  return id
}

async function createDepartment(): Promise<string> {
  const res = await api<{ id: string }>('POST', '/api/v1/departments', adminToken, {
    name: `E2E TokenRevoked Dept ${Date.now()}`,
  })
  if (!res.data?.id) throw new Error(`createDepartment failed: ${res.error?.message ?? res.status}`)
  departmentIds.push(res.data.id)
  return res.data.id
}

/**
 * Creates a user with the given role and clears the forced password change (ISS-160), so the
 * account signs in with PASS_FIRST.
 */
async function createAccount(roleName: string, prefix: string): Promise<Account> {
  const email = `${prefix}-${Date.now()}@e2e-test.local`
  const fullName = `E2E ${prefix}`
  const created = await api<{ id: string; temporary_password: string }>('POST', '/api/v1/users', adminToken, {
    email,
    full_name: fullName,
    role_id: await roleIdByName(roleName),
  })
  if (!created.data?.id || !created.data.temporary_password) {
    throw new Error(`createAccount(${prefix}) failed: ${created.error?.message ?? created.status}`)
  }
  const login = await loginClearingForceChange(BASE, email, [created.data.temporary_password], PASS_FIRST)
  if (!login) throw new Error(`createAccount(${prefix}): first sign-in failed`)
  const account: Account = {
    id: created.data.id,
    email,
    fullName,
    password: login.password,
    token: login.token,
  }
  accounts.push(account)
  return account
}

function sidebar(page: Page): Locator {
  return page.getByRole('navigation', { name: /main/i })
}

function sidebarLink(page: Page, href: string): Locator {
  return sidebar(page).locator(`a[href="${href}"]`)
}

/**
 * Signs in through the form. The access token the page receives must belong to the account under test,
 * so the spec cannot pass by exercising another user's session.
 */
async function signIn(page: Page, account: Account) {
  await page.goto('/login')
  const login = page.waitForResponse((r) => new URL(r.url()).pathname === '/api/v1/auth/login' && r.status() === 200)
  await submitLogin(page, account.email, account.password)
  const body = (await (await login).json()) as { data: { access_token: string } }
  const subject = JSON.parse(Buffer.from(body.data.access_token.split('.')[1], 'base64url').toString('utf8')) as {
    sub?: string
  }
  expect(subject.sub, 'the page token must belong to the account under test').toBe(account.id)
}

/** A context with no storage state, so it never inherits the project's seeded admin or employee token. */
function isolatedContext(browser: Browser) {
  return browser.newContext({ storageState: { cookies: [], origins: [] } })
}

async function submitLogin(page: Page, email: string, password: string) {
  await page.getByRole('textbox', { name: /email/i }).fill(email)
  await page.getByLabel(/пароль|password|құпиясөз/i).first().fill(password)
  await page.getByRole('button', { name: /войти|login|sign in|кіру/i }).click()
}

function settle(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, SETTLE_MS))
}

/** Context B changes the account's password through the API; that revokes context A's token. */
async function changePasswordInContextB(browser: Browser, account: Account, newPassword: string) {
  const ctxB = await isolatedContext(browser)
  try {
    const res = await ctxB.request.post(`${BASE}/api/v1/auth/change-password`, {
      headers: { Authorization: `Bearer ${account.token}` },
      data: { current_password: account.password, new_password: newPassword },
    })
    expect(res.status(), 'password change in context B').toBe(200)
  } finally {
    await ctxB.close()
  }
  account.password = newPassword
}

/** Records every /api/v1/ response of the page, in arrival order. */
function trackApi(page: Page): Hit[] {
  const hits: Hit[] = []
  page.on('response', (res) => {
    const { pathname } = new URL(res.url())
    if (!pathname.startsWith('/api/v1/')) return
    hits.push({ method: res.request().method(), path: pathname, status: res.status() })
  })
  return hits
}

/** Protected means not an auth or public tenant-config call. */
function isProtected(h: Hit): boolean {
  return !h.path.startsWith('/api/v1/auth/') && !h.path.startsWith('/api/v1/tenant/')
}

/** True once a protected 401, then a refresh 200, then a protected 200 have been seen in order. */
function retryAfterRefresh(hits: Hit[]): boolean {
  const revoked = hits.findIndex((h) => isProtected(h) && h.status === 401)
  if (revoked < 0) return false
  const refreshed = hits.findIndex((h, i) => i > revoked && h.path === '/api/v1/auth/refresh' && h.status === 200)
  if (refreshed < 0) return false
  return hits.some((h, i) => i > refreshed && isProtected(h) && h.status === 200)
}

/**
 * Sequence (a) outcome: the session ended at /login with the notice, and no protected request
 * followed the refresh 401.
 */
async function assertSessionEnded(page: Page, hits: Hit[]) {
  await expect(page).toHaveURL(/\/login(\?|$)/, { timeout: 20_000 })
  const alert = page.getByRole('alert')
  await expect(alert).toBeVisible()
  await expect(alert).toContainText(NOTICE_TEXT)

  await page.waitForTimeout(QUIET_MS)

  const revoked = hits.findIndex((h) => isProtected(h) && h.status === 401)
  expect(revoked, 'a protected request must have been rejected with 401 TOKEN_REVOKED').toBeGreaterThanOrEqual(0)
  const refreshFailed = hits.findIndex((h, i) => i > revoked && h.path === '/api/v1/auth/refresh' && h.status === 401)
  expect(refreshFailed, 'the refresh after the revoked request must be rejected with 401').toBeGreaterThanOrEqual(0)
  const after = hits.slice(refreshFailed + 1)
  expect(after.filter(isProtected), 'protected requests sent after the refresh 401').toEqual([])
  expect(
    after.filter((h) => h.path === '/api/v1/auth/refresh'),
    'refresh attempted again after its 401',
  ).toEqual([])
}
