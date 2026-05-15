import { test, expect } from '@playwright/test'
import { mockRefreshExpired, mockRefreshSuccess } from './fixtures/helpers'
import { mockLogin, mockMe, mockTenantConfig, adminUser } from './fixtures/api'

test.describe('Auth — login flow', () => {
  test.beforeEach(async ({ page }) => {
    await mockRefreshExpired(page)
    await mockTenantConfig(page)
  })

  test('redirects to /login when unauthenticated', async ({ page }) => {
    await page.goto('/')
    await expect(page).toHaveURL(/\/login/)
  })

  test('shows the login form with email and password fields', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByRole('textbox', { name: /email/i })).toBeVisible()
    await expect(page.getByLabel(/password/i)).toBeVisible()
    await expect(page.getByRole('button', { name: /login|sign in/i })).toBeVisible()
  })

  test('shows an error on invalid credentials', async ({ page }) => {
    await mockLogin(page, false)
    await page.goto('/login')
    await page.getByRole('textbox', { name: /email/i }).fill('wrong@example.com')
    await page.getByLabel(/password/i).fill('wrongpassword')
    await page.getByRole('button', { name: /login|sign in/i }).click()
    await expect(page.getByText(/invalid|credentials|incorrect/i)).toBeVisible()
  })

  test('navigates to /admin after successful login', async ({ page }) => {
    await mockLogin(page, true)
    await mockMe(page)
    await page.route('**/api/v1/users*', (route) =>
      route.fulfill({ json: { data: { items: [], meta: { page: 1, per_page: 20, total: 0 } }, error: null } }),
    )
    await page.goto('/login')
    await page.getByRole('textbox', { name: /email/i }).fill('admin@example.com')
    await page.getByLabel(/password/i).fill('password123')
    await page.getByRole('button', { name: /login|sign in/i }).click()
    await expect(page).toHaveURL(/\/admin/, { timeout: 10000 })
  })
})

test.describe('Auth — admin access control', () => {
  test('admin can access /admin', async ({ page }) => {
    await mockRefreshSuccess(page)
    await mockMe(page, adminUser)
    await mockTenantConfig(page)
    await page.route('**/api/v1/users*', (route) =>
      route.fulfill({ json: { data: { items: [], meta: { page: 1, per_page: 20, total: 0 } }, error: null } }),
    )
    await page.goto('/admin/users')
    await expect(page).not.toHaveURL(/\/login/)
    await expect(page.getByRole('navigation')).toBeVisible()
  })
})
