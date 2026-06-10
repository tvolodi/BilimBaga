# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: uat-exam-configuration-20260609.spec.ts >> Scenario 3: Active exam requires unpublish to edit >> S3-S3-S4: Click Unpublish, confirm — exam returns to Draft
- Location: e2e\uat-temp\uat-exam-configuration-20260609.spec.ts:581:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByRole('button', { name: /назад|back|артқа/i })
Expected: visible
Timeout: 15000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 15000ms
  - waiting for getByRole('button', { name: /назад|back|артқа/i })

```

```yaml
- link "Skip to main content":
  - /url: "#main-content"
- complementary:
  - text: BilimBaga
  - button "Collapse sidebar"
  - navigation "Main navigation":
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
  - text: UAT Admin Super Admin
  - button "Sign out"
- main:
  - navigation "breadcrumb":
    - link "Exams":
      - /url: /admin/exams
    - link "9aec75d5-c379-4529-ab2a-29a57aef4ff4":
      - /url: /admin/exams/9aec75d5-c379-4529-ab2a-29a57aef4ff4
    - text: Edit
  - heading "Edit Exam" [level=1]
  - navigation: 1 Basic Settings 2 Question Rules 3 Assignments 4 Review & Publish
  - text: This operation is only allowed on exams in draft status. Exam title *
  - textbox "Exam title *":
    - /placeholder: Exam title
    - text: UAT Security Assessment
  - text: Description
  - textbox "Description"
  - text: Time limit (minutes)
  - spinbutton "Time limit (minutes)": "30"
  - text: Passing score (%)
  - spinbutton "Passing score (%)": "70"
  - text: Max attempts
  - spinbutton "Max attempts": "2"
  - text: Available from
  - textbox "Available from"
  - textbox [disabled]
  - text: Available until
  - textbox "Available until"
  - textbox [disabled]
  - text: Show answers
  - combobox "Show answers":
    - option "Never"
    - option "After submission" [selected]
    - option "After exam closes"
  - text: On tab switch
  - combobox "On tab switch":
    - option "Do nothing"
    - option "Warn user" [selected]
    - option "Auto-submit"
  - text: Shuffle questions
  - switch "Shuffle questions"
  - text: Shuffle answer options
  - switch "Shuffle answer options"
  - text: Issue certificate on pass
  - switch "Issue certificate on pass" [checked]
  - button "Next"
```

# Test source

```ts
  501 | 
  502 | test.describe('Scenario 3: Active exam requires unpublish to edit', () => {
  503 | 
  504 |   test('S3-S1: Navigate to UAT Security Assessment (Active) edit page', async ({ page }) => {
  505 |     // Get the active exam ID via API
  506 |     const token = await getToken()
  507 |     const ctx = await request.newContext()
  508 |     const listResp = await ctx.get(`${API}/exams?limit=100`, {
  509 |       headers: { Authorization: `Bearer ${token}` }
  510 |     })
  511 |     const listBody = await listResp.json()
  512 |     const exam = (listBody?.data?.items ?? []).find(
  513 |       (e: { title: string; status: string }) => e.title === 'UAT Security Assessment' && e.status === 'active'
  514 |     )
  515 |     await ctx.dispose()
  516 | 
  517 |     expect(exam).toBeTruthy()
  518 |     publishedExamId = exam.id
  519 | 
  520 |     await login(page)
  521 |     await page.goto(`${BASE}/admin/exams/${exam.id}/edit`)
  522 |     await waitForNetworkIdle(page)
  523 |     await expect(page).toHaveURL(/\/admin\/exams\/.+\/edit/)
  524 |   })
  525 | 
  526 |   test('S3-S2: Edit controls are disabled or absent for active exam, Unpublish button visible', async ({ page }) => {
  527 |     const token = await getToken()
  528 |     const ctx = await request.newContext()
  529 |     const listResp = await ctx.get(`${API}/exams?limit=100`, {
  530 |       headers: { Authorization: `Bearer ${token}` }
  531 |     })
  532 |     const listBody = await listResp.json()
  533 |     const exam = (listBody?.data?.items ?? []).find(
  534 |       (e: { title: string; status: string }) => e.title === 'UAT Security Assessment' && e.status === 'active'
  535 |     )
  536 |     await ctx.dispose()
  537 | 
  538 |     await login(page)
  539 |     await page.goto(`${BASE}/admin/exams/${exam.id}/edit`)
  540 |     await waitForNetworkIdle(page)
  541 | 
  542 |     // Navigate to Step 4 (review) where the Unpublish button lives
  543 |     const titleField = page.locator('#title')
  544 |     if (await titleField.isVisible({ timeout: 5_000 }).catch(() => false)) {
  545 |       // Step 1 is visible — advance to step 4 via the next buttons
  546 |       await page.getByRole('button', { name: /далее|next|алға/i }).click()
  547 |       await expect(page.getByRole('button', { name: /назад|back|артқа/i })).toBeVisible({ timeout: 15_000 })
  548 |       await page.getByRole('button', { name: /далее|next|алға/i }).click()
  549 |       await expect(page.getByText(/назначен|assignment/i).first()).toBeVisible({ timeout: 15_000 })
  550 |       await page.getByRole('button', { name: /далее|next|алға/i }).click()
  551 |     }
  552 | 
  553 |     // Step 4 review: Unpublish button should be visible
  554 |     const unpublishBtn = page.getByRole('button', { name: /unpublish|снять с публикации|жариялауды тоқтат/i })
  555 |     const isUnpublishVisible = await unpublishBtn.isVisible({ timeout: 10_000 }).catch(() => false)
  556 | 
  557 |     // Check if edit controls for title etc. are disabled
  558 |     const titleInput = page.locator('#title')
  559 |     const isTitleVisible = await titleInput.isVisible({ timeout: 3_000 }).catch(() => false)
  560 |     let isTitleDisabled = false
  561 |     if (isTitleVisible) {
  562 |       isTitleDisabled = await titleInput.isDisabled().catch(() => false)
  563 |     }
  564 | 
  565 |     // Per AC#5: edit controls should be disabled and Unpublish button should be visible
  566 |     // OBSERVATION: Step 1 form may still be editable (no disabled state in Step1 for active exams)
  567 |     // and the Unpublish button is only on Step 4
  568 |     test.info().annotations.push({
  569 |       type: 'observation',
  570 |       description: JSON.stringify({
  571 |         unpublishButtonVisible: isUnpublishVisible,
  572 |         titleFieldVisible: isTitleVisible,
  573 |         titleFieldDisabled: isTitleDisabled,
  574 |       })
  575 |     })
  576 | 
  577 |     // The Unpublish button MUST be visible (on Step 4)
  578 |     expect(isUnpublishVisible).toBeTruthy()
  579 |   })
  580 | 
  581 |   test('S3-S3-S4: Click Unpublish, confirm — exam returns to Draft', async ({ page }) => {
  582 |     const token = await getToken()
  583 |     const ctx = await request.newContext()
  584 |     const listResp = await ctx.get(`${API}/exams?limit=100`, {
  585 |       headers: { Authorization: `Bearer ${token}` }
  586 |     })
  587 |     const listBody = await listResp.json()
  588 |     const exam = (listBody?.data?.items ?? []).find(
  589 |       (e: { title: string; status: string }) => e.title === 'UAT Security Assessment' && e.status === 'active'
  590 |     )
  591 |     await ctx.dispose()
  592 | 
  593 |     await login(page)
  594 |     await page.goto(`${BASE}/admin/exams/${exam.id}/edit`)
  595 |     await waitForNetworkIdle(page)
  596 | 
  597 |     // Navigate to Step 4
  598 |     const titleField = page.locator('#title')
  599 |     if (await titleField.isVisible({ timeout: 5_000 }).catch(() => false)) {
  600 |       await page.getByRole('button', { name: /далее|next|алға/i }).click()
> 601 |       await expect(page.getByRole('button', { name: /назад|back|артқа/i })).toBeVisible({ timeout: 15_000 })
      |                                                                             ^ Error: expect(locator).toBeVisible() failed
  602 |       await page.getByRole('button', { name: /далее|next|алға/i }).click()
  603 |       await expect(page.getByText(/назначен|assignment/i).first()).toBeVisible({ timeout: 15_000 })
  604 |       await page.getByRole('button', { name: /далее|next|алға/i }).click()
  605 |     }
  606 | 
  607 |     const unpublishBtn = page.getByRole('button', { name: /unpublish|снять с публикации|жариялауды тоқтат/i })
  608 |     await expect(unpublishBtn).toBeVisible({ timeout: 10_000 })
  609 |     await unpublishBtn.click()
  610 | 
  611 |     // Confirmation dialog
  612 |     const dialog = page.getByRole('dialog')
  613 |     await expect(dialog).toBeVisible({ timeout: 8_000 })
  614 |     const confirmBtn = dialog.getByRole('button').filter({ hasText: /unpublish|снять|жариялауды тоқтат|confirm|ок/i }).first()
  615 |     await expect(confirmBtn).toBeVisible({ timeout: 5_000 })
  616 |     await confirmBtn.click()
  617 | 
  618 |     // Should navigate back to edit or exams list
  619 |     await page.waitForLoadState('networkidle')
  620 | 
  621 |     // Verify exam is now Draft via API
  622 |     const token2 = await getToken()
  623 |     const ctx2 = await request.newContext()
  624 |     const examResp = await ctx2.get(`${API}/exams/${exam.id}`, {
  625 |       headers: { Authorization: `Bearer ${token2}` }
  626 |     })
  627 |     const examBody = await examResp.json()
  628 |     await ctx2.dispose()
  629 |     expect(examBody?.data?.status).toBe('draft')
  630 |   })
  631 | 
  632 |   test('S3-S5: Edit title to UAT Security Assessment v2, save', async ({ page }) => {
  633 |     const token = await getToken()
  634 |     const ctx = await request.newContext()
  635 |     const listResp = await ctx.get(`${API}/exams?limit=100`, {
  636 |       headers: { Authorization: `Bearer ${token}` }
  637 |     })
  638 |     const listBody = await listResp.json()
  639 |     const exam = (listBody?.data?.items ?? []).find(
  640 |       (e: { title: string; status: string }) => e.title === 'UAT Security Assessment' && e.status === 'draft'
  641 |     )
  642 |     await ctx.dispose()
  643 | 
  644 |     await login(page)
  645 |     await page.goto(`${BASE}/admin/exams/${exam.id}/edit`)
  646 |     await waitForNetworkIdle(page)
  647 |     await page.waitForSelector('#title', { timeout: 15_000 })
  648 | 
  649 |     await page.fill('#title', 'UAT Security Assessment v2')
  650 |     await page.getByRole('button', { name: /далее|next|алға/i }).click()
  651 |     // Confirm it advanced to step 2 (title was saved successfully)
  652 |     await expect(page.getByRole('button', { name: /назад|back|артқа/i })).toBeVisible({ timeout: 15_000 })
  653 | 
  654 |     // Verify title updated via API
  655 |     const token2 = await getToken()
  656 |     const ctx2 = await request.newContext()
  657 |     const examResp = await ctx2.get(`${API}/exams/${exam.id}`, {
  658 |       headers: { Authorization: `Bearer ${token2}` }
  659 |     })
  660 |     const examBody = await examResp.json()
  661 |     await ctx2.dispose()
  662 |     expect(examBody?.data?.title).toBe('UAT Security Assessment v2')
  663 |   })
  664 | 
  665 |   test('S3-S6: Re-publish exam — status returns to Active with updated title', async ({ page }) => {
  666 |     const token = await getToken()
  667 |     const ctx = await request.newContext()
  668 |     const listResp = await ctx.get(`${API}/exams?limit=100`, {
  669 |       headers: { Authorization: `Bearer ${token}` }
  670 |     })
  671 |     const listBody = await listResp.json()
  672 |     const exam = (listBody?.data?.items ?? []).find(
  673 |       (e: { title: string; status: string }) => e.title === 'UAT Security Assessment v2' && e.status === 'draft'
  674 |     )
  675 |     await ctx.dispose()
  676 | 
  677 |     await login(page)
  678 |     await page.goto(`${BASE}/admin/exams/${exam.id}/edit`)
  679 |     await waitForNetworkIdle(page)
  680 |     await page.waitForSelector('#title', { timeout: 15_000 })
  681 | 
  682 |     // Navigate through all steps to Step 4
  683 |     await page.getByRole('button', { name: /далее|next|алға/i }).click()
  684 |     await expect(page.getByRole('button', { name: /назад|back|артқа/i })).toBeVisible({ timeout: 15_000 })
  685 |     await page.getByRole('button', { name: /далее|next|алға/i }).click()
  686 |     await expect(page.getByText(/назначен|assignment/i).first()).toBeVisible({ timeout: 15_000 })
  687 |     await page.getByRole('button', { name: /далее|next|алға/i }).click()
  688 |     const publishBtn = page.getByRole('button', { name: /опубликов|publish|жариялау/i })
  689 |     await expect(publishBtn).toBeVisible({ timeout: 15_000 })
  690 |     await publishBtn.click()
  691 | 
  692 |     const dialog = page.getByRole('dialog')
  693 |     await expect(dialog).toBeVisible({ timeout: 8_000 })
  694 |     const confirmBtn = dialog.getByRole('button', { name: /опубликов|publish|жариялау/i })
  695 |     await confirmBtn.click()
  696 | 
  697 |     // Wait for success or redirect
  698 |     await page.waitForLoadState('networkidle')
  699 |     await page.waitForTimeout(2000)
  700 | 
  701 |     // Verify via API
```