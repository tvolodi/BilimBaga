# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: uat-exam-configuration-20260609.spec.ts >> Scenario 4: Archive an exam >> S4-S2-S3: Click Archive button, confirm — exam status changes to Archived
- Location: e2e\uat-temp\uat-exam-configuration-20260609.spec.ts:736:3

# Error details

```
Error: Archive button should be visible on exams list page

expect(received).toBeTruthy()

Received: false
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
          - generic [ref=e86]: Exams
        - generic [ref=e87]:
          - generic [ref=e88]:
            - heading "Exams" [level=1] [ref=e89]
            - link "Create Exam" [ref=e90] [cursor=pointer]:
              - /url: /admin/exams/new
              - button "Create Exam" [ref=e91]:
                - img [ref=e92]
                - text: Create Exam
          - table [ref=e94]:
            - rowgroup [ref=e95]:
              - row "Exam title Status Time limit (minutes) Passing score (%) Actions" [ref=e96]:
                - columnheader "Exam title" [ref=e97]
                - columnheader "Status" [ref=e98]
                - columnheader "Time limit (minutes)" [ref=e99]
                - columnheader "Passing score (%)" [ref=e100]
                - columnheader "Actions" [ref=e101]
            - rowgroup [ref=e102]:
              - row "UAT Unsatisfiable Exam active 60 min 60% Edit Exam Analytics Unpublish" [ref=e103]:
                - cell "UAT Unsatisfiable Exam" [ref=e104]
                - cell "active" [ref=e105]:
                  - generic [ref=e106]: active
                - cell "60 min" [ref=e107]
                - cell "60%" [ref=e108]
                - cell "Edit Exam Analytics Unpublish" [ref=e109]:
                  - generic [ref=e110]:
                    - link "Edit" [ref=e111] [cursor=pointer]:
                      - /url: /admin/exams/cd0bd144-e7e3-4e42-a12d-48e43f4c994e/edit
                      - button "Edit" [ref=e112]:
                        - img [ref=e113]
                    - link "Exam Analytics" [ref=e116] [cursor=pointer]:
                      - /url: /admin/exams/cd0bd144-e7e3-4e42-a12d-48e43f4c994e/analytics
                      - button "Exam Analytics" [ref=e117]:
                        - img [ref=e118]
                        - text: Exam Analytics
                    - button "Unpublish" [ref=e119]
              - row "UAT Unsatisfiable Exam draft 60 min 60% Edit Exam Analytics" [ref=e120]:
                - cell "UAT Unsatisfiable Exam" [ref=e121]
                - cell "draft" [ref=e122]:
                  - generic [ref=e123]: draft
                - cell "60 min" [ref=e124]
                - cell "60%" [ref=e125]
                - cell "Edit Exam Analytics" [ref=e126]:
                  - generic [ref=e127]:
                    - link "Edit" [ref=e128] [cursor=pointer]:
                      - /url: /admin/exams/fb2a9a1f-9ece-46c1-9881-1a3568dbcfca/edit
                      - button "Edit" [ref=e129]:
                        - img [ref=e130]
                    - link "Exam Analytics" [ref=e133] [cursor=pointer]:
                      - /url: /admin/exams/fb2a9a1f-9ece-46c1-9881-1a3568dbcfca/analytics
                      - button "Exam Analytics" [ref=e134]:
                        - img [ref=e135]
                        - text: Exam Analytics
              - row "UAT Unsatisfiable Exam draft 60 min 60% Edit Exam Analytics" [ref=e136]:
                - cell "UAT Unsatisfiable Exam" [ref=e137]
                - cell "draft" [ref=e138]:
                  - generic [ref=e139]: draft
                - cell "60 min" [ref=e140]
                - cell "60%" [ref=e141]
                - cell "Edit Exam Analytics" [ref=e142]:
                  - generic [ref=e143]:
                    - link "Edit" [ref=e144] [cursor=pointer]:
                      - /url: /admin/exams/271158c7-7fe4-4254-b91f-7cbade7bcd7c/edit
                      - button "Edit" [ref=e145]:
                        - img [ref=e146]
                    - link "Exam Analytics" [ref=e149] [cursor=pointer]:
                      - /url: /admin/exams/271158c7-7fe4-4254-b91f-7cbade7bcd7c/analytics
                      - button "Exam Analytics" [ref=e150]:
                        - img [ref=e151]
                        - text: Exam Analytics
              - row "UAT Security Assessment active 30 min 70% Edit Exam Analytics Unpublish" [ref=e152]:
                - cell "UAT Security Assessment" [ref=e153]
                - cell "active" [ref=e154]:
                  - generic [ref=e155]: active
                - cell "30 min" [ref=e156]
                - cell "70%" [ref=e157]
                - cell "Edit Exam Analytics Unpublish" [ref=e158]:
                  - generic [ref=e159]:
                    - link "Edit" [ref=e160] [cursor=pointer]:
                      - /url: /admin/exams/9aec75d5-c379-4529-ab2a-29a57aef4ff4/edit
                      - button "Edit" [ref=e161]:
                        - img [ref=e162]
                    - link "Exam Analytics" [ref=e165] [cursor=pointer]:
                      - /url: /admin/exams/9aec75d5-c379-4529-ab2a-29a57aef4ff4/analytics
                      - button "Exam Analytics" [ref=e166]:
                        - img [ref=e167]
                        - text: Exam Analytics
                    - button "Unpublish" [ref=e168]
              - row "UAT Security Assessment v2 active 30 min 70% Edit Exam Analytics Unpublish" [ref=e169]:
                - cell "UAT Security Assessment v2" [ref=e170]
                - cell "active" [ref=e171]:
                  - generic [ref=e172]: active
                - cell "30 min" [ref=e173]
                - cell "70%" [ref=e174]
                - cell "Edit Exam Analytics Unpublish" [ref=e175]:
                  - generic [ref=e176]:
                    - link "Edit" [ref=e177] [cursor=pointer]:
                      - /url: /admin/exams/d604f4e5-90cb-41b5-b220-ae5e5982910e/edit
                      - button "Edit" [ref=e178]:
                        - img [ref=e179]
                    - link "Exam Analytics" [ref=e182] [cursor=pointer]:
                      - /url: /admin/exams/d604f4e5-90cb-41b5-b220-ae5e5982910e/analytics
                      - button "Exam Analytics" [ref=e183]:
                        - img [ref=e184]
                        - text: Exam Analytics
                    - button "Unpublish" [ref=e185]
              - row "UAT Security Assessment draft 30 min 70% Edit Exam Analytics" [ref=e186]:
                - cell "UAT Security Assessment" [ref=e187]
                - cell "draft" [ref=e188]:
                  - generic [ref=e189]: draft
                - cell "30 min" [ref=e190]
                - cell "70%" [ref=e191]
                - cell "Edit Exam Analytics" [ref=e192]:
                  - generic [ref=e193]:
                    - link "Edit" [ref=e194] [cursor=pointer]:
                      - /url: /admin/exams/99668dae-b182-4343-827e-0066c38fe184/edit
                      - button "Edit" [ref=e195]:
                        - img [ref=e196]
                    - link "Exam Analytics" [ref=e199] [cursor=pointer]:
                      - /url: /admin/exams/99668dae-b182-4343-827e-0066c38fe184/analytics
                      - button "Exam Analytics" [ref=e200]:
                        - img [ref=e201]
                        - text: Exam Analytics
              - row "UAT Security Assessment draft 30 min 70% Edit Exam Analytics" [ref=e202]:
                - cell "UAT Security Assessment" [ref=e203]
                - cell "draft" [ref=e204]:
                  - generic [ref=e205]: draft
                - cell "30 min" [ref=e206]
                - cell "70%" [ref=e207]
                - cell "Edit Exam Analytics" [ref=e208]:
                  - generic [ref=e209]:
                    - link "Edit" [ref=e210] [cursor=pointer]:
                      - /url: /admin/exams/351643ce-21a2-4262-84f3-a4240d9b26b1/edit
                      - button "Edit" [ref=e211]:
                        - img [ref=e212]
                    - link "Exam Analytics" [ref=e215] [cursor=pointer]:
                      - /url: /admin/exams/351643ce-21a2-4262-84f3-a4240d9b26b1/analytics
                      - button "Exam Analytics" [ref=e216]:
                        - img [ref=e217]
                        - text: Exam Analytics
              - row "UAT Security Assessment draft 30 min 70% Edit Exam Analytics" [ref=e218]:
                - cell "UAT Security Assessment" [ref=e219]
                - cell "draft" [ref=e220]:
                  - generic [ref=e221]: draft
                - cell "30 min" [ref=e222]
                - cell "70%" [ref=e223]
                - cell "Edit Exam Analytics" [ref=e224]:
                  - generic [ref=e225]:
                    - link "Edit" [ref=e226] [cursor=pointer]:
                      - /url: /admin/exams/4695a1c3-89bc-45ce-90c8-94c69a20891f/edit
                      - button "Edit" [ref=e227]:
                        - img [ref=e228]
                    - link "Exam Analytics" [ref=e231] [cursor=pointer]:
                      - /url: /admin/exams/4695a1c3-89bc-45ce-90c8-94c69a20891f/analytics
                      - button "Exam Analytics" [ref=e232]:
                        - img [ref=e233]
                        - text: Exam Analytics
              - row "E2E Walkthrough Exam draft 30 min 70% Edit Exam Analytics" [ref=e234]:
                - cell "E2E Walkthrough Exam" [ref=e235]
                - cell "draft" [ref=e236]:
                  - generic [ref=e237]: draft
                - cell "30 min" [ref=e238]
                - cell "70%" [ref=e239]
                - cell "Edit Exam Analytics" [ref=e240]:
                  - generic [ref=e241]:
                    - link "Edit" [ref=e242] [cursor=pointer]:
                      - /url: /admin/exams/d7641a11-3552-4d73-9b31-5e3bd30a94fb/edit
                      - button "Edit" [ref=e243]:
                        - img [ref=e244]
                    - link "Exam Analytics" [ref=e247] [cursor=pointer]:
                      - /url: /admin/exams/d7641a11-3552-4d73-9b31-5e3bd30a94fb/analytics
                      - button "Exam Analytics" [ref=e248]:
                        - img [ref=e249]
                        - text: Exam Analytics
              - row "E2E Wizard No Rules Test active 60 min 70% Edit Exam Analytics Unpublish" [ref=e250]:
                - cell "E2E Wizard No Rules Test" [ref=e251]
                - cell "active" [ref=e252]:
                  - generic [ref=e253]: active
                - cell "60 min" [ref=e254]
                - cell "70%" [ref=e255]
                - cell "Edit Exam Analytics Unpublish" [ref=e256]:
                  - generic [ref=e257]:
                    - link "Edit" [ref=e258] [cursor=pointer]:
                      - /url: /admin/exams/07f54c4d-4d8a-44e4-be6d-340f72eb7a91/edit
                      - button "Edit" [ref=e259]:
                        - img [ref=e260]
                    - link "Exam Analytics" [ref=e263] [cursor=pointer]:
                      - /url: /admin/exams/07f54c4d-4d8a-44e4-be6d-340f72eb7a91/analytics
                      - button "Exam Analytics" [ref=e264]:
                        - img [ref=e265]
                        - text: Exam Analytics
                    - button "Unpublish" [ref=e266]
              - row "E2E Wizard Publish Test draft 60 min 70% Edit Exam Analytics" [ref=e267]:
                - cell "E2E Wizard Publish Test" [ref=e268]
                - cell "draft" [ref=e269]:
                  - generic [ref=e270]: draft
                - cell "60 min" [ref=e271]
                - cell "70%" [ref=e272]
                - cell "Edit Exam Analytics" [ref=e273]:
                  - generic [ref=e274]:
                    - link "Edit" [ref=e275] [cursor=pointer]:
                      - /url: /admin/exams/94f2acc7-5edd-4bdf-aecf-d57cc0c2c677/edit
                      - button "Edit" [ref=e276]:
                        - img [ref=e277]
                    - link "Exam Analytics" [ref=e280] [cursor=pointer]:
                      - /url: /admin/exams/94f2acc7-5edd-4bdf-aecf-d57cc0c2c677/analytics
                      - button "Exam Analytics" [ref=e281]:
                        - img [ref=e282]
                        - text: Exam Analytics
              - row "E2E Wizard Test Exam draft 60 min 70% Edit Exam Analytics" [ref=e283]:
                - cell "E2E Wizard Test Exam" [ref=e284]
                - cell "draft" [ref=e285]:
                  - generic [ref=e286]: draft
                - cell "60 min" [ref=e287]
                - cell "70%" [ref=e288]
                - cell "Edit Exam Analytics" [ref=e289]:
                  - generic [ref=e290]:
                    - link "Edit" [ref=e291] [cursor=pointer]:
                      - /url: /admin/exams/f2dcf6af-1755-48d0-a7c0-4c874a9447a9/edit
                      - button "Edit" [ref=e292]:
                        - img [ref=e293]
                    - link "Exam Analytics" [ref=e296] [cursor=pointer]:
                      - /url: /admin/exams/f2dcf6af-1755-48d0-a7c0-4c874a9447a9/analytics
                      - button "Exam Analytics" [ref=e297]:
                        - img [ref=e298]
                        - text: Exam Analytics
              - row "E2E Wizard Test Exam draft 60 min 70% Edit Exam Analytics" [ref=e299]:
                - cell "E2E Wizard Test Exam" [ref=e300]
                - cell "draft" [ref=e301]:
                  - generic [ref=e302]: draft
                - cell "60 min" [ref=e303]
                - cell "70%" [ref=e304]
                - cell "Edit Exam Analytics" [ref=e305]:
                  - generic [ref=e306]:
                    - link "Edit" [ref=e307] [cursor=pointer]:
                      - /url: /admin/exams/c0eca9be-c48c-4d15-9058-fc0947a417af/edit
                      - button "Edit" [ref=e308]:
                        - img [ref=e309]
                    - link "Exam Analytics" [ref=e312] [cursor=pointer]:
                      - /url: /admin/exams/c0eca9be-c48c-4d15-9058-fc0947a417af/analytics
                      - button "Exam Analytics" [ref=e313]:
                        - img [ref=e314]
                        - text: Exam Analytics
              - row "E2E Wizard Test Exam draft 60 min 70% Edit Exam Analytics" [ref=e315]:
                - cell "E2E Wizard Test Exam" [ref=e316]
                - cell "draft" [ref=e317]:
                  - generic [ref=e318]: draft
                - cell "60 min" [ref=e319]
                - cell "70%" [ref=e320]
                - cell "Edit Exam Analytics" [ref=e321]:
                  - generic [ref=e322]:
                    - link "Edit" [ref=e323] [cursor=pointer]:
                      - /url: /admin/exams/244e4e59-fa76-434f-9c21-5c19cba6ea69/edit
                      - button "Edit" [ref=e324]:
                        - img [ref=e325]
                    - link "Exam Analytics" [ref=e328] [cursor=pointer]:
                      - /url: /admin/exams/244e4e59-fa76-434f-9c21-5c19cba6ea69/analytics
                      - button "Exam Analytics" [ref=e329]:
                        - img [ref=e330]
                        - text: Exam Analytics
              - row "E2E Wizard Test Exam draft 60 min 70% Edit Exam Analytics" [ref=e331]:
                - cell "E2E Wizard Test Exam" [ref=e332]
                - cell "draft" [ref=e333]:
                  - generic [ref=e334]: draft
                - cell "60 min" [ref=e335]
                - cell "70%" [ref=e336]
                - cell "Edit Exam Analytics" [ref=e337]:
                  - generic [ref=e338]:
                    - link "Edit" [ref=e339] [cursor=pointer]:
                      - /url: /admin/exams/1c393be8-2607-497e-9819-6ede53b02a80/edit
                      - button "Edit" [ref=e340]:
                        - img [ref=e341]
                    - link "Exam Analytics" [ref=e344] [cursor=pointer]:
                      - /url: /admin/exams/1c393be8-2607-497e-9819-6ede53b02a80/analytics
                      - button "Exam Analytics" [ref=e345]:
                        - img [ref=e346]
                        - text: Exam Analytics
              - row "E2E Archive Test 1779189826261 archived 60 min 70% Edit Exam Analytics" [ref=e347]:
                - cell "E2E Archive Test 1779189826261" [ref=e348]
                - cell "archived" [ref=e349]:
                  - generic [ref=e350]: archived
                - cell "60 min" [ref=e351]
                - cell "70%" [ref=e352]
                - cell "Edit Exam Analytics" [ref=e353]:
                  - generic [ref=e354]:
                    - link "Edit" [ref=e355] [cursor=pointer]:
                      - /url: /admin/exams/dcdc4b17-fa41-45fc-8215-2ada751ec5ea/edit
                      - button "Edit" [ref=e356]:
                        - img [ref=e357]
                    - link "Exam Analytics" [ref=e360] [cursor=pointer]:
                      - /url: /admin/exams/dcdc4b17-fa41-45fc-8215-2ada751ec5ea/analytics
                      - button "Exam Analytics" [ref=e361]:
                        - img [ref=e362]
                        - text: Exam Analytics
              - row "Планеты Солнечной системы active 60 min 70% Edit Exam Analytics Unpublish" [ref=e363]:
                - cell "Планеты Солнечной системы" [ref=e364]
                - cell "active" [ref=e365]:
                  - generic [ref=e366]: active
                - cell "60 min" [ref=e367]
                - cell "70%" [ref=e368]
                - cell "Edit Exam Analytics Unpublish" [ref=e369]:
                  - generic [ref=e370]:
                    - link "Edit" [ref=e371] [cursor=pointer]:
                      - /url: /admin/exams/1b6c5e21-5694-4902-b037-ac082910a6ac/edit
                      - button "Edit" [ref=e372]:
                        - img [ref=e373]
                    - link "Exam Analytics" [ref=e376] [cursor=pointer]:
                      - /url: /admin/exams/1b6c5e21-5694-4902-b037-ac082910a6ac/analytics
                      - button "Exam Analytics" [ref=e377]:
                        - img [ref=e378]
                        - text: Exam Analytics
                    - button "Unpublish" [ref=e379]
              - row "E2E Walkthrough Exam draft 30 min 70% Edit Exam Analytics" [ref=e380]:
                - cell "E2E Walkthrough Exam" [ref=e381]
                - cell "draft" [ref=e382]:
                  - generic [ref=e383]: draft
                - cell "30 min" [ref=e384]
                - cell "70%" [ref=e385]
                - cell "Edit Exam Analytics" [ref=e386]:
                  - generic [ref=e387]:
                    - link "Edit" [ref=e388] [cursor=pointer]:
                      - /url: /admin/exams/4fa9f7e5-9838-4da2-a24a-d7bab36758e7/edit
                      - button "Edit" [ref=e389]:
                        - img [ref=e390]
                    - link "Exam Analytics" [ref=e393] [cursor=pointer]:
                      - /url: /admin/exams/4fa9f7e5-9838-4da2-a24a-d7bab36758e7/analytics
                      - button "Exam Analytics" [ref=e394]:
                        - img [ref=e395]
                        - text: Exam Analytics
              - row "E2E Wizard No Rules Test active 60 min 70% Edit Exam Analytics Unpublish" [ref=e396]:
                - cell "E2E Wizard No Rules Test" [ref=e397]
                - cell "active" [ref=e398]:
                  - generic [ref=e399]: active
                - cell "60 min" [ref=e400]
                - cell "70%" [ref=e401]
                - cell "Edit Exam Analytics Unpublish" [ref=e402]:
                  - generic [ref=e403]:
                    - link "Edit" [ref=e404] [cursor=pointer]:
                      - /url: /admin/exams/62d917c6-9372-4829-a3db-ca181605461d/edit
                      - button "Edit" [ref=e405]:
                        - img [ref=e406]
                    - link "Exam Analytics" [ref=e409] [cursor=pointer]:
                      - /url: /admin/exams/62d917c6-9372-4829-a3db-ca181605461d/analytics
                      - button "Exam Analytics" [ref=e410]:
                        - img [ref=e411]
                        - text: Exam Analytics
                    - button "Unpublish" [ref=e412]
              - row "E2E Wizard Publish Test draft 60 min 70% Edit Exam Analytics" [ref=e413]:
                - cell "E2E Wizard Publish Test" [ref=e414]
                - cell "draft" [ref=e415]:
                  - generic [ref=e416]: draft
                - cell "60 min" [ref=e417]
                - cell "70%" [ref=e418]
                - cell "Edit Exam Analytics" [ref=e419]:
                  - generic [ref=e420]:
                    - link "Edit" [ref=e421] [cursor=pointer]:
                      - /url: /admin/exams/79f5aeac-e47c-4918-827a-a84087146b95/edit
                      - button "Edit" [ref=e422]:
                        - img [ref=e423]
                    - link "Exam Analytics" [ref=e426] [cursor=pointer]:
                      - /url: /admin/exams/79f5aeac-e47c-4918-827a-a84087146b95/analytics
                      - button "Exam Analytics" [ref=e427]:
                        - img [ref=e428]
                        - text: Exam Analytics
          - generic [ref=e429]:
            - button "Previous" [disabled]
            - generic [ref=e430]: Page 1 of 8
            - button "Next" [ref=e431]
```

# Test source

```ts
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
  702 |     const token2 = await getToken()
  703 |     const ctx2 = await request.newContext()
  704 |     const examResp = await ctx2.get(`${API}/exams/${exam.id}`, {
  705 |       headers: { Authorization: `Bearer ${token2}` }
  706 |     })
  707 |     const examBody = await examResp.json()
  708 |     await ctx2.dispose()
  709 |     expect(examBody?.data?.status).toBe('active')
  710 |     expect(examBody?.data?.title).toBe('UAT Security Assessment v2')
  711 |   })
  712 | })
  713 | 
  714 | // ─── Scenario 4: Archive an Exam ─────────────────────────────────────────────
  715 | 
  716 | test.describe('Scenario 4: Archive an exam', () => {
  717 | 
  718 |   test('S4-S1: Open active exam UAT Security Assessment v2 detail page', async ({ page }) => {
  719 |     const token = await getToken()
  720 |     const ctx = await request.newContext()
  721 |     const listResp = await ctx.get(`${API}/exams?limit=100`, {
  722 |       headers: { Authorization: `Bearer ${token}` }
  723 |     })
  724 |     const listBody = await listResp.json()
  725 |     const exam = (listBody?.data?.items ?? []).find(
  726 |       (e: { title: string; status: string }) => e.title === 'UAT Security Assessment v2' && e.status === 'active'
  727 |     )
  728 |     await ctx.dispose()
  729 | 
  730 |     await login(page)
  731 |     await page.goto(`${BASE}/admin/exams/${exam.id}/edit`)
  732 |     await waitForNetworkIdle(page)
  733 |     await expect(page).toHaveURL(/\/admin\/exams\/.+\/edit/)
  734 |   })
  735 | 
  736 |   test('S4-S2-S3: Click Archive button, confirm — exam status changes to Archived', async ({ page }) => {
  737 |     const token = await getToken()
  738 |     const ctx = await request.newContext()
  739 |     const listResp = await ctx.get(`${API}/exams?limit=100`, {
  740 |       headers: { Authorization: `Bearer ${token}` }
  741 |     })
  742 |     const listBody = await listResp.json()
  743 |     const exam = (listBody?.data?.items ?? []).find(
  744 |       (e: { title: string; status: string }) => e.title === 'UAT Security Assessment v2' && e.status === 'active'
  745 |     )
  746 |     await ctx.dispose()
  747 | 
  748 |     await login(page)
  749 |     await page.goto(`${BASE}/admin/exams/${exam.id}/edit`)
  750 |     await waitForNetworkIdle(page)
  751 | 
  752 |     // Look for Archive button in the exam wizard / list / edit page
  753 |     const archiveBtn = page.getByRole('button', { name: /archive|архив|мұрағат/i })
  754 |     const archiveBtnVisible = await archiveBtn.isVisible({ timeout: 5_000 }).catch(() => false)
  755 | 
  756 |     if (!archiveBtnVisible) {
  757 |       // Also check the Exams list page for an archive action
  758 |       await page.goto(`${BASE}/admin/exams`)
  759 |       await waitForNetworkIdle(page)
  760 |       const archiveBtnList = page.getByRole('button', { name: /archive|архив|мұрағат/i })
  761 |       const archiveBtnListVisible = await archiveBtnList.isVisible({ timeout: 5_000 }).catch(() => false)
  762 | 
  763 |       test.info().annotations.push({
  764 |         type: 'defect',
  765 |         description: 'ARCHIVE BUTTON NOT FOUND — neither on exam edit page nor on exams list. Archive functionality is not exposed in the UI. The backend has an Archive endpoint (DELETE /api/v1/exams/:id) but no frontend UI element triggers it.'
  766 |       })
  767 | 
  768 |       // This test will fail per AC#7 requirement
> 769 |       expect(archiveBtnListVisible, 'Archive button should be visible on exams list page').toBeTruthy()
      |                                                                                            ^ Error: Archive button should be visible on exams list page
  770 |       return
  771 |     }
  772 | 
  773 |     await archiveBtn.click()
  774 |     const dialog = page.getByRole('dialog')
  775 |     await expect(dialog).toBeVisible({ timeout: 8_000 })
  776 |     const confirmBtn = dialog.getByRole('button', { name: /confirm|archive|архив|ок/i }).first()
  777 |     await confirmBtn.click()
  778 |     await page.waitForLoadState('networkidle')
  779 | 
  780 |     // Verify archived via API
  781 |     const token2 = await getToken()
  782 |     const ctx2 = await request.newContext()
  783 |     const examResp = await ctx2.get(`${API}/exams/${exam.id}`, {
  784 |       headers: { Authorization: `Bearer ${token2}` }
  785 |     })
  786 |     const examBody = await examResp.json()
  787 |     await ctx2.dispose()
  788 |     expect(examBody?.data?.status).toBe('archived')
  789 |   })
  790 | 
  791 |   test('S4-check: Archived exam does not appear in assignable exams list', async ({ page }) => {
  792 |     // Get a recently archived exam
  793 |     const token = await getToken()
  794 |     const ctx = await request.newContext()
  795 |     const listResp = await ctx.get(`${API}/exams?limit=100`, {
  796 |       headers: { Authorization: `Bearer ${token}` }
  797 |     })
  798 |     const listBody = await listResp.json()
  799 |     const archivedExam = (listBody?.data?.items ?? []).find(
  800 |       (e: { title: string; status: string }) =>
  801 |         (e.title === 'UAT Security Assessment v2' || e.title === 'UAT Security Assessment') &&
  802 |         e.status === 'archived'
  803 |     )
  804 |     await ctx.dispose()
  805 | 
  806 |     if (!archivedExam) {
  807 |       test.skip(true, 'No archived exam available to check assignable list')
  808 |       return
  809 |     }
  810 | 
  811 |     await login(page)
  812 |     await page.goto(`${BASE}/admin/exams`)
  813 |     await waitForNetworkIdle(page)
  814 | 
  815 |     // The archived exam should either not appear or appear with archived badge
  816 |     // but it should NOT appear as assignable (active exams only are assignable)
  817 |     // Check that the exam assignments page / exam session start does not show archived exams
  818 |     // For this UAT check, we simply verify the exam list shows 'archived' badge or is hidden
  819 |     const examTitleInList = page.getByText(archivedExam.title)
  820 |     const titleVisible = await examTitleInList.first().isVisible({ timeout: 5_000 }).catch(() => false)
  821 | 
  822 |     if (titleVisible) {
  823 |       // If visible, check it shows archived status (not active/assignable)
  824 |       const row = page.locator('tr').filter({ hasText: archivedExam.title })
  825 |       const hasArchivedBadge = await row.locator('[class*="red"]').isVisible({ timeout: 3_000 }).catch(() => false)
  826 |       test.info().annotations.push({
  827 |         type: 'note',
  828 |         description: `Archived exam visible in list: ${titleVisible}, has archived badge: ${hasArchivedBadge}`
  829 |       })
  830 |     }
  831 |   })
  832 | })
  833 | 
```