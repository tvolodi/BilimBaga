import { test, expect } from '@playwright/test'
import { mockRefreshSuccess } from './fixtures/helpers'
import { mockMe, mockTenantConfig, mockCategories, mockTags, adminUser, sampleCategory } from './fixtures/api'
import type { Page } from '@playwright/test'

// ---- Fixtures ---------------------------------------------------------------

const sampleExam = {
  id: 'exam-1',
  title: 'Safety Exam',
  description: 'Annual safety examination',
  status: 'draft',
  time_limit_minutes: 60,
  passing_score_pct: 70,
  max_attempts: 3,
  shuffle_questions: true,
  shuffle_options: false,
  show_answers: 'never',
  on_tab_switch: 'log',
  certificate_enabled: false,
  available_from: null,
  available_until: null,
  rules: [],
  sections: [{ id: 'sec-1', title: 'Default', sort_order: 1 }],
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
}

async function mockExamsApi(page: Page, exam = sampleExam) {
  // GET exam by id
  await page.route(`**/api/v1/exams/${exam.id}`, (route) => {
    if (route.request().method() === 'GET') {
      route.fulfill({ json: { data: exam, error: null } })
    } else {
      route.fulfill({ json: { data: { ...exam, ...JSON.parse(route.request().postData() ?? '{}') }, error: null } })
    }
  })

  // POST create exam
  await page.route('**/api/v1/exams', (route) => {
    if (route.request().method() === 'POST') {
      route.fulfill({ json: { data: { ...exam, id: 'exam-new' }, error: null } })
    } else {
      route.fulfill({ json: { data: { items: [exam], meta: { page: 1, per_page: 20, total: 1 } }, error: null } })
    }
  })

  // POST sections (create section after exam)
  await page.route('**/api/v1/exams/*/sections', (route) =>
    route.fulfill({ json: { data: { id: 'sec-new', title: 'Default', sort_order: 1 }, error: null } }),
  )

  // Rules
  await page.route('**/api/v1/exams/*/rules', (route) =>
    route.fulfill({ json: { data: { id: 'rule-1', mode: 'random', count: 10, section_id: 'sec-1' }, error: null } }),
  )

  // Assignments
  await page.route('**/api/v1/exams/*/assignments', (route) =>
    route.fulfill({ json: { data: [], error: null } }),
  )

  // Publish
  await page.route('**/api/v1/exams/*/publish', (route) =>
    route.fulfill({ json: { data: { ...exam, status: 'active' }, error: null } }),
  )
}

async function mockCommonApis(page: Page) {
  await mockRefreshSuccess(page)
  await mockMe(page, adminUser)
  await mockTenantConfig(page)
  await mockCategories(page, [sampleCategory])
  await mockTags(page)

  // Users list (for assignments)
  await page.route('**/api/v1/users*', (route) =>
    route.fulfill({
      json: { data: { items: [], meta: { page: 1, per_page: 200, total: 0 } }, error: null },
    }),
  )

  // Departments
  await page.route('**/api/v1/departments*', (route) =>
    route.fulfill({ json: { data: [], error: null } }),
  )
}

// ---- Tests ------------------------------------------------------------------

test.describe('Exam Wizard — Create', () => {
  test.beforeEach(async ({ page }) => {
    await mockCommonApis(page)
    await mockExamsApi(page)
  })

  test('renders the step indicator and Basic Settings heading', async ({ page }) => {
    await page.goto('/admin/exams/new')
    await expect(page.getByText('Basic Settings')).toBeVisible()
    // Step indicator shows 4 steps
    await expect(page.getByText('1')).toBeVisible()
    await expect(page.getByText('4')).toBeVisible()
  })

  test('shows validation error when title is empty', async ({ page }) => {
    await page.goto('/admin/exams/new')
    await page.getByRole('button', { name: /next/i }).click()
    // Some validation feedback visible (title required)
    await expect(page.locator('[data-error], .text-destructive, [role="alert"]').first()).toBeVisible()
  })

  test('golden path: fill Step 1 and advance to Step 2', async ({ page }) => {
    await page.goto('/admin/exams/new')

    // Fill required fields
    await page.getByLabel(/exam title/i).fill('Safety Exam')
    await page.getByLabel(/time limit/i).fill('60')
    await page.getByLabel(/passing score/i).fill('70')
    await page.getByLabel(/max attempts/i).fill('3')

    // Click Next
    await page.getByRole('button', { name: /next/i }).click()

    // Should advance to Step 2
    await expect(page.getByText('Question Rules')).toBeVisible()
  })

  test('can navigate back from Step 2 to Step 1', async ({ page }) => {
    await page.goto('/admin/exams/new')
    await page.getByLabel(/exam title/i).fill('Safety Exam')
    await page.getByLabel(/time limit/i).fill('60')
    await page.getByLabel(/passing score/i).fill('70')
    await page.getByLabel(/max attempts/i).fill('3')
    await page.getByRole('button', { name: /next/i }).click()
    await expect(page.getByText('Question Rules')).toBeVisible()

    await page.getByRole('button', { name: /back/i }).click()
    await expect(page.getByText('Basic Settings')).toBeVisible()
  })

  test('can reach Step 3 (Assignments) after passing steps 1 and 2', async ({ page }) => {
    await page.goto('/admin/exams/new')
    // Step 1
    await page.getByLabel(/exam title/i).fill('Safety Exam')
    await page.getByLabel(/time limit/i).fill('60')
    await page.getByLabel(/passing score/i).fill('70')
    await page.getByLabel(/max attempts/i).fill('3')
    await page.getByRole('button', { name: /next/i }).click()

    // Step 2 — just advance without adding rules
    await expect(page.getByText('Question Rules')).toBeVisible()
    await page.getByRole('button', { name: /next/i }).click()

    // Step 3
    await expect(page.getByText('Assignments')).toBeVisible()
  })

  test('can reach Step 4 (Review & Publish)', async ({ page }) => {
    // Mock the exam for the review step
    await page.route('**/api/v1/exams/exam-new', (route) =>
      route.fulfill({ json: { data: sampleExam, error: null } }),
    )

    await page.goto('/admin/exams/new')
    await page.getByLabel(/exam title/i).fill('Safety Exam')
    await page.getByLabel(/time limit/i).fill('60')
    await page.getByLabel(/passing score/i).fill('70')
    await page.getByLabel(/max attempts/i).fill('3')
    await page.getByRole('button', { name: /next/i }).click()

    await expect(page.getByText('Question Rules')).toBeVisible()
    await page.getByRole('button', { name: /next/i }).click()

    await expect(page.getByText('Assignments')).toBeVisible()
    await page.getByRole('button', { name: /next/i }).click()

    await expect(page.getByText(/Review & Publish/i)).toBeVisible()
  })
})

test.describe('Exam Wizard — Edit (pre-populated)', () => {
  test.beforeEach(async ({ page }) => {
    await mockCommonApis(page)
    await mockExamsApi(page)
  })

  test('loads existing exam data into Step 1', async ({ page }) => {
    // The edit page will use examId from URL and load the exam
    await page.route('**/api/v1/exams/exam-1', (route) =>
      route.fulfill({ json: { data: sampleExam, error: null } }),
    )

    await page.goto('/admin/exams/exam-1/edit')
    await expect(page.getByText(/Edit Exam|Basic Settings/i)).toBeVisible()
  })
})

test.describe('Exam Wizard — Publish flow', () => {
  test('publish button opens confirmation dialog', async ({ page }) => {
    await mockCommonApis(page)
    await mockExamsApi(page)

    // Mock exam for review step
    await page.route('**/api/v1/exams/exam-new', (route) =>
      route.fulfill({ json: { data: sampleExam, error: null } }),
    )

    await page.goto('/admin/exams/new')
    await page.getByLabel(/exam title/i).fill('Safety Exam')
    await page.getByLabel(/time limit/i).fill('60')
    await page.getByLabel(/passing score/i).fill('70')
    await page.getByLabel(/max attempts/i).fill('3')
    await page.getByRole('button', { name: /next/i }).click()
    await page.getByRole('button', { name: /next/i }).click()
    await page.getByRole('button', { name: /next/i }).click()

    await expect(page.getByText(/Review & Publish/i)).toBeVisible()
    const publishBtn = page.getByRole('button', { name: /publish/i })
    await expect(publishBtn).toBeVisible()
    await publishBtn.click()

    // Confirmation dialog should appear
    await expect(page.getByRole('dialog')).toBeVisible()
    await expect(page.getByText(/cannot be undone/i)).toBeVisible()
  })

  test('publish 422 shows unsatisfied rules warning', async ({ page }) => {
    await mockCommonApis(page)
    await mockExamsApi(page)

    // Override publish to return 422
    await page.route('**/api/v1/exams/*/publish', (route) =>
      route.fulfill({
        status: 422,
        json: {
          data: null,
          error: {
            code: 'ERR_VALIDATION',
            message: 'Insufficient questions',
            unsatisfied_rules: [{ rule_id: 'rule-abc123', required: 5, available: 2 }],
          },
        },
      }),
    )

    await page.route('**/api/v1/exams/exam-new', (route) =>
      route.fulfill({ json: { data: sampleExam, error: null } }),
    )

    await page.goto('/admin/exams/new')
    await page.getByLabel(/exam title/i).fill('Safety Exam')
    await page.getByLabel(/time limit/i).fill('60')
    await page.getByLabel(/passing score/i).fill('70')
    await page.getByLabel(/max attempts/i).fill('3')
    await page.getByRole('button', { name: /next/i }).click()
    await page.getByRole('button', { name: /next/i }).click()
    await page.getByRole('button', { name: /next/i }).click()

    // Attempt publish
    await page.getByRole('button', { name: /publish/i }).click()
    // Confirm in dialog
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await dialog.getByRole('button', { name: /publish/i }).click()

    // Warning banner should appear
    await expect(page.getByText(/cannot be satisfied|rules cannot/i)).toBeVisible()
  })
})

test.describe('Exam Wizard — Role-based access', () => {
  test('examiner can see Assignments step but cannot add assignments', async ({ page }) => {
    const examinerUser = {
      ...adminUser,
      role_name: 'examiner',
      role_id: 'role-ex',
    }
    await mockRefreshSuccess(page, 'examiner')
    await mockMe(page, examinerUser)
    await mockTenantConfig(page)
    await mockCategories(page, [sampleCategory])
    await mockTags(page)
    await page.route('**/api/v1/users*', (route) =>
      route.fulfill({ json: { data: { items: [], meta: { page: 1, per_page: 200, total: 0 } }, error: null } }),
    )
    await page.route('**/api/v1/departments*', (route) =>
      route.fulfill({ json: { data: [], error: null } }),
    )
    await mockExamsApi(page)
    await page.route('**/api/v1/exams/exam-new', (route) =>
      route.fulfill({ json: { data: sampleExam, error: null } }),
    )

    await page.goto('/admin/exams/new')
    await page.getByLabel(/exam title/i).fill('Safety Exam')
    await page.getByLabel(/time limit/i).fill('60')
    await page.getByLabel(/passing score/i).fill('70')
    await page.getByLabel(/max attempts/i).fill('3')
    await page.getByRole('button', { name: /next/i }).click()
    await page.getByRole('button', { name: /next/i }).click()

    await expect(page.getByText('Assignments')).toBeVisible()
    // "Add assignee" button should NOT be visible for non-admin
    await expect(page.getByRole('button', { name: /add assignee/i })).not.toBeVisible()
  })
})
