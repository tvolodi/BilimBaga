# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: uat-exam-configuration-20260609.spec.ts >> Scenario 1: Create, Configure and Publish >> S1-S13: Review step shows correct summary (title, 30 min, 70%, 2 attempts)
- Location: e2e\uat-temp\uat-exam-configuration-20260609.spec.ts:267:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByText('2')
Expected: visible
Error: strict mode violation: getByText('2') resolved to 2 elements:
    1) <div class="flex items-center justify-center w-8 h-8 rounded-full text-sm font-medium transition-colors bg-primary/20 text-primary">2</div> aka locator('div').filter({ hasText: /^2$/ })
    2) <span class="text-sm font-medium text-right">2</span> aka locator('span').filter({ hasText: '2' })

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for getByText('2')

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
      - navigation "Main navigation" [ref=e11]:
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
          - generic [ref=e77]: UAT Admin
          - generic [ref=e78]: Super Admin
          - button "Sign out" [ref=e79]:
            - img [ref=e80]
      - main [ref=e83]:
        - navigation "breadcrumb" [ref=e84]:
          - link "Exams" [ref=e86] [cursor=pointer]:
            - /url: /admin/exams
          - generic [ref=e87]:
            - img [ref=e88]
            - generic [ref=e90]: New
        - generic [ref=e91]:
          - heading "Create Exam" [level=1] [ref=e92]
          - navigation [ref=e93]:
            - generic [ref=e94]:
              - generic [ref=e95]: "1"
              - generic [ref=e96]: Basic Settings
            - generic [ref=e98]:
              - generic [ref=e99]: "2"
              - generic [ref=e100]: Question Rules
            - generic [ref=e102]:
              - generic [ref=e103]: "3"
              - generic [ref=e104]: Assignments
            - generic [ref=e106]:
              - generic [ref=e107]: "4"
              - generic [ref=e108]: Review & Publish
          - generic [ref=e109]:
            - heading "Review & Publish" [level=2] [ref=e110]
            - generic [ref=e111]:
              - heading "Basic Settings" [level=3] [ref=e112]
              - generic [ref=e113]:
                - generic [ref=e114]: Exam title
                - generic [ref=e115]: UAT Security Assessment
              - generic [ref=e116]:
                - generic [ref=e117]: Description
                - generic [ref=e118]: —
              - generic [ref=e119]:
                - generic [ref=e120]: Time limit (minutes)
                - generic [ref=e121]: 30 min
              - generic [ref=e122]:
                - generic [ref=e123]: Passing score (%)
                - generic [ref=e124]: 70%
              - generic [ref=e125]:
                - generic [ref=e126]: Max attempts
                - generic [ref=e127]: "2"
              - generic [ref=e128]:
                - generic [ref=e129]: Available from
                - generic [ref=e130]: —
              - generic [ref=e131]:
                - generic [ref=e132]: Available until
                - generic [ref=e133]: —
              - generic [ref=e134]:
                - generic [ref=e135]: Shuffle questions
                - generic [ref=e136]: —
              - generic [ref=e137]:
                - generic [ref=e138]: Shuffle answer options
                - generic [ref=e139]: —
              - generic [ref=e140]:
                - generic [ref=e141]: Show answers
                - generic [ref=e142]: Never
              - generic [ref=e143]:
                - generic [ref=e144]: On tab switch
                - generic [ref=e145]: Do nothing
              - generic [ref=e146]:
                - generic [ref=e147]: Issue certificate on pass
                - generic [ref=e148]: —
            - generic [ref=e149]:
              - heading "Question Rules" [level=3] [ref=e150]
              - list [ref=e151]:
                - listitem [ref=e152]:
                  - generic [ref=e153]: "Rule 1: manual, 3 questions, medium"
                  - generic [ref=e154]: 0 eligible
            - generic [ref=e155]:
              - button "Back" [ref=e156]
              - button "Publish" [ref=e157]
```

# Test source

```ts
  201 |     })
  202 |     await advanceToStep2(page)
  203 | 
  204 |     // Click "Add rule" button
  205 |     const addRuleBtn = page.getByRole('button', { name: /add rule|добавить правило|ереже қосу|правило/i })
  206 |     await expect(addRuleBtn).toBeVisible({ timeout: 10_000 })
  207 |     await addRuleBtn.click()
  208 | 
  209 |     // A rule row should appear; set mode to Random
  210 |     await page.waitForSelector('.border.rounded-lg', { timeout: 10_000 })
  211 | 
  212 |     // Toggle to Random mode (click random button in rule row)
  213 |     const randomBtn = page.locator('.border.rounded-lg').last().getByRole('button', { name: /random|случайн|кездейсоқ/i })
  214 |     if (await randomBtn.isVisible({ timeout: 3_000 }).catch(() => false)) {
  215 |       await randomBtn.click()
  216 |     }
  217 | 
  218 |     // Select category UAT Security
  219 |     const categorySelect = page.locator('.border.rounded-lg').last().locator('select').first()
  220 |     await categorySelect.selectOption({ value: UAT_SECURITY_CATEGORY_ID })
  221 | 
  222 |     // Select difficulty Medium
  223 |     const difficultySelect = page.locator('.border.rounded-lg').last().locator('select').nth(1)
  224 |     await difficultySelect.selectOption({ value: 'medium' })
  225 | 
  226 |     // Set count to 3
  227 |     const countInput = page.locator('.border.rounded-lg').last().locator('input[type="number"]')
  228 |     await countInput.fill('3')
  229 | 
  230 |     // Rule row should be visible
  231 |     await expect(page.locator('.border.rounded-lg').last()).toBeVisible()
  232 |   })
  233 | 
  234 |   test('S1-S12: Advance to Step 3 and then Step 4 (review step)', async ({ page }) => {
  235 |     await login(page)
  236 |     await fillStep1(page, {
  237 |       title: 'UAT Security Assessment',
  238 |       timeLimitMinutes: '30',
  239 |       passingScorePct: '70',
  240 |       maxAttempts: '2',
  241 |     })
  242 |     await advanceToStep2(page)
  243 | 
  244 |     // Add rule
  245 |     const addRuleBtn = page.getByRole('button', { name: /add rule|добавить правило|ереже қосу|правило/i })
  246 |     await addRuleBtn.click()
  247 |     await page.waitForSelector('.border.rounded-lg', { timeout: 10_000 })
  248 |     const randomBtn = page.locator('.border.rounded-lg').last().getByRole('button', { name: /random|случайн|кездейсоқ/i })
  249 |     if (await randomBtn.isVisible({ timeout: 3_000 }).catch(() => false)) {
  250 |       await randomBtn.click()
  251 |     }
  252 |     const categorySelect = page.locator('.border.rounded-lg').last().locator('select').first()
  253 |     await categorySelect.selectOption({ value: UAT_SECURITY_CATEGORY_ID })
  254 |     const difficultySelect = page.locator('.border.rounded-lg').last().locator('select').nth(1)
  255 |     await difficultySelect.selectOption({ value: 'medium' })
  256 |     const countInput = page.locator('.border.rounded-lg').last().locator('input[type="number"]')
  257 |     await countInput.fill('3')
  258 | 
  259 |     // Wait a moment for the save button to appear/be active
  260 |     await advanceToStep3(page)
  261 |     await advanceToStep4(page)
  262 | 
  263 |     // Step 4 heading visible
  264 |     await expect(page.getByText(/review|обзор|шолу|publish/i).first()).toBeVisible({ timeout: 10_000 })
  265 |   })
  266 | 
  267 |   test('S1-S13: Review step shows correct summary (title, 30 min, 70%, 2 attempts)', async ({ page }) => {
  268 |     await login(page)
  269 |     await fillStep1(page, {
  270 |       title: 'UAT Security Assessment',
  271 |       timeLimitMinutes: '30',
  272 |       passingScorePct: '70',
  273 |       maxAttempts: '2',
  274 |     })
  275 |     await advanceToStep2(page)
  276 | 
  277 |     const addRuleBtn = page.getByRole('button', { name: /add rule|добавить правило|ереже|правило/i })
  278 |     await addRuleBtn.click()
  279 |     await page.waitForSelector('.border.rounded-lg', { timeout: 10_000 })
  280 |     const randomBtn = page.locator('.border.rounded-lg').last().getByRole('button', { name: /random|случайн|кездейсоқ/i })
  281 |     if (await randomBtn.isVisible({ timeout: 3_000 }).catch(() => false)) {
  282 |       await randomBtn.click()
  283 |     }
  284 |     const categorySelect = page.locator('.border.rounded-lg').last().locator('select').first()
  285 |     await categorySelect.selectOption({ value: UAT_SECURITY_CATEGORY_ID })
  286 |     const difficultySelect = page.locator('.border.rounded-lg').last().locator('select').nth(1)
  287 |     await difficultySelect.selectOption({ value: 'medium' })
  288 |     const countInput = page.locator('.border.rounded-lg').last().locator('input[type="number"]')
  289 |     await countInput.fill('3')
  290 | 
  291 |     await advanceToStep3(page)
  292 |     await advanceToStep4(page)
  293 | 
  294 |     // Review should show the title
  295 |     await expect(page.getByText('UAT Security Assessment')).toBeVisible({ timeout: 10_000 })
  296 |     // Time limit 30 min
  297 |     await expect(page.getByText(/30 min/i)).toBeVisible({ timeout: 5_000 })
  298 |     // Passing score 70%
  299 |     await expect(page.getByText(/70%/)).toBeVisible({ timeout: 5_000 })
  300 |     // Max attempts 2
> 301 |     await expect(page.getByText('2')).toBeVisible({ timeout: 5_000 })
      |                                       ^ Error: expect(locator).toBeVisible() failed
  302 |   })
  303 | 
  304 |   test('S1-S14: Publish exam — status becomes Active, success message shown', async ({ page }) => {
  305 |     await login(page)
  306 |     await fillStep1(page, {
  307 |       title: 'UAT Security Assessment',
  308 |       timeLimitMinutes: '30',
  309 |       passingScorePct: '70',
  310 |       maxAttempts: '2',
  311 |       showAnswers: 'after_completion',
  312 |       onTabSwitch: 'warn',
  313 |       certificateEnabled: true,
  314 |     })
  315 |     await advanceToStep2(page)
  316 | 
  317 |     const addRuleBtn = page.getByRole('button', { name: /add rule|добавить правило|ереже|правило/i })
  318 |     await addRuleBtn.click()
  319 |     await page.waitForSelector('.border.rounded-lg', { timeout: 10_000 })
  320 |     const randomBtn = page.locator('.border.rounded-lg').last().getByRole('button', { name: /random|случайн|кездейсоқ/i })
  321 |     if (await randomBtn.isVisible({ timeout: 3_000 }).catch(() => false)) {
  322 |       await randomBtn.click()
  323 |     }
  324 |     const categorySelect = page.locator('.border.rounded-lg').last().locator('select').first()
  325 |     await categorySelect.selectOption({ value: UAT_SECURITY_CATEGORY_ID })
  326 |     const difficultySelect = page.locator('.border.rounded-lg').last().locator('select').nth(1)
  327 |     await difficultySelect.selectOption({ value: 'medium' })
  328 |     const countInput = page.locator('.border.rounded-lg').last().locator('input[type="number"]')
  329 |     await countInput.fill('3')
  330 | 
  331 |     await advanceToStep3(page)
  332 |     await advanceToStep4(page)
  333 | 
  334 |     // Click Publish
  335 |     const publishBtn = page.getByRole('button', { name: /опубликов|publish|жариялау/i })
  336 |     await publishBtn.click()
  337 | 
  338 |     // Confirmation dialog appears
  339 |     const dialog = page.getByRole('dialog')
  340 |     await expect(dialog).toBeVisible({ timeout: 10_000 })
  341 |     const confirmBtn = dialog.getByRole('button', { name: /опубликов|publish|жариялау/i })
  342 |     await confirmBtn.click()
  343 | 
  344 |     // Success banner or redirect
  345 |     const successBanner = page.locator('.bg-green-50, [class*="green"]').filter({ hasText: /success|success|опублик|жарияланды/i })
  346 |     const redirected = page.waitForURL(/\/admin\/exams$/, { timeout: 10_000 }).then(() => true).catch(() => false)
  347 | 
  348 |     const bannerVisible = await successBanner.isVisible({ timeout: 10_000 }).catch(() => false)
  349 |     const didRedirect = await redirected
  350 | 
  351 |     // Either a success banner is shown OR we get redirected to the exams list
  352 |     expect(bannerVisible || didRedirect).toBeTruthy()
  353 | 
  354 |     // Save exam ID for subsequent scenarios
  355 |     const currentUrl = page.url()
  356 |     const match = currentUrl.match(/\/admin\/exams\/([^/]+)/)
  357 |     if (match) {
  358 |       publishedExamId = match[1]
  359 |     }
  360 | 
  361 |     // Verify exam is active via API
  362 |     const token = await getToken()
  363 |     const ctx = await request.newContext()
  364 |     const listResp = await ctx.get(`${API}/exams?limit=100`, {
  365 |       headers: { Authorization: `Bearer ${token}` }
  366 |     })
  367 |     const listBody = await listResp.json()
  368 |     const exam = (listBody?.data?.items ?? []).find((e: { title: string; status: string }) => e.title === 'UAT Security Assessment')
  369 |     await ctx.dispose()
  370 |     expect(exam?.status).toBe('active')
  371 |   })
  372 | })
  373 | 
  374 | // ─── Scenario 2: Publish Fails When Rule Cannot Be Satisfied ────────────────
  375 | 
  376 | test.describe('Scenario 2: Unsatisfiable rule prevents publish', () => {
  377 | 
  378 |   test('S2-S1: Create a new exam titled UAT Unsatisfiable Exam', async ({ page }) => {
  379 |     await login(page)
  380 |     await fillStep1(page, {
  381 |       title: 'UAT Unsatisfiable Exam',
  382 |       timeLimitMinutes: '60',
  383 |       passingScorePct: '60',
  384 |       maxAttempts: '1',
  385 |     })
  386 |     await advanceToStep2(page)
  387 |     await expect(page.getByText(/правила вопросов|question rules|сұрақ ережелері/i).first()).toBeVisible({ timeout: 10_000 })
  388 |   })
  389 | 
  390 |   test('S2-S2: Add unsatisfiable rule: UAT Security, Hard, count 50 — warning shown', async ({ page }) => {
  391 |     await login(page)
  392 |     await fillStep1(page, {
  393 |       title: 'UAT Unsatisfiable Exam',
  394 |       timeLimitMinutes: '60',
  395 |       passingScorePct: '60',
  396 |       maxAttempts: '1',
  397 |     })
  398 |     await advanceToStep2(page)
  399 | 
  400 |     const addRuleBtn = page.getByRole('button', { name: /add rule|добавить правило|ереже|правило/i })
  401 |     await addRuleBtn.click()
```