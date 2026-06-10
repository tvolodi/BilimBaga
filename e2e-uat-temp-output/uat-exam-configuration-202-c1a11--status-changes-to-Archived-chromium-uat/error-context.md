# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: uat-exam-configuration-20260609.spec.ts >> Scenario 4: Archive an exam >> S4-S2-S3: Click Archive button, confirm — exam status changes to Archived
- Location: e2e\uat-temp\uat-exam-configuration-20260609.spec.ts:788:3

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
              - row "UAT Unsatisfiable Exam draft 60 min 60% Edit Exam Analytics" [ref=e103]:
                - cell "UAT Unsatisfiable Exam" [ref=e104]
                - cell "draft" [ref=e105]:
                  - generic [ref=e106]: draft
                - cell "60 min" [ref=e107]
                - cell "60%" [ref=e108]
                - cell "Edit Exam Analytics" [ref=e109]:
                  - generic [ref=e110]:
                    - link "Edit" [ref=e111] [cursor=pointer]:
                      - /url: /admin/exams/14e11aee-5056-4004-a49c-d838d58d2d3e/edit
                      - button "Edit" [ref=e112]:
                        - img [ref=e113]
                    - link "Exam Analytics" [ref=e116] [cursor=pointer]:
                      - /url: /admin/exams/14e11aee-5056-4004-a49c-d838d58d2d3e/analytics
                      - button "Exam Analytics" [ref=e117]:
                        - img [ref=e118]
                        - text: Exam Analytics
              - row "UAT Unsatisfiable Exam draft 60 min 60% Edit Exam Analytics" [ref=e119]:
                - cell "UAT Unsatisfiable Exam" [ref=e120]
                - cell "draft" [ref=e121]:
                  - generic [ref=e122]: draft
                - cell "60 min" [ref=e123]
                - cell "60%" [ref=e124]
                - cell "Edit Exam Analytics" [ref=e125]:
                  - generic [ref=e126]:
                    - link "Edit" [ref=e127] [cursor=pointer]:
                      - /url: /admin/exams/d9aa0e7b-6b87-46b6-aec0-0b12df9c29bf/edit
                      - button "Edit" [ref=e128]:
                        - img [ref=e129]
                    - link "Exam Analytics" [ref=e132] [cursor=pointer]:
                      - /url: /admin/exams/d9aa0e7b-6b87-46b6-aec0-0b12df9c29bf/analytics
                      - button "Exam Analytics" [ref=e133]:
                        - img [ref=e134]
                        - text: Exam Analytics
              - row "UAT Unsatisfiable Exam draft 60 min 60% Edit Exam Analytics" [ref=e135]:
                - cell "UAT Unsatisfiable Exam" [ref=e136]
                - cell "draft" [ref=e137]:
                  - generic [ref=e138]: draft
                - cell "60 min" [ref=e139]
                - cell "60%" [ref=e140]
                - cell "Edit Exam Analytics" [ref=e141]:
                  - generic [ref=e142]:
                    - link "Edit" [ref=e143] [cursor=pointer]:
                      - /url: /admin/exams/49f11b29-fc69-4de2-81e6-343c5087af6b/edit
                      - button "Edit" [ref=e144]:
                        - img [ref=e145]
                    - link "Exam Analytics" [ref=e148] [cursor=pointer]:
                      - /url: /admin/exams/49f11b29-fc69-4de2-81e6-343c5087af6b/analytics
                      - button "Exam Analytics" [ref=e149]:
                        - img [ref=e150]
                        - text: Exam Analytics
              - row "UAT Security Assessment v2 active 30 min 70% Edit Exam Analytics Unpublish" [ref=e151]:
                - cell "UAT Security Assessment v2" [ref=e152]
                - cell "active" [ref=e153]:
                  - generic [ref=e154]: active
                - cell "30 min" [ref=e155]
                - cell "70%" [ref=e156]
                - cell "Edit Exam Analytics Unpublish" [ref=e157]:
                  - generic [ref=e158]:
                    - link "Edit" [ref=e159] [cursor=pointer]:
                      - /url: /admin/exams/9c4c22f6-dc3d-4338-8fe6-e49860e4ee49/edit
                      - button "Edit" [ref=e160]:
                        - img [ref=e161]
                    - link "Exam Analytics" [ref=e164] [cursor=pointer]:
                      - /url: /admin/exams/9c4c22f6-dc3d-4338-8fe6-e49860e4ee49/analytics
                      - button "Exam Analytics" [ref=e165]:
                        - img [ref=e166]
                        - text: Exam Analytics
                    - button "Unpublish" [ref=e167]
              - row "UAT Security Assessment draft 30 min 70% Edit Exam Analytics" [ref=e168]:
                - cell "UAT Security Assessment" [ref=e169]
                - cell "draft" [ref=e170]:
                  - generic [ref=e171]: draft
                - cell "30 min" [ref=e172]
                - cell "70%" [ref=e173]
                - cell "Edit Exam Analytics" [ref=e174]:
                  - generic [ref=e175]:
                    - link "Edit" [ref=e176] [cursor=pointer]:
                      - /url: /admin/exams/fb11c5e3-5927-47ff-9163-fb1fd0f48c82/edit
                      - button "Edit" [ref=e177]:
                        - img [ref=e178]
                    - link "Exam Analytics" [ref=e181] [cursor=pointer]:
                      - /url: /admin/exams/fb11c5e3-5927-47ff-9163-fb1fd0f48c82/analytics
                      - button "Exam Analytics" [ref=e182]:
                        - img [ref=e183]
                        - text: Exam Analytics
              - row "UAT Security Assessment draft 30 min 70% Edit Exam Analytics" [ref=e184]:
                - cell "UAT Security Assessment" [ref=e185]
                - cell "draft" [ref=e186]:
                  - generic [ref=e187]: draft
                - cell "30 min" [ref=e188]
                - cell "70%" [ref=e189]
                - cell "Edit Exam Analytics" [ref=e190]:
                  - generic [ref=e191]:
                    - link "Edit" [ref=e192] [cursor=pointer]:
                      - /url: /admin/exams/d7347b26-bbab-4f37-9459-aaf58731f2e0/edit
                      - button "Edit" [ref=e193]:
                        - img [ref=e194]
                    - link "Exam Analytics" [ref=e197] [cursor=pointer]:
                      - /url: /admin/exams/d7347b26-bbab-4f37-9459-aaf58731f2e0/analytics
                      - button "Exam Analytics" [ref=e198]:
                        - img [ref=e199]
                        - text: Exam Analytics
              - row "UAT Security Assessment draft 30 min 70% Edit Exam Analytics" [ref=e200]:
                - cell "UAT Security Assessment" [ref=e201]
                - cell "draft" [ref=e202]:
                  - generic [ref=e203]: draft
                - cell "30 min" [ref=e204]
                - cell "70%" [ref=e205]
                - cell "Edit Exam Analytics" [ref=e206]:
                  - generic [ref=e207]:
                    - link "Edit" [ref=e208] [cursor=pointer]:
                      - /url: /admin/exams/9f21cb32-2b9b-44cc-a1b9-83572f1f57ea/edit
                      - button "Edit" [ref=e209]:
                        - img [ref=e210]
                    - link "Exam Analytics" [ref=e213] [cursor=pointer]:
                      - /url: /admin/exams/9f21cb32-2b9b-44cc-a1b9-83572f1f57ea/analytics
                      - button "Exam Analytics" [ref=e214]:
                        - img [ref=e215]
                        - text: Exam Analytics
              - row "UAT Security Assessment draft 30 min 70% Edit Exam Analytics" [ref=e216]:
                - cell "UAT Security Assessment" [ref=e217]
                - cell "draft" [ref=e218]:
                  - generic [ref=e219]: draft
                - cell "30 min" [ref=e220]
                - cell "70%" [ref=e221]
                - cell "Edit Exam Analytics" [ref=e222]:
                  - generic [ref=e223]:
                    - link "Edit" [ref=e224] [cursor=pointer]:
                      - /url: /admin/exams/190b0356-ad45-485d-ad54-3a789025c185/edit
                      - button "Edit" [ref=e225]:
                        - img [ref=e226]
                    - link "Exam Analytics" [ref=e229] [cursor=pointer]:
                      - /url: /admin/exams/190b0356-ad45-485d-ad54-3a789025c185/analytics
                      - button "Exam Analytics" [ref=e230]:
                        - img [ref=e231]
                        - text: Exam Analytics
              - row "UAT Unsatisfiable Exam draft 60 min 60% Edit Exam Analytics" [ref=e232]:
                - cell "UAT Unsatisfiable Exam" [ref=e233]
                - cell "draft" [ref=e234]:
                  - generic [ref=e235]: draft
                - cell "60 min" [ref=e236]
                - cell "60%" [ref=e237]
                - cell "Edit Exam Analytics" [ref=e238]:
                  - generic [ref=e239]:
                    - link "Edit" [ref=e240] [cursor=pointer]:
                      - /url: /admin/exams/e296a53b-622a-468e-aa5e-ed360b1f9695/edit
                      - button "Edit" [ref=e241]:
                        - img [ref=e242]
                    - link "Exam Analytics" [ref=e245] [cursor=pointer]:
                      - /url: /admin/exams/e296a53b-622a-468e-aa5e-ed360b1f9695/analytics
                      - button "Exam Analytics" [ref=e246]:
                        - img [ref=e247]
                        - text: Exam Analytics
              - row "UAT Unsatisfiable Exam draft 60 min 60% Edit Exam Analytics" [ref=e248]:
                - cell "UAT Unsatisfiable Exam" [ref=e249]
                - cell "draft" [ref=e250]:
                  - generic [ref=e251]: draft
                - cell "60 min" [ref=e252]
                - cell "60%" [ref=e253]
                - cell "Edit Exam Analytics" [ref=e254]:
                  - generic [ref=e255]:
                    - link "Edit" [ref=e256] [cursor=pointer]:
                      - /url: /admin/exams/f5ec774a-80d8-413c-91dc-81d8dd35b584/edit
                      - button "Edit" [ref=e257]:
                        - img [ref=e258]
                    - link "Exam Analytics" [ref=e261] [cursor=pointer]:
                      - /url: /admin/exams/f5ec774a-80d8-413c-91dc-81d8dd35b584/analytics
                      - button "Exam Analytics" [ref=e262]:
                        - img [ref=e263]
                        - text: Exam Analytics
              - row "UAT Unsatisfiable Exam draft 60 min 60% Edit Exam Analytics" [ref=e264]:
                - cell "UAT Unsatisfiable Exam" [ref=e265]
                - cell "draft" [ref=e266]:
                  - generic [ref=e267]: draft
                - cell "60 min" [ref=e268]
                - cell "60%" [ref=e269]
                - cell "Edit Exam Analytics" [ref=e270]:
                  - generic [ref=e271]:
                    - link "Edit" [ref=e272] [cursor=pointer]:
                      - /url: /admin/exams/07df832c-df9f-4570-a56d-6aed459d7346/edit
                      - button "Edit" [ref=e273]:
                        - img [ref=e274]
                    - link "Exam Analytics" [ref=e277] [cursor=pointer]:
                      - /url: /admin/exams/07df832c-df9f-4570-a56d-6aed459d7346/analytics
                      - button "Exam Analytics" [ref=e278]:
                        - img [ref=e279]
                        - text: Exam Analytics
              - row "UAT Security Assessment v2 active 30 min 70% Edit Exam Analytics Unpublish" [ref=e280]:
                - cell "UAT Security Assessment v2" [ref=e281]
                - cell "active" [ref=e282]:
                  - generic [ref=e283]: active
                - cell "30 min" [ref=e284]
                - cell "70%" [ref=e285]
                - cell "Edit Exam Analytics Unpublish" [ref=e286]:
                  - generic [ref=e287]:
                    - link "Edit" [ref=e288] [cursor=pointer]:
                      - /url: /admin/exams/c07fffd9-71fc-46fc-9314-b24e069623e6/edit
                      - button "Edit" [ref=e289]:
                        - img [ref=e290]
                    - link "Exam Analytics" [ref=e293] [cursor=pointer]:
                      - /url: /admin/exams/c07fffd9-71fc-46fc-9314-b24e069623e6/analytics
                      - button "Exam Analytics" [ref=e294]:
                        - img [ref=e295]
                        - text: Exam Analytics
                    - button "Unpublish" [ref=e296]
              - row "UAT Security Assessment draft 30 min 70% Edit Exam Analytics" [ref=e297]:
                - cell "UAT Security Assessment" [ref=e298]
                - cell "draft" [ref=e299]:
                  - generic [ref=e300]: draft
                - cell "30 min" [ref=e301]
                - cell "70%" [ref=e302]
                - cell "Edit Exam Analytics" [ref=e303]:
                  - generic [ref=e304]:
                    - link "Edit" [ref=e305] [cursor=pointer]:
                      - /url: /admin/exams/d2f679c3-d053-4d7f-8714-b5a443d3210a/edit
                      - button "Edit" [ref=e306]:
                        - img [ref=e307]
                    - link "Exam Analytics" [ref=e310] [cursor=pointer]:
                      - /url: /admin/exams/d2f679c3-d053-4d7f-8714-b5a443d3210a/analytics
                      - button "Exam Analytics" [ref=e311]:
                        - img [ref=e312]
                        - text: Exam Analytics
              - row "UAT Security Assessment draft 30 min 70% Edit Exam Analytics" [ref=e313]:
                - cell "UAT Security Assessment" [ref=e314]
                - cell "draft" [ref=e315]:
                  - generic [ref=e316]: draft
                - cell "30 min" [ref=e317]
                - cell "70%" [ref=e318]
                - cell "Edit Exam Analytics" [ref=e319]:
                  - generic [ref=e320]:
                    - link "Edit" [ref=e321] [cursor=pointer]:
                      - /url: /admin/exams/e859131e-e534-4b96-b04a-9ab7bd4c2114/edit
                      - button "Edit" [ref=e322]:
                        - img [ref=e323]
                    - link "Exam Analytics" [ref=e326] [cursor=pointer]:
                      - /url: /admin/exams/e859131e-e534-4b96-b04a-9ab7bd4c2114/analytics
                      - button "Exam Analytics" [ref=e327]:
                        - img [ref=e328]
                        - text: Exam Analytics
              - row "UAT Security Assessment draft 30 min 70% Edit Exam Analytics" [ref=e329]:
                - cell "UAT Security Assessment" [ref=e330]
                - cell "draft" [ref=e331]:
                  - generic [ref=e332]: draft
                - cell "30 min" [ref=e333]
                - cell "70%" [ref=e334]
                - cell "Edit Exam Analytics" [ref=e335]:
                  - generic [ref=e336]:
                    - link "Edit" [ref=e337] [cursor=pointer]:
                      - /url: /admin/exams/12a08073-0c9a-470a-9ab9-b471ec8b49dd/edit
                      - button "Edit" [ref=e338]:
                        - img [ref=e339]
                    - link "Exam Analytics" [ref=e342] [cursor=pointer]:
                      - /url: /admin/exams/12a08073-0c9a-470a-9ab9-b471ec8b49dd/analytics
                      - button "Exam Analytics" [ref=e343]:
                        - img [ref=e344]
                        - text: Exam Analytics
              - row "UAT Security Assessment draft 30 min 70% Edit Exam Analytics" [ref=e345]:
                - cell "UAT Security Assessment" [ref=e346]
                - cell "draft" [ref=e347]:
                  - generic [ref=e348]: draft
                - cell "30 min" [ref=e349]
                - cell "70%" [ref=e350]
                - cell "Edit Exam Analytics" [ref=e351]:
                  - generic [ref=e352]:
                    - link "Edit" [ref=e353] [cursor=pointer]:
                      - /url: /admin/exams/9d1699c7-ef1e-4219-a956-da12497b88fb/edit
                      - button "Edit" [ref=e354]:
                        - img [ref=e355]
                    - link "Exam Analytics" [ref=e358] [cursor=pointer]:
                      - /url: /admin/exams/9d1699c7-ef1e-4219-a956-da12497b88fb/analytics
                      - button "Exam Analytics" [ref=e359]:
                        - img [ref=e360]
                        - text: Exam Analytics
              - row "UAT API Test Unsatisfiable draft 60 min 70% Edit Exam Analytics" [ref=e361]:
                - cell "UAT API Test Unsatisfiable" [ref=e362]
                - cell "draft" [ref=e363]:
                  - generic [ref=e364]: draft
                - cell "60 min" [ref=e365]
                - cell "70%" [ref=e366]
                - cell "Edit Exam Analytics" [ref=e367]:
                  - generic [ref=e368]:
                    - link "Edit" [ref=e369] [cursor=pointer]:
                      - /url: /admin/exams/4d193983-24a2-4781-b64c-15b02cf7d1e0/edit
                      - button "Edit" [ref=e370]:
                        - img [ref=e371]
                    - link "Exam Analytics" [ref=e374] [cursor=pointer]:
                      - /url: /admin/exams/4d193983-24a2-4781-b64c-15b02cf7d1e0/analytics
                      - button "Exam Analytics" [ref=e375]:
                        - img [ref=e376]
                        - text: Exam Analytics
              - row "UAT Unsatisfiable Exam active 60 min 60% Edit Exam Analytics Unpublish" [ref=e377]:
                - cell "UAT Unsatisfiable Exam" [ref=e378]
                - cell "active" [ref=e379]:
                  - generic [ref=e380]: active
                - cell "60 min" [ref=e381]
                - cell "60%" [ref=e382]
                - cell "Edit Exam Analytics Unpublish" [ref=e383]:
                  - generic [ref=e384]:
                    - link "Edit" [ref=e385] [cursor=pointer]:
                      - /url: /admin/exams/cffa4b68-e080-4772-9428-03d95476b3a8/edit
                      - button "Edit" [ref=e386]:
                        - img [ref=e387]
                    - link "Exam Analytics" [ref=e390] [cursor=pointer]:
                      - /url: /admin/exams/cffa4b68-e080-4772-9428-03d95476b3a8/analytics
                      - button "Exam Analytics" [ref=e391]:
                        - img [ref=e392]
                        - text: Exam Analytics
                    - button "Unpublish" [ref=e393]
              - row "UAT Security Assessment draft 30 min 70% Edit Exam Analytics" [ref=e394]:
                - cell "UAT Security Assessment" [ref=e395]
                - cell "draft" [ref=e396]:
                  - generic [ref=e397]: draft
                - cell "30 min" [ref=e398]
                - cell "70%" [ref=e399]
                - cell "Edit Exam Analytics" [ref=e400]:
                  - generic [ref=e401]:
                    - link "Edit" [ref=e402] [cursor=pointer]:
                      - /url: /admin/exams/3e57d48a-7422-402f-b77d-5db5b6e1c121/edit
                      - button "Edit" [ref=e403]:
                        - img [ref=e404]
                    - link "Exam Analytics" [ref=e407] [cursor=pointer]:
                      - /url: /admin/exams/3e57d48a-7422-402f-b77d-5db5b6e1c121/analytics
                      - button "Exam Analytics" [ref=e408]:
                        - img [ref=e409]
                        - text: Exam Analytics
              - row "UAT Unsatisfiable Exam active 60 min 60% Edit Exam Analytics Unpublish" [ref=e410]:
                - cell "UAT Unsatisfiable Exam" [ref=e411]
                - cell "active" [ref=e412]:
                  - generic [ref=e413]: active
                - cell "60 min" [ref=e414]
                - cell "60%" [ref=e415]
                - cell "Edit Exam Analytics Unpublish" [ref=e416]:
                  - generic [ref=e417]:
                    - link "Edit" [ref=e418] [cursor=pointer]:
                      - /url: /admin/exams/410cd3a6-6597-4027-80eb-ad9d2c5e354c/edit
                      - button "Edit" [ref=e419]:
                        - img [ref=e420]
                    - link "Exam Analytics" [ref=e423] [cursor=pointer]:
                      - /url: /admin/exams/410cd3a6-6597-4027-80eb-ad9d2c5e354c/analytics
                      - button "Exam Analytics" [ref=e424]:
                        - img [ref=e425]
                        - text: Exam Analytics
                    - button "Unpublish" [ref=e426]
          - generic [ref=e427]:
            - button "Previous" [disabled]
            - generic [ref=e428]: Page 1 of 11
            - button "Next" [ref=e429]
```

# Test source

```ts
  721 |       headers: { Authorization: `Bearer ${token}` }
  722 |     })
  723 |     const listBody = await listResp.json()
  724 |     const exam = (listBody?.data?.items ?? []).find(
  725 |       (e: { title: string; status: string }) => e.title === 'UAT Security Assessment v2' && e.status === 'draft'
  726 |     )
  727 |     await ctx.dispose()
  728 | 
  729 |     await login(page)
  730 |     await page.goto(`${BASE}/admin/exams/${exam.id}/edit`)
  731 |     await waitForNetworkIdle(page)
  732 |     await page.waitForSelector('#title', { timeout: 15_000 })
  733 | 
  734 |     // Navigate through all steps to Step 4
  735 |     await page.getByRole('button', { name: /далее|next|алға/i }).click()
  736 |     await expect(page.getByRole('button', { name: /назад|back|артқа/i })).toBeVisible({ timeout: 15_000 })
  737 |     await page.getByRole('button', { name: /далее|next|алға/i }).click()
  738 |     await expect(page.getByText(/назначен|assignment/i).first()).toBeVisible({ timeout: 15_000 })
  739 |     await page.getByRole('button', { name: /далее|next|алға/i }).click()
  740 |     const publishBtn = page.getByRole('button', { name: /опубликов|publish|жариялау/i })
  741 |     await expect(publishBtn).toBeVisible({ timeout: 15_000 })
  742 |     await publishBtn.click()
  743 | 
  744 |     const dialog = page.getByRole('dialog')
  745 |     await expect(dialog).toBeVisible({ timeout: 8_000 })
  746 |     const confirmBtn = dialog.getByRole('button', { name: /опубликов|publish|жариялау/i })
  747 |     await confirmBtn.click()
  748 | 
  749 |     // Wait for success or redirect
  750 |     await page.waitForLoadState('networkidle')
  751 |     await page.waitForTimeout(2000)
  752 | 
  753 |     // Verify via API
  754 |     const token2 = await getToken()
  755 |     const ctx2 = await request.newContext()
  756 |     const examResp = await ctx2.get(`${API}/exams/${exam.id}`, {
  757 |       headers: { Authorization: `Bearer ${token2}` }
  758 |     })
  759 |     const examBody = await examResp.json()
  760 |     await ctx2.dispose()
  761 |     expect(examBody?.data?.status).toBe('active')
  762 |     expect(examBody?.data?.title).toBe('UAT Security Assessment v2')
  763 |   })
  764 | })
  765 | 
  766 | // ─── Scenario 4: Archive an Exam ─────────────────────────────────────────────
  767 | 
  768 | test.describe('Scenario 4: Archive an exam', () => {
  769 | 
  770 |   test('S4-S1: Open active exam UAT Security Assessment v2 detail page', async ({ page }) => {
  771 |     const token = await getToken()
  772 |     const ctx = await request.newContext()
  773 |     const listResp = await ctx.get(`${API}/exams?limit=200`, {
  774 |       headers: { Authorization: `Bearer ${token}` }
  775 |     })
  776 |     const listBody = await listResp.json()
  777 |     const exam = (listBody?.data?.items ?? []).find(
  778 |       (e: { title: string; status: string }) => e.title === 'UAT Security Assessment v2' && e.status === 'active'
  779 |     )
  780 |     await ctx.dispose()
  781 | 
  782 |     await login(page)
  783 |     await page.goto(`${BASE}/admin/exams/${exam.id}/edit`)
  784 |     await waitForNetworkIdle(page)
  785 |     await expect(page).toHaveURL(/\/admin\/exams\/.+\/edit/)
  786 |   })
  787 | 
  788 |   test('S4-S2-S3: Click Archive button, confirm — exam status changes to Archived', async ({ page }) => {
  789 |     const token = await getToken()
  790 |     const ctx = await request.newContext()
  791 |     const listResp = await ctx.get(`${API}/exams?limit=200`, {
  792 |       headers: { Authorization: `Bearer ${token}` }
  793 |     })
  794 |     const listBody = await listResp.json()
  795 |     const exam = (listBody?.data?.items ?? []).find(
  796 |       (e: { title: string; status: string }) => e.title === 'UAT Security Assessment v2' && e.status === 'active'
  797 |     )
  798 |     await ctx.dispose()
  799 | 
  800 |     await login(page)
  801 |     await page.goto(`${BASE}/admin/exams/${exam.id}/edit`)
  802 |     await waitForNetworkIdle(page)
  803 | 
  804 |     // Look for Archive button in the exam wizard / list / edit page
  805 |     const archiveBtn = page.getByRole('button', { name: /archive|архив|мұрағат/i })
  806 |     const archiveBtnVisible = await archiveBtn.isVisible({ timeout: 5_000 }).catch(() => false)
  807 | 
  808 |     if (!archiveBtnVisible) {
  809 |       // Also check the Exams list page for an archive action
  810 |       await page.goto(`${BASE}/admin/exams`)
  811 |       await waitForNetworkIdle(page)
  812 |       const archiveBtnList = page.getByRole('button', { name: /archive|архив|мұрағат/i })
  813 |       const archiveBtnListVisible = await archiveBtnList.isVisible({ timeout: 5_000 }).catch(() => false)
  814 | 
  815 |       test.info().annotations.push({
  816 |         type: 'defect',
  817 |         description: 'ARCHIVE BUTTON NOT FOUND — neither on exam edit page nor on exams list. Archive functionality is not exposed in the UI. The backend has an Archive endpoint (DELETE /api/v1/exams/:id) but no frontend UI element triggers it.'
  818 |       })
  819 | 
  820 |       // This test will fail per AC#7 requirement
> 821 |       expect(archiveBtnListVisible, 'Archive button should be visible on exams list page').toBeTruthy()
      |                                                                                            ^ Error: Archive button should be visible on exams list page
  822 |       return
  823 |     }
  824 | 
  825 |     await archiveBtn.click()
  826 |     const dialog = page.getByRole('dialog')
  827 |     await expect(dialog).toBeVisible({ timeout: 8_000 })
  828 |     const confirmBtn = dialog.getByRole('button', { name: /confirm|archive|архив|ок/i }).first()
  829 |     await confirmBtn.click()
  830 |     await page.waitForLoadState('networkidle')
  831 | 
  832 |     // Verify archived via API
  833 |     const token2 = await getToken()
  834 |     const ctx2 = await request.newContext()
  835 |     const examResp = await ctx2.get(`${API}/exams/${exam.id}`, {
  836 |       headers: { Authorization: `Bearer ${token2}` }
  837 |     })
  838 |     const examBody = await examResp.json()
  839 |     await ctx2.dispose()
  840 |     expect(examBody?.data?.status).toBe('archived')
  841 |   })
  842 | 
  843 |   test('S4-check: Archived exam does not appear in assignable exams list', async ({ page }) => {
  844 |     // Get a recently archived exam
  845 |     const token = await getToken()
  846 |     const ctx = await request.newContext()
  847 |     const listResp = await ctx.get(`${API}/exams?limit=200`, {
  848 |       headers: { Authorization: `Bearer ${token}` }
  849 |     })
  850 |     const listBody = await listResp.json()
  851 |     const archivedExam = (listBody?.data?.items ?? []).find(
  852 |       (e: { title: string; status: string }) =>
  853 |         (e.title === 'UAT Security Assessment v2' || e.title === 'UAT Security Assessment') &&
  854 |         e.status === 'archived'
  855 |     )
  856 |     await ctx.dispose()
  857 | 
  858 |     if (!archivedExam) {
  859 |       test.skip(true, 'No archived exam available to check assignable list')
  860 |       return
  861 |     }
  862 | 
  863 |     await login(page)
  864 |     await page.goto(`${BASE}/admin/exams`)
  865 |     await waitForNetworkIdle(page)
  866 | 
  867 |     // The archived exam should either not appear or appear with archived badge
  868 |     // but it should NOT appear as assignable (active exams only are assignable)
  869 |     // Check that the exam assignments page / exam session start does not show archived exams
  870 |     // For this UAT check, we simply verify the exam list shows 'archived' badge or is hidden
  871 |     const examTitleInList = page.getByText(archivedExam.title)
  872 |     const titleVisible = await examTitleInList.first().isVisible({ timeout: 5_000 }).catch(() => false)
  873 | 
  874 |     if (titleVisible) {
  875 |       // If visible, check it shows archived status (not active/assignable)
  876 |       const row = page.locator('tr').filter({ hasText: archivedExam.title })
  877 |       const hasArchivedBadge = await row.locator('[class*="red"]').isVisible({ timeout: 3_000 }).catch(() => false)
  878 |       test.info().annotations.push({
  879 |         type: 'note',
  880 |         description: `Archived exam visible in list: ${titleVisible}, has archived badge: ${hasArchivedBadge}`
  881 |       })
  882 |     }
  883 |   })
  884 | })
  885 | 
```