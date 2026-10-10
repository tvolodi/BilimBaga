import { test, expect } from '@playwright/test'

// #415: written, NOT run here. It needs the live stack and admin credentials; UAT runs it live.
// It relies on the same session setup as the other admin specs (the e2e build seeds the token).

const TOKEN_KEY = '__e2e_access_token__'

test.describe('Logout and reload (#415)', () => {
  test('after logout, a reload lands on /login and no access token is stored', async ({ page }) => {
    await page.goto('/admin/dashboard')
    await expect(page).toHaveURL(/\/admin/, { timeout: 15_000 })

    // A reload keeps the session. In a default build the access token comes from POST /auth/refresh
    // (httpOnly cookie), never from localStorage.
    await page.reload()
    await expect(page).toHaveURL(/\/admin/, { timeout: 15_000 })

    // Sign out from the top bar (labels: Sign out / Выйти / Шығу).
    await page.getByRole('button', { name: /^(Sign out|Выйти|Шығу)$/ }).first().click()
    await expect(page).toHaveURL(/\/login/, { timeout: 10_000 })

    // After logout, a reload must not sign the user back in.
    await page.reload()
    await expect(page).toHaveURL(/\/login/, { timeout: 10_000 })

    // No access token is left in localStorage.
    const stored = await page.evaluate((key) => localStorage.getItem(key), TOKEN_KEY)
    expect(stored).toBeNull()
  })
})
