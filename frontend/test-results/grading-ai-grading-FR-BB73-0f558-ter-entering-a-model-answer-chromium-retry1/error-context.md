# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: grading\ai-grading.spec.ts >> FR-BB73 — Auto-Grading: Question Editor >> auto_grade checkbox is enabled after entering a model answer
- Location: e2e\grading\ai-grading.spec.ts:81:3

# Error details

```
Test timeout of 30000ms exceeded.
```

```
Error: locator.fill: Test timeout of 30000ms exceeded.
Call log:
  - waiting for getByPlaceholder(/enter the expected answer for ai comparison/i)

```

# Page snapshot

```yaml
- generic [ref=e2]:
  - link "Skip to main content" [ref=e3] [cursor=pointer]:
    - /url: "#main-content"
  - generic [ref=e4]:
    - complementary [ref=e5]:
      - generic [ref=e6]:
        - generic [ref=e7]: BilimBaga
        - button "Collapse sidebar" [ref=e8]:
          - img [ref=e9]
      - navigation [ref=e11]:
        - link "Dashboard" [ref=e12] [cursor=pointer]:
          - /url: /admin
          - img [ref=e13]
          - generic [ref=e18]: Dashboard
        - link "Users" [ref=e19] [cursor=pointer]:
          - /url: /admin/users
          - img [ref=e20]
          - generic [ref=e25]: Users
        - link "Departments" [ref=e26] [cursor=pointer]:
          - /url: /admin/departments
          - img [ref=e27]
          - generic [ref=e31]: Departments
        - link "Questions" [ref=e32] [cursor=pointer]:
          - /url: /admin/questions
          - img [ref=e33]
          - generic [ref=e36]: Questions
        - link "Categories" [ref=e37] [cursor=pointer]:
          - /url: /admin/categories
          - img [ref=e38]
          - generic [ref=e43]: Categories
        - link "Tags" [ref=e44] [cursor=pointer]:
          - /url: /admin/tags
          - img [ref=e45]
          - generic [ref=e48]: Tags
        - link "Exams" [ref=e49] [cursor=pointer]:
          - /url: /admin/exams
          - img [ref=e50]
          - generic [ref=e53]: Exams
        - link "Manual Grading" [ref=e54] [cursor=pointer]:
          - /url: /admin/grading
          - img [ref=e55]
          - generic [ref=e59]: Manual Grading
        - link "Reports" [ref=e60] [cursor=pointer]:
          - /url: /admin/reports
          - img [ref=e61]
          - generic [ref=e62]: Reports
        - link "Audit Log" [ref=e63] [cursor=pointer]:
          - /url: /admin/audit
          - img [ref=e64]
          - generic [ref=e67]: Audit Log
        - link "Settings" [ref=e68] [cursor=pointer]:
          - /url: /admin/settings/branding
          - img [ref=e69]
          - generic [ref=e72]: Settings
    - generic [ref=e73]:
      - banner [ref=e74]:
        - generic [ref=e75]:
          - combobox [ref=e76]:
            - option "Қазақша"
            - option "Русский"
            - option "English" [selected]
          - generic [ref=e77]: Admin User
          - generic [ref=e78]: Super Admin
          - button [ref=e79]:
            - img [ref=e80]
      - main [ref=e83]:
        - generic [ref=e84]:
          - generic [ref=e85]:
            - button "Go back" [ref=e86]:
              - img [ref=e87]
            - heading "New Question" [level=1] [ref=e89]
            - button "Save Draft" [ref=e90]
          - generic [ref=e92]:
            - button "EN ○" [ref=e93]:
              - generic [ref=e94]: EN
              - generic "Translation missing" [ref=e95]: ○
            - button "KK ○" [ref=e96]:
              - generic [ref=e97]: KK
              - generic "Translation missing" [ref=e98]: ○
            - button "RU ○" [ref=e99]:
              - generic [ref=e100]: RU
              - generic "Translation missing" [ref=e101]: ○
          - generic [ref=e102]:
            - generic [ref=e103]:
              - generic [ref=e104]:
                - heading "Metadata" [level=2] [ref=e105]
                - generic [ref=e106]:
                  - text: Question Type
                  - combobox [ref=e107]:
                    - option "Single Choice"
                    - option "Multiple Choice"
                    - option "True / False"
                    - option "Short Text" [selected]
                    - option "Likert Scale"
                - generic [ref=e108]:
                  - text: Difficulty
                  - combobox [ref=e109]:
                    - option "Easy"
                    - option "Medium" [selected]
                    - option "Hard"
                - generic [ref=e110]:
                  - text: Category
                  - button "Select a category" [ref=e112]:
                    - generic [ref=e113]: Select a category
                    - img [ref=e114]
                - generic [ref=e116]:
                  - text: Tags
                  - textbox "Add tags…" [ref=e119]
              - generic [ref=e120]:
                - heading "Content — EN" [level=2] [ref=e121]
                - generic [ref=e122]:
                  - text: Question Stem
                  - textbox "Enter the question text…" [ref=e123]
                - button "+ Add explanation" [ref=e125]
            - generic [ref=e127]:
              - heading "Preview" [level=3] [ref=e128]
              - generic [ref=e129]:
                - paragraph [ref=e130]: Question stem will appear here…
                - generic [ref=e131]: Free-text answer…
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
  78  |     await expect(checkbox).toBeDisabled()
  79  |   })
  80  | 
  81  |   test('auto_grade checkbox is enabled after entering a model answer', async ({ page }) => {
  82  |     await page.goto('/admin/questions/new')
  83  |     await typeSelect(page).selectOption('shorttext')
  84  |     // Fill model answer
> 85  |     await page.getByPlaceholder(MODEL_ANSWER_PLACEHOLDER).fill('Paris is the capital.')
      |                                                           ^ Error: locator.fill: Test timeout of 30000ms exceeded.
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
  179 | 
  180 |     await page.goto('/admin/grading')
  181 |     // Click into the session
  182 |     await page.getByText('John Doe').click()
  183 |     // Should show AI Graded badge
  184 |     await expect(page.getByText(/ai graded/i)).toBeVisible()
  185 |   })
```