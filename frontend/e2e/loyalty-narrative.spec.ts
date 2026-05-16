/**
 * E2E tests for FR-BB75 — Loyalty Profile Narrative
 *
 * Covers:
 *  - Golden path: department_admin generates a narrative for a loyalty session
 *  - Error path: AI service unavailable shows error message
 *  - Visibility: non-loyalty sessions do not show Values Profile section
 *  - Role guard: employee role does not see Values Profile section
 */

import { test, expect } from '@playwright/test'
import { mockMe, mockTenantConfig, adminUser } from './fixtures/api'
import { mockRefreshSuccess } from './fixtures/helpers'

const DEPT_ADMIN = {
  ...adminUser,
  id: 'u-admin',
  role_name: 'department_admin',
  department_id: 'dept-1',
}

const EMPLOYEE_USER = {
  ...adminUser,
  id: 'u-emp',
  role_name: 'employee',
}

const EMPLOYEE_PROFILE_USER = {
  ...adminUser,
  id: 'u-emp',
  email: 'emp@example.com',
  full_name: 'Jane Employee',
  role_name: 'employee',
}

const LOYALTY_SESSION_ID = 'sess-loyalty-1'
const NON_LOYALTY_SESSION_ID = 'sess-other-1'

const loyaltySession = {
  session_id: LOYALTY_SESSION_ID,
  exam_id: 'exam-1',
  exam_title: 'Values Assessment',
  started_at: '2025-01-10T09:00:00Z',
  submitted_at: '2025-01-10T09:30:00Z',
  score_pct: 80,
  passed: true,
  time_taken_seconds: 1800,
  status: 'submitted',
  certificate_id: null,
  exam_category_track: 'loyalty',
}

const regularSession = {
  session_id: NON_LOYALTY_SESSION_ID,
  exam_id: 'exam-2',
  exam_title: 'Technical Exam',
  started_at: '2025-01-15T09:00:00Z',
  submitted_at: '2025-01-15T10:00:00Z',
  score_pct: 90,
  passed: true,
  time_taken_seconds: 3600,
  status: 'submitted',
  certificate_id: null,
  exam_category_track: 'technical',
}

async function mockEmployeeRecord(
  page: import('@playwright/test').Page,
  sessions: typeof loyaltySession[],
) {
  await page.route('**/api/v1/users/u-emp', (route) =>
    route.fulfill({ json: { data: EMPLOYEE_PROFILE_USER, error: null } }),
  )
  await page.route('**/api/v1/reports/employees/u-emp/sessions*', (route) =>
    route.fulfill({
      json: {
        data: {
          sessions,
          meta: { page: 1, per_page: 20, total: sessions.length },
        },
        error: null,
      },
    }),
  )
  await page.route('**/api/v1/reports/employees/u-emp/progress', (route) =>
    route.fulfill({ json: { data: { tracks: [] }, error: null } }),
  )
}

test.describe('FR-BB75 — Values Profile Section', () => {
  test.beforeEach(async ({ page }) => {
    await mockRefreshSuccess(page)
    await mockTenantConfig(page)
  })

  test('department_admin sees Generate button for loyalty session', async ({ page }) => {
    await mockMe(page, DEPT_ADMIN)
    await mockEmployeeRecord(page, [loyaltySession])

    await page.goto('/admin/employees/u-emp')

    // The Values Profile section title should be visible
    await expect(page.getByText(/values profile/i)).toBeVisible({ timeout: 10000 })
    // Generate button should be present
    await expect(page.getByRole('button', { name: /generate/i })).toBeVisible()
  })

  test('clicking Generate shows narrative on success', async ({ page }) => {
    await mockMe(page, DEPT_ADMIN)
    await mockEmployeeRecord(page, [loyaltySession])

    // Mock the loyalty narrative endpoint
    await page.route(`**/api/v1/admin/ai/loyalty-summary/${LOYALTY_SESSION_ID}`, (route) =>
      route.fulfill({
        json: {
          data: {
            narrative: 'This employee demonstrates strong loyalty and commitment to team values.',
            generated_at: '2025-01-10T12:00:00Z',
          },
          error: null,
        },
      }),
    )

    await page.goto('/admin/employees/u-emp')

    await page.getByRole('button', { name: /^generate$/i }).click()

    // Narrative text should appear
    await expect(
      page.getByText(/strong loyalty and commitment/i),
    ).toBeVisible({ timeout: 10000 })

    // Regenerate button should now appear (replacing Generate)
    await expect(page.getByRole('button', { name: /regenerate/i })).toBeVisible()
  })

  test('shows error message when AI is unavailable', async ({ page }) => {
    await mockMe(page, DEPT_ADMIN)
    await mockEmployeeRecord(page, [loyaltySession])

    // Mock AI as unavailable
    await page.route(`**/api/v1/admin/ai/loyalty-summary/${LOYALTY_SESSION_ID}`, (route) =>
      route.fulfill({
        status: 503,
        json: {
          data: null,
          error: { code: 'AI_UNAVAILABLE', message: 'AI service is temporarily unavailable.' },
        },
      }),
    )

    await page.goto('/admin/employees/u-emp')

    await page.getByRole('button', { name: /^generate$/i }).click()

    // Error message should be visible
    await expect(page.getByText(/unavailable/i)).toBeVisible({ timeout: 10000 })
  })

  test('Values Profile section is NOT shown for non-loyalty sessions', async ({ page }) => {
    await mockMe(page, DEPT_ADMIN)
    await mockEmployeeRecord(page, [regularSession])

    await page.goto('/admin/employees/u-emp')

    // Wait for page to load
    await expect(page.getByText('Jane Employee')).toBeVisible({ timeout: 10000 })

    // Values Profile section should NOT appear
    await expect(page.getByText(/values profile/i)).not.toBeVisible()
  })

  test('employee role does NOT see Values Profile section', async ({ page }) => {
    await mockMe(page, EMPLOYEE_USER)
    await mockEmployeeRecord(page, [loyaltySession])

    await page.goto('/admin/employees/u-emp')

    // If the page even renders (access guard may redirect), no Values Profile section
    // Either redirect or no section visible
    const hasSection = await page.getByText(/values profile/i).isVisible().catch(() => false)
    expect(hasSection).toBe(false)
  })
})
