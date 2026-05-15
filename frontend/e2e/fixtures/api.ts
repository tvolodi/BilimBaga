import type { Page } from '@playwright/test'

export const adminUser = {
  id: 'u-1',
  email: 'admin@example.com',
  full_name: 'Admin User',
  department_id: null,
  department_name: null,
  role_id: 'role-sa',
  role_name: 'super_admin',
  status: 'active',
  force_password_change: false,
  created_at: '2024-01-01T00:00:00Z',
}

export const employeeUser = {
  ...adminUser,
  id: 'u-2',
  email: 'employee@example.com',
  role_name: 'employee',
  role_id: 'role-emp',
}

export const sampleCategory = {
  id: 'cat-1',
  name: 'Science',
  parent_id: null,
  track: null,
  sort_order: 1,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  children: [],
}

export const sampleTags = [
  { id: 'tag-1', name: 'Biology', created_at: '2024-01-01T00:00:00Z', usage_count: 5 },
  { id: 'tag-2', name: 'Chemistry', created_at: '2024-01-01T00:00:00Z', usage_count: 3 },
]

export const sampleQuestion = {
  id: 'q-1',
  type: 'single',
  difficulty: 'easy',
  status: 'draft',
  category_id: 'cat-1',
  category_name: 'Science',
  default_locale: 'en',
  version: 1,
  locale_coverage: ['en'],
  stem_preview: 'What is H2O?',
  tags: [],
  created_by: 'u-1',
  created_by_name: 'Admin User',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
}

/** Mock the /api/v1/users/me endpoint */
export async function mockMe(page: Page, user = adminUser) {
  await page.route('**/api/v1/users/me', (route) =>
    route.fulfill({ json: { data: user, error: null } }),
  )
}

/** Mock an unauthorized /me response */
export async function mockMeUnauthorized(page: Page) {
  await page.route('**/api/v1/users/me', (route) =>
    route.fulfill({
      status: 401,
      json: { data: null, error: { code: 'UNAUTHORIZED', message: 'unauthorized' } },
    }),
  )
}

/** Mock the tenant config endpoint (required by many pages) */
export async function mockTenantConfig(page: Page) {
  await page.route('**/api/v1/tenant/config', (route) =>
    route.fulfill({
      json: {
        data: {
          primary_color: '#4F46E5',
          logo_url: null,
          app_name: 'BilimBaga',
          favicon_url: null,
        },
        error: null,
      },
    }),
  )
}

/** Mock /api/v1/categories */
export async function mockCategories(page: Page, categories = [sampleCategory]) {
  await page.route('**/api/v1/categories', (route) =>
    route.fulfill({ json: { data: categories, error: null } }),
  )
}

/** Mock /api/v1/tags */
export async function mockTags(page: Page, tags = sampleTags) {
  await page.route('**/api/v1/tags', (route) =>
    route.fulfill({ json: { data: tags, error: null } }),
  )
}

/** Mock /api/v1/questions list */
export async function mockQuestions(page: Page, questions = [sampleQuestion]) {
  await page.route('**/api/v1/questions*', (route) => {
    const url = route.request().url()
    if (url.includes('/export')) return route.continue()
    route.fulfill({
      json: {
        data: {
          items: questions,
          meta: { page: 1, per_page: 20, total: questions.length },
        },
        error: null,
      },
    })
  })
}

/** Mock a login POST to return an access token + user */
export async function mockLogin(page: Page, success = true) {
  const header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64')
  const payload = Buffer.from(JSON.stringify({ sub: 'u-1', email: 'admin@example.com', role: 'super_admin', exp: 9999999999 })).toString('base64')
  const token = `${header}.${payload}.sig`
  await page.route('**/api/v1/auth/login', (route) => {
    if (success) {
      route.fulfill({
        json: {
          data: {
            access_token: token,
            token_type: 'Bearer',
            expires_in: 3600,
            user: { ...adminUser, role: 'super_admin' },
          },
          error: null,
        },
      })
    } else {
      route.fulfill({
        status: 401,
        json: { data: null, error: { code: 'INVALID_CREDENTIALS', message: 'Invalid email or password' } },
      })
    }
  })
}
