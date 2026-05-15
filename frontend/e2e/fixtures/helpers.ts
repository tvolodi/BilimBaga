import type { Page } from '@playwright/test'

export function makeJwt(role = 'super_admin') {
  const header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64')
  const payload = Buffer.from(JSON.stringify({ sub: 'u-1', email: 'admin@example.com', role, exp: 9999999999 })).toString('base64')
  return `${header}.${payload}.sig`
}

export async function mockRefreshSuccess(page: Page, role = 'super_admin') {
  await page.route('**/api/v1/auth/refresh', (route) =>
    route.fulfill({ json: { data: { access_token: makeJwt(role) }, error: null } }),
  )
}

export async function mockRefreshExpired(page: Page) {
  await page.route('**/api/v1/auth/refresh', (route) =>
    route.fulfill({ status: 401, json: { data: null, error: { code: 'UNAUTHORIZED', message: 'expired' } } }),
  )
}
