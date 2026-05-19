import { test, expect } from '@playwright/test'

const ADMIN_EMAIL = process.env.E2E_ADMIN_EMAIL ?? 'admin@bilimbaga.local'
const ADMIN_PASS = process.env.E2E_ADMIN_PASS ?? 'Admin1234!'

test.describe('Auth — login flow', () => {
  test('redirects to /login when unauthenticated', async ({ browser }) => {
    // Use a fresh context with no stored auth
    const ctx = await browser.newContext()
    const page = await ctx.newPage()
    await page.goto('/')
    await expect(page).toHaveURL(/\/login/)
    await ctx.close()
  })

  test('shows the login form with email and password fields', async ({ browser }) => {
    const ctx = await browser.newContext()
    const page = await ctx.newPage()
    await page.goto('/login')
    await expect(page.getByRole('textbox', { name: /email/i })).toBeVisible()
    await expect(page.getByLabel(/пароль|password/i).first()).toBeVisible()
    await expect(page.getByRole('button', { name: /войти|login|sign in/i })).toBeVisible()
    await ctx.close()
  })

  test('shows an error on invalid credentials', async ({ browser }) => {
    const ctx = await browser.newContext()
    const page = await ctx.newPage()
    await page.goto('/login')
    await page.getByRole('textbox', { name: /email/i }).fill('nobody@example.com')
    await page.getByLabel(/пароль|password/i).first().fill('wrongpassword')
    await page.getByRole('button', { name: /войти|login|sign in/i }).click()
    await expect(page.getByText(/неверный|invalid|credentials|incorrect|unauthorized/i)).toBeVisible({ timeout: 10_000 })
    await ctx.close()
  })

  test('navigates to /admin after successful login', async ({ browser }) => {
    const ctx = await browser.newContext()
    const page = await ctx.newPage()
    await page.goto('/login')
    await page.getByRole('textbox', { name: /email/i }).fill(ADMIN_EMAIL)
    await page.getByLabel(/пароль|password/i).first().fill(ADMIN_PASS)
    await page.getByRole('button', { name: /войти|login|sign in/i }).click()
    // After login, app navigates to /admin, /portal, or /change-password depending on role
    await expect(page).toHaveURL(/\/(admin|portal|change-password)/, { timeout: 20_000 })
    await ctx.close()
  })
})

test.describe('Auth — admin access control', () => {
  // Uses the storageState-injected admin session from global setup
  test('admin can access /admin', async ({ page }) => {
    await page.goto('/admin/users')
    await expect(page).not.toHaveURL(/\/login/)
    await expect(page.getByRole('navigation', { name: /main/i })).toBeVisible()
  })
})
