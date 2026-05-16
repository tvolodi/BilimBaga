# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: grading\ai-grading.spec.ts >> FR-BB73 — Auto-Grading: Grading Queue UI >> AI reasoning toggle shows and hides reasoning text
- Location: e2e\grading\ai-grading.spec.ts:187:3

# Error details

```
Test timeout of 30000ms exceeded.
```

```
Error: locator.click: Test timeout of 30000ms exceeded.
Call log:
  - waiting for getByText('John Doe')

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
          - heading "Manual Grading Queue" [level=1] [ref=e85]
          - paragraph [ref=e86]: Failed to load. Please try again.
```

# Test source

```ts
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
  186 | 
  187 |   test('AI reasoning toggle shows and hides reasoning text', async ({ page }) => {
  188 |     await page.route('**/api/v1/grading/queue', (route) =>
  189 |       route.fulfill({ json: { data: [aiGradedSession], error: null } }),
  190 |     )
  191 |     await page.route('**/api/v1/grading/sessions/sess-1/questions', (route) =>
  192 |       route.fulfill({ json: { data: [aiGradedQuestion], error: null } }),
  193 |     )
  194 | 
  195 |     await page.goto('/admin/grading')
> 196 |     await page.getByText('John Doe').click()
      |                                      ^ Error: locator.click: Test timeout of 30000ms exceeded.
  197 | 
  198 |     // Reasoning is hidden initially — show button visible
  199 |     await expect(page.getByText(/show ai reasoning/i)).toBeVisible()
  200 |     await expect(page.getByText('Answer is correct. Paris is the capital of France.')).not.toBeVisible()
  201 | 
  202 |     // Click show
  203 |     await page.getByText(/show ai reasoning/i).click()
  204 |     await expect(page.getByText('Answer is correct. Paris is the capital of France.')).toBeVisible()
  205 |     await expect(page.getByText(/hide ai reasoning/i)).toBeVisible()
  206 | 
  207 |     // Click hide
  208 |     await page.getByText(/hide ai reasoning/i).click()
  209 |     await expect(page.getByText('Answer is correct. Paris is the capital of France.')).not.toBeVisible()
  210 |   })
  211 | })
  212 | 
```