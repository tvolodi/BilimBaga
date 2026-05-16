# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: grading\ai-grading.spec.ts >> FR-BB73 — Auto-Grading: Question Editor >> auto_grade checkbox is disabled when model_answer is empty
- Location: e2e\grading\ai-grading.spec.ts:74:3

# Error details

```
Error: expect(locator).toBeDisabled() failed

Locator: locator('input[type="checkbox"]')
Expected: disabled
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeDisabled" with timeout 5000ms
  - waiting for locator('input[type="checkbox"]')

```

```yaml
- link "Skip to main content":
  - /url: "#main-content"
- complementary:
  - text: BilimBaga
  - button "Collapse sidebar"
  - navigation:
    - link "Dashboard":
      - /url: /admin
    - link "Users":
      - /url: /admin/users
    - link "Departments":
      - /url: /admin/departments
    - link "Questions":
      - /url: /admin/questions
    - link "Categories":
      - /url: /admin/categories
    - link "Tags":
      - /url: /admin/tags
    - link "Exams":
      - /url: /admin/exams
    - link "Manual Grading":
      - /url: /admin/grading
    - link "Reports":
      - /url: /admin/reports
    - link "Audit Log":
      - /url: /admin/audit
    - link "Settings":
      - /url: /admin/settings/branding
- banner:
  - combobox:
    - option "Қазақша"
    - option "Русский"
    - option "English" [selected]
  - text: Admin User Super Admin
  - button
- main:
  - button "Go back"
  - heading "New Question" [level=1]
  - button "Save Draft"
  - button "EN ○"
  - button "KK ○"
  - button "RU ○"
  - heading "Metadata" [level=2]
  - text: Question Type
  - combobox:
    - option "Single Choice"
    - option "Multiple Choice"
    - option "True / False"
    - option "Short Text" [selected]
    - option "Likert Scale"
  - text: Difficulty
  - combobox:
    - option "Easy"
    - option "Medium" [selected]
    - option "Hard"
  - text: Category
  - button "Select a category"
  - text: Tags
  - textbox "Add tags…"
  - heading "Content — EN" [level=2]
  - text: Question Stem
  - textbox "Enter the question text…"
  - button "+ Add explanation"
  - heading "Preview" [level=3]
  - paragraph: Question stem will appear here…
  - text: Free-text answer…
```

# Test source

```ts
  1   | import { test, expect } from '@playwright/test'
  2   | import { mockRefreshSuccess } from '../fixtures/helpers'
  3   | import { mockMe, mockTenantConfig, mockCategories, adminUser } from '../fixtures/api'
  4   | 
  5   | const shortTextQuestionDetail = {
  6   |   id: 'q-st',
  7   |   type: 'shorttext',
  8   |   difficulty: 'medium',
  9   |   status: 'draft',
  10  |   category_id: 'cat-1',
  11  |   default_locale: 'en',
  12  |   version: 1,
  13  |   parent_id: null,
  14  |   locale_coverage: ['en'],
  15  |   tag_ids: [],
  16  |   translations: {
  17  |     en: { stem: 'What is the capital of France?', explanation: null },
  18  |   },
  19  |   answer_options: [],
  20  |   tags: [],
  21  |   auto_grade: false,
  22  |   model_answer: null,
  23  |   created_by: 'u-1',
  24  |   created_at: '2024-01-01T00:00:00Z',
  25  |   updated_at: '2024-01-01T00:00:00Z',
  26  | }
  27  | 
  28  | const shortTextWithAutoGrade = {
  29  |   ...shortTextQuestionDetail,
  30  |   auto_grade: true,
  31  |   model_answer: 'Paris',
  32  | }
  33  | 
  34  | function setupCommonMocks(page: Parameters<typeof mockRefreshSuccess>[0]) {
  35  |   return Promise.all([
  36  |     mockRefreshSuccess(page),
  37  |     mockMe(page, adminUser),
  38  |     mockTenantConfig(page),
  39  |     mockCategories(page),
  40  |     page.route('**/api/v1/tags', (route) =>
  41  |       route.fulfill({ json: { data: [], error: null } }),
  42  |     ),
  43  |   ])
  44  | }
  45  | 
  46  | // Helper: the type <select> is the one that has a 'shorttext' option value.
  47  | // page.locator('select').first() would match the LocaleSwitcher; use this filter instead.
  48  | function typeSelect(page: Parameters<typeof setupCommonMocks>[0]) {
  49  |   return page.locator('select').filter({ has: page.locator('option[value="shorttext"]') })
  50  | }
  51  | 
  52  | // The model answer textarea placeholder comes from i18n:
  53  | // questionEditor.field.modelAnswerPlaceholder = "Enter the expected answer for AI comparison…"
  54  | const MODEL_ANSWER_PLACEHOLDER = /enter the expected answer for ai comparison/i
  55  | 
  56  | test.describe('FR-BB73 — Auto-Grading: Question Editor', () => {
  57  |   test.beforeEach(async ({ page }) => {
  58  |     await setupCommonMocks(page)
  59  |   })
  60  | 
  61  |   test('does NOT show auto-grading section for non-shorttext question type', async ({ page }) => {
  62  |     await page.goto('/admin/questions/new')
  63  |     // Default type is 'single' — auto-grading section should be absent
  64  |     await expect(page.getByText(/auto.grading/i)).not.toBeVisible()
  65  |   })
  66  | 
  67  |   test('shows auto-grading section when type is switched to shorttext', async ({ page }) => {
  68  |     await page.goto('/admin/questions/new')
  69  |     // Switch type to shorttext
  70  |     await typeSelect(page).selectOption('shorttext')
  71  |     await expect(page.getByText(/auto.grading/i)).toBeVisible()
  72  |   })
  73  | 
  74  |   test('auto_grade checkbox is disabled when model_answer is empty', async ({ page }) => {
  75  |     await page.goto('/admin/questions/new')
  76  |     await typeSelect(page).selectOption('shorttext')
  77  |     const checkbox = page.locator('input[type="checkbox"]')
> 78  |     await expect(checkbox).toBeDisabled()
      |                            ^ Error: expect(locator).toBeDisabled() failed
  79  |   })
  80  | 
  81  |   test('auto_grade checkbox is enabled after entering a model answer', async ({ page }) => {
  82  |     await page.goto('/admin/questions/new')
  83  |     await typeSelect(page).selectOption('shorttext')
  84  |     // Fill model answer
  85  |     await page.getByPlaceholder(MODEL_ANSWER_PLACEHOLDER).fill('Paris is the capital.')
  86  |     const checkbox = page.locator('input[type="checkbox"]')
  87  |     await expect(checkbox).not.toBeDisabled()
  88  |   })
  89  | 
  90  |   test('loads existing shorttext question with auto_grade and model_answer pre-populated', async ({ page }) => {
  91  |     await page.route('**/api/v1/questions/q-st', (route) =>
  92  |       route.fulfill({ json: { data: shortTextWithAutoGrade, error: null } }),
  93  |     )
  94  |     await page.goto('/admin/questions/q-st')
  95  |     // Model answer textarea should contain the loaded value
  96  |     await expect(page.getByPlaceholder(MODEL_ANSWER_PLACEHOLDER)).toHaveValue('Paris')
  97  |     // Checkbox should be checked
  98  |     const checkbox = page.locator('input[type="checkbox"]')
  99  |     await expect(checkbox).toBeChecked()
  100 |   })
  101 | 
  102 |   test('saves shorttext question with auto_grade and model_answer via PUT', async ({ page }) => {
  103 |     await page.route('**/api/v1/questions/q-st', (route) => {
  104 |       if (route.request().method() === 'GET') {
  105 |         return route.fulfill({ json: { data: shortTextQuestionDetail, error: null } })
  106 |       }
  107 |       if (route.request().method() === 'PUT') {
  108 |         const body = route.request().postDataJSON()
  109 |         // Verify auto_grade and model_answer sent correctly
  110 |         if (body?.auto_grade === true && body?.model_answer === 'The capital of France is Paris.') {
  111 |           return route.fulfill({ json: { data: { ...shortTextWithAutoGrade, model_answer: body.model_answer }, error: null } })
  112 |         }
  113 |         return route.fulfill({ status: 400, json: { data: null, error: { code: 'BAD_REQUEST', message: 'invalid payload' } } })
  114 |       }
  115 |       return route.continue()
  116 |     })
  117 | 
  118 |     await page.goto('/admin/questions/q-st')
  119 |     await page.getByPlaceholder(MODEL_ANSWER_PLACEHOLDER).fill('The capital of France is Paris.')
  120 |     await page.locator('input[type="checkbox"]').check()
  121 |     await page.getByRole('button', { name: /save draft/i }).click()
  122 |     // Expect no error toast — save succeeded
  123 |     await expect(page.getByRole('alert')).not.toBeVisible({ timeout: 3000 }).catch(() => {
  124 |       // Some implementations show success toast; that's fine
  125 |     })
  126 |   })
  127 | 
  128 |   test('clearing model_answer unchecks auto_grade', async ({ page }) => {
  129 |     await page.route('**/api/v1/questions/q-st', (route) =>
  130 |       route.fulfill({ json: { data: shortTextWithAutoGrade, error: null } }),
  131 |     )
  132 |     await page.goto('/admin/questions/q-st')
  133 |     // Model answer is 'Paris', checkbox is checked
  134 |     await expect(page.locator('input[type="checkbox"]')).toBeChecked()
  135 |     // Clear the model answer
  136 |     await page.getByPlaceholder(MODEL_ANSWER_PLACEHOLDER).fill('')
  137 |     // Checkbox should now be unchecked (and disabled)
  138 |     await expect(page.locator('input[type="checkbox"]')).not.toBeChecked()
  139 |   })
  140 | })
  141 | 
  142 | test.describe('FR-BB73 — Auto-Grading: Grading Queue UI', () => {
  143 |   test.beforeEach(async ({ page }) => {
  144 |     await setupCommonMocks(page)
  145 |   })
  146 | 
  147 |   const aiGradedSession = {
  148 |     session_id: 'sess-1',
  149 |     assignment_id: 'assign-1',
  150 |     user_id: 'u-1',
  151 |     user_name: 'John Doe',
  152 |     exam_id: 'exam-1',
  153 |     exam_title: 'Geography Quiz',
  154 |     submitted_at: '2024-03-01T10:00:00Z',
  155 |     pending_count: 0,
  156 |     ai_graded_count: 1,
  157 |   }
  158 | 
  159 |   const aiGradedQuestion = {
  160 |     session_question_id: 'sq-1',
  161 |     question_id: 'q-st',
  162 |     stem: 'What is the capital of France?',
  163 |     question_type: 'shorttext',
  164 |     text_answer: 'Paris',
  165 |     grading_status: 'ai_graded',
  166 |     current_score: 0.8,
  167 |     max_score: 1,
  168 |     feedback: null,
  169 |     ai_reasoning: 'Answer is correct. Paris is the capital of France.',
  170 |   }
  171 | 
  172 |   test('shows AI Graded badge for ai_graded questions in grading queue', async ({ page }) => {
  173 |     await page.route('**/api/v1/grading/queue', (route) =>
  174 |       route.fulfill({ json: { data: [aiGradedSession], error: null } }),
  175 |     )
  176 |     await page.route('**/api/v1/grading/sessions/sess-1/questions', (route) =>
  177 |       route.fulfill({ json: { data: [aiGradedQuestion], error: null } }),
  178 |     )
```