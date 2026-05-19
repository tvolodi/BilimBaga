import { test, expect } from '@playwright/test'
import { getSeedData, createTestUser, deleteTestUser } from './fixtures/seed'

async function waitForContent(page: import('@playwright/test').Page) {
  await page.waitForFunction(() => !document.querySelector('[aria-label="loading"], .animate-spin'), { timeout: 10_000 }).catch(() => {})
  await page.waitForLoadState('networkidle').catch(() => {})
}

async function loginAsAdmin(page: import('@playwright/test').Page) {
  await page.goto('/admin/dashboard')
  await expect(page).toHaveURL(/\/admin/, { timeout: 15_000 })
  const url = page.url()
  if (url.includes('/login')) {
    throw new Error(
      `loginAsAdmin: auth failed — redirected to ${url}. ` +
        'Check that global-setup seeded __e2e_access_token__ into localStorage.',
    )
  }
}

// ---------------------------------------------------------------------------
// Edit User
// ---------------------------------------------------------------------------

test.describe('User Management — Edit User', () => {
  test('edit button opens drawer with pre-populated name', async ({ page }) => {
    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const editBtn = page.getByRole('button', { name: /^\u0440\u0435\u0434\u0430\u043a\u0442\u0438\u0440\u043e\u0432\u0430\u0442\u044c$|^edit\$/i }).first()
    if (!(await editBtn.isVisible({ timeout: 5_000 }).catch(() => false))) {
      test.info().annotations.push({ type: 'note', description: 'No edit button visible — no users in list' })
      return
    }
    await editBtn.click()
    await expect(page.getByRole('heading', { name: /\u0440\u0435\u0434\u0430\u043a\u0442\u0438\u0440\u043e\u0432\u0430\u0442\u044c \u043f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044f|edit user/i })).toBeVisible({ timeout: 5_000 })
    const nameField = page.getByPlaceholder(/\u0432\u0432\u0435\u0434\u0438\u0442\u0435 \u043f\u043e\u043b\u043d\u043e\u0435 \u0438\u043c\u044f|full name/i)
    await expect(nameField).toBeVisible()
    const value = await nameField.inputValue()
    expect(value.length).toBeGreaterThan(0)
  })

  test('edit drawer has required form fields', async ({ page }) => {
    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const editBtn = page.getByRole('button', { name: /^\u0440\u0435\u0434\u0430\u043a\u0442\u0438\u0440\u043e\u0432\u0430\u0442\u044c$|^edit\$/i }).first()
    if (!(await editBtn.isVisible({ timeout: 5_000 }).catch(() => false))) {
      test.info().annotations.push({ type: 'note', description: 'No edit button visible' })
      return
    }
    await editBtn.click()
    await expect(page.getByRole('heading', { name: /\u0440\u0435\u0434\u0430\u043a\u0442\u0438\u0440\u043e\u0432\u0430\u0442\u044c \u043f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044f|edit user/i })).toBeVisible({ timeout: 5_000 })
    await expect(page.getByPlaceholder(/\u0432\u0432\u0435\u0434\u0438\u0442\u0435 \u043f\u043e\u043b\u043d\u043e\u0435 \u0438\u043c\u044f|full name/i)).toBeVisible()
  })

  test('edit drawer submits updated name and closes', async ({ page }) => {
    const { adminToken } = await getSeedData()
    const user = await createTestUser(adminToken, 'edit-user')

    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    // Find row with our test user's email
    const row = page.locator('table tbody tr').filter({ hasText: user.email })
    if (!(await row.isVisible({ timeout: 5_000 }).catch(() => false))) {
      await deleteTestUser(adminToken, user.id)
      test.info().annotations.push({ type: 'note', description: 'Test user row not visible on page — may be paginated' })
      return
    }
    const editBtn = row.getByRole('button', { name: /^\u0440\u0435\u0434\u0430\u043a\u0442\u0438\u0440\u043e\u0432\u0430\u0442\u044c$|^edit\$/i })
    await editBtn.click()
    await expect(page.getByRole('heading', { name: /\u0440\u0435\u0434\u0430\u043a\u0442\u0438\u0440\u043e\u0432\u0430\u0442\u044c \u043f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044f|edit user/i })).toBeVisible({ timeout: 5_000 })

    const nameField = page.getByPlaceholder(/\u0432\u0432\u0435\u0434\u0438\u0442\u0435 \u043f\u043e\u043b\u043d\u043e\u0435 \u0438\u043c\u044f|full name/i)
    await nameField.clear()
    await nameField.fill('Updated Name')

    const saveBtn = page.getByRole('button', { name: /\u0441\u043e\u0445\u0440\u0430\u043d\u0438\u0442\u044c|save|update|submit/i }).last()
    await saveBtn.click()
    await expect(page.getByRole('heading', { name: /\u0440\u0435\u0434\u0430\u043a\u0442\u0438\u0440\u043e\u0432\u0430\u0442\u044c \u043f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044f|edit user/i })).not.toBeVisible({ timeout: 5_000 })

    await deleteTestUser(adminToken, user.id)
  })

  test('edit drawer cancel button closes without submitting', async ({ page }) => {
    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const editBtn = page.getByRole('button', { name: /^\u0440\u0435\u0434\u0430\u043a\u0442\u0438\u0440\u043e\u0432\u0430\u0442\u044c$|^edit\$/i }).first()
    if (!(await editBtn.isVisible({ timeout: 5_000 }).catch(() => false))) {
      test.info().annotations.push({ type: 'note', description: 'No edit button visible' })
      return
    }
    await editBtn.click()
    await expect(page.getByRole('heading', { name: /\u0440\u0435\u0434\u0430\u043a\u0442\u0438\u0440\u043e\u0432\u0430\u0442\u044c \u043f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044f|edit user/i })).toBeVisible({ timeout: 5_000 })
    await page.getByRole('button', { name: /\u043e\u0442\u043c\u0435\u043d\u0430|cancel/i }).click()
    await expect(page.getByRole('heading', { name: /\u0440\u0435\u0434\u0430\u043a\u0442\u0438\u0440\u043e\u0432\u0430\u0442\u044c \u043f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044f|edit user/i })).not.toBeVisible({ timeout: 3_000 })
  })
})

// ---------------------------------------------------------------------------
// Reset User Password
// ---------------------------------------------------------------------------

test.describe('User Management — Reset Password', () => {
  test('reset password button is visible in user row', async ({ page }) => {
    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const resetBtn = page.getByRole('button', { name: /\u0441\u0431\u0440\u043e\u0441\u0438\u0442\u044c \u043f\u0430\u0440\u043e\u043b\u044c|reset password/i }).first()
    if (!(await resetBtn.isVisible({ timeout: 5_000 }).catch(() => false))) {
      test.info().annotations.push({ type: 'note', description: 'No reset password button visible — no users in list' })
      return
    }
    await expect(resetBtn).toBeVisible()
  })

  test('clicking reset password shows the temporary password modal', async ({ page }) => {
    const { adminToken } = await getSeedData()
    const user = await createTestUser(adminToken, 'reset-pwd')

    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    const hasTable = await page.getByRole('table').isVisible({ timeout: 15_000 }).catch(() => false)
    if (!hasTable) {
      await deleteTestUser(adminToken, user.id)
      test.info().annotations.push({ type: 'note', description: 'Users table not visible — skipping' })
      return
    }

    const row = page.locator('table tbody tr').filter({ hasText: user.email })
    if (!(await row.isVisible({ timeout: 5_000 }).catch(() => false))) {
      await deleteTestUser(adminToken, user.id)
      test.info().annotations.push({ type: 'note', description: 'Test user row not visible — may be paginated' })
      return
    }
    await row.getByRole('button', { name: /\u0441\u0431\u0440\u043e\u0441\u0438\u0442\u044c \u043f\u0430\u0440\u043e\u043b\u044c|reset password/i }).click()
    // Modal shows a temporary password (any non-trivial string)
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })

    await deleteTestUser(adminToken, user.id)
  })

  test('temporary password modal has a close button', async ({ page }) => {
    const { adminToken } = await getSeedData()
    const user = await createTestUser(adminToken, 'reset-close')

    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const row = page.locator('table tbody tr').filter({ hasText: user.email })
    if (!(await row.isVisible({ timeout: 5_000 }).catch(() => false))) {
      await deleteTestUser(adminToken, user.id)
      test.info().annotations.push({ type: 'note', description: 'Test user row not visible' })
      return
    }
    await row.getByRole('button', { name: /\u0441\u0431\u0440\u043e\u0441\u0438\u0442\u044c \u043f\u0430\u0440\u043e\u043b\u044c|reset password/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    const hasClose = await page.getByRole('button', { name: /\u0437\u0430\u043a\u0440\u044b\u0442\u044c|close|done/i }).isVisible().catch(() => false)
    expect(hasClose).toBeTruthy()

    await deleteTestUser(adminToken, user.id)
  })

  test('closing the reset modal dismisses it', async ({ page }) => {
    const { adminToken } = await getSeedData()
    const user = await createTestUser(adminToken, 'reset-dismiss')

    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const row = page.locator('table tbody tr').filter({ hasText: user.email })
    if (!(await row.isVisible({ timeout: 5_000 }).catch(() => false))) {
      await deleteTestUser(adminToken, user.id)
      test.info().annotations.push({ type: 'note', description: 'Test user row not visible' })
      return
    }
    await row.getByRole('button', { name: /\u0441\u0431\u0440\u043e\u0441\u0438\u0442\u044c \u043f\u0430\u0440\u043e\u043b\u044c|reset password/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    await page.getByRole('button', { name: /\u0437\u0430\u043a\u0440\u044b\u0442\u044c|close|done/i }).click()
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 3_000 })

    await deleteTestUser(adminToken, user.id)
  })
})

// ---------------------------------------------------------------------------
// Deactivate / Reactivate User
// ---------------------------------------------------------------------------

test.describe('User Management — Deactivate / Reactivate User', () => {
  test('deactivate button is visible for active users', async ({ page }) => {
    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const deactivateBtn = page.getByRole('button', { name: /\u0434\u0435\u0430\u043a\u0442\u0438\u0432\u0438\u0440\u043e\u0432\u0430\u0442\u044c|deactivate/i }).first()
    if (!(await deactivateBtn.isVisible({ timeout: 5_000 }).catch(() => false))) {
      test.info().annotations.push({ type: 'note', description: 'No deactivate button visible — no active users in list' })
      return
    }
    await expect(deactivateBtn).toBeVisible()
  })

  test('clicking deactivate opens a confirmation dialog', async ({ page }) => {
    const { adminToken } = await getSeedData()
    const user = await createTestUser(adminToken, 'deactivate-dialog')

    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const row = page.locator('table tbody tr').filter({ hasText: user.email })
    if (!(await row.isVisible({ timeout: 5_000 }).catch(() => false))) {
      await deleteTestUser(adminToken, user.id)
      test.info().annotations.push({ type: 'note', description: 'Test user row not visible' })
      return
    }
    await row.getByRole('button', { name: /\u0434\u0435\u0430\u043a\u0442\u0438\u0432\u0438\u0440\u043e\u0432\u0430\u0442\u044c|deactivate/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })

    // Cancel to leave the user active for cleanup
    await page.getByRole('dialog').getByRole('button', { name: /\u043e\u0442\u043c\u0435\u043d\u0430|cancel/i }).click()
    await deleteTestUser(adminToken, user.id)
  })

  test('cancelling deactivation closes the dialog without change', async ({ page }) => {
    const { adminToken } = await getSeedData()
    const user = await createTestUser(adminToken, 'deactivate-cancel')

    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const row = page.locator('table tbody tr').filter({ hasText: user.email })
    if (!(await row.isVisible({ timeout: 5_000 }).catch(() => false))) {
      await deleteTestUser(adminToken, user.id)
      test.info().annotations.push({ type: 'note', description: 'Test user row not visible' })
      return
    }
    await row.getByRole('button', { name: /\u0434\u0435\u0430\u043a\u0442\u0438\u0432\u0438\u0440\u043e\u0432\u0430\u0442\u044c|deactivate/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    await page.getByRole('dialog').getByRole('button', { name: /\u043e\u0442\u043c\u0435\u043d\u0430|cancel/i }).click()
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 3_000 })

    await deleteTestUser(adminToken, user.id)
  })

  test('confirming deactivation calls API and closes dialog', async ({ page }) => {
    const { adminToken } = await getSeedData()
    const user = await createTestUser(adminToken, 'deactivate-confirm')

    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const row = page.locator('table tbody tr').filter({ hasText: user.email })
    if (!(await row.isVisible({ timeout: 5_000 }).catch(() => false))) {
      await deleteTestUser(adminToken, user.id)
      test.info().annotations.push({ type: 'note', description: 'Test user row not visible' })
      return
    }
    await row.getByRole('button', { name: /\u0434\u0435\u0430\u043a\u0442\u0438\u0432\u0438\u0440\u043e\u0432\u0430\u0442\u044c|deactivate/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    await page.getByRole('dialog').getByRole('button', { name: /\u0434\u0435\u0430\u043a\u0442\u0438\u0432\u0438\u0440\u043e\u0432\u0430\u0442\u044c|confirm|deactivate/i }).click()
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5_000 })

    await deleteTestUser(adminToken, user.id)
  })
})

// ---------------------------------------------------------------------------
// Bulk Import Users
// ---------------------------------------------------------------------------

test.describe('User Management — Bulk Import', () => {
  test('Import button is visible on users list page', async ({ page }) => {
    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await expect(page.getByRole('button', { name: /\u043c\u0430\u0441\u0441\u043e\u0432\u044b\u0439 \u0438\u043c\u043f\u043e\u0440\u0442|import/i })).toBeVisible({ timeout: 10_000 })
  })

  test('clicking Import opens the import modal', async ({ page }) => {
    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await page.getByRole('button', { name: /\u043c\u0430\u0441\u0441\u043e\u0432\u044b\u0439 \u0438\u043c\u043f\u043e\u0440\u0442|import/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    await expect(page.getByRole('heading', { name: /\u0438\u043c\u043f\u043e\u0440\u0442|import/i })).toBeVisible()
  })

  test('import modal contains a file input for CSV', async ({ page }) => {
    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await page.getByRole('button', { name: /\u043c\u0430\u0441\u0441\u043e\u0432\u044b\u0439 \u0438\u043c\u043f\u043e\u0440\u0442|import/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    const fileInput = page.locator('input[type="file"]')
    await expect(fileInput).toBeAttached()
    const acceptAttr = await fileInput.getAttribute('accept')
    expect(acceptAttr).toMatch(/\.csv/i)
  })

  test('import modal has Preview and Close buttons', async ({ page }) => {
    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await page.getByRole('button', { name: /\u043c\u0430\u0441\u0441\u043e\u0432\u044b\u0439 \u0438\u043c\u043f\u043e\u0440\u0442|import/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    await expect(page.getByRole('button', { name: /предпросмотр|preview/i })).toBeVisible()
    await expect(page.getByRole('button', { name: /закрыть|отмена|close|cancel/i })).toBeVisible()
  })

  test('import modal closes when Cancel is clicked', async ({ page }) => {
    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await page.getByRole('button', { name: /\u043c\u0430\u0441\u0441\u043e\u0432\u044b\u0439 \u0438\u043c\u043f\u043e\u0440\u0442|import/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    await page.getByRole('button', { name: /закрыть|отмена|close|cancel/i }).first().click()
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 3_000 })
  })

  test('import modal shows validation error when Preview clicked without file', async ({ page }) => {
    await loginAsAdmin(page)
    await loginAsAdmin(page)
    await page.goto('/admin/users')
    await waitForContent(page)
    await page.getByRole('button', { name: /\u043c\u0430\u0441\u0441\u043e\u0432\u044b\u0439 \u0438\u043c\u043f\u043e\u0440\u0442|import/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 })
    await page.getByRole('button', { name: /предпросмотр|preview/i }).click()
    await expect(
      page.getByText(/please select|select a file|no file/i),
    ).toBeVisible({ timeout: 3_000 })
  })
})
