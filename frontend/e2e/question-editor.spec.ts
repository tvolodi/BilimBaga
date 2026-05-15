import { test, expect } from '@playwright/test'
import { mockRefreshSuccess } from './fixtures/helpers'
import { mockMe, mockTenantConfig, mockCategories, adminUser } from './fixtures/api'

const sampleQuestionDetail = {
  id: 'q-1',
  type: 'single',
  difficulty: 'easy',
  status: 'draft',
  category_id: 'cat-1',
  default_locale: 'en',
  version: 1,
  parent_id: null,
  locale_coverage: ['en'],
  tag_ids: [],
  translations: {
    en: { stem: 'What is H2O?', explanation: null },
  },
  answer_options: [
    {
      id: 'opt-1',
      sort_order: 0,
      is_correct: true,
      likert_weight: null,
      likert_polarity: null,
      translations: { en: { body: 'Water' } },
    },
    {
      id: 'opt-2',
      sort_order: 1,
      is_correct: false,
      likert_weight: null,
      likert_polarity: null,
      translations: { en: { body: 'Fire' } },
    },
  ],
  tags: [],
  created_by: 'u-1',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
}

test.describe('Question Editor — new question', () => {
  test.beforeEach(async ({ page }) => {
    await mockRefreshSuccess(page)
    await mockMe(page, adminUser)
    await mockTenantConfig(page)
    await mockCategories(page)
    await page.route('**/api/v1/tags', (route) =>
      route.fulfill({ json: { data: [], error: null } }),
    )
  })

  test('renders the editor with Save Draft button', async ({ page }) => {
    await page.goto('/admin/questions/new')
    await expect(page.getByRole('button', { name: /save draft/i })).toBeVisible()
  })

  test('shows type selector', async ({ page }) => {
    await page.goto('/admin/questions/new')
    // Type select is present with options
    await expect(page.locator('select').first()).toBeVisible()
  })
})

test.describe('Question Editor — edit existing question', () => {
  test.beforeEach(async ({ page }) => {
    await mockRefreshSuccess(page)
    await mockMe(page, adminUser)
    await mockTenantConfig(page)
    await mockCategories(page)
    await page.route('**/api/v1/tags', (route) =>
      route.fulfill({ json: { data: [], error: null } }),
    )
    await page.route('**/api/v1/questions/q-1', (route) =>
      route.fulfill({ json: { data: sampleQuestionDetail, error: null } }),
    )
  })

  test('pre-populates form with existing question stem', async ({ page }) => {
    await page.goto('/admin/questions/q-1/edit')
    await expect(page.locator('textarea').filter({ hasText: 'What is H2O?' })).toBeVisible()
  })

  test('shows existing answer options', async ({ page }) => {
    await page.goto('/admin/questions/q-1/edit')
    await expect(page.locator('input[value="Water"]')).toBeVisible()
    await expect(page.locator('input[value="Fire"]')).toBeVisible()
  })
})
