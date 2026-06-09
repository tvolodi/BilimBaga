---
id: ISS-035
title: "UAT Defect: Exam Assignment — No UI path to manage assignments for active exams"
status: resolved
severity: high
layer: frontend
module: exams
tags: [uat, exam-assignment, regression, iss-034-side-effect]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: [ISS-034]
regression_test: frontend/src/pages/ExamWizard/Step1BasicSettings.test.tsx, frontend/src/pages/admin/ExamsListPage.test.tsx
---

## Symptom
UAT Scenario: `Exam Assignment — Assign Exam to Individual Employee`, Step 3
Actor: Super Admin
Action: Click "Assign" / navigate to the Assignments tab (Step 3) for an active exam "UAT Security Assessment"
Expected: Assignment form/modal appears — admin can select employees and confirm the assignment
Actual: There is no "Assign" button on the ExamsListPage for active exams. The wizard Step 1 Next button is `disabled={isReadOnly}` (where `isReadOnly = exam?.status === 'active'`), so Step 3 (Assignments) is unreachable via the wizard. The StepIndicator uses non-clickable `<div>` elements, so there is no way to jump directly to Step 3. Admin cannot assign or manage assignments for an active exam via the UI at all.
Screenshot: test-results/uat-exam-assignment-S1/test-failed-1.png

This same blocking defect causes failures in all four UAT scenarios:
- S1 Steps 3–5 (assign individual, no deadline)
- S2 Steps 1–2 (assign with deadline)
- S3 Steps 1–3 (remove assignment)
- S4 Step 2 (view completion status table)

Total: 8 of 18 steps failed due to this single root cause.

## Root Cause
(to be filled by Issue Resolution)

The ISS-034 fix added `const isReadOnly = exam?.status === 'active'` in `Step1BasicSettings.tsx` and applied `disabled={isReadOnly}` to the Step 1 submit/Next button. This correctly prevents editing exam *settings* for active exams, but it also prevents navigation to the Assignments step (Step 3) which does not involve editing exam settings at all. Assignment management is a distinct operation that must remain available while an exam is active — that is precisely when assignments are needed.

Additionally, the ExamsListPage action menu for active exams offers: Edit | Analytics | Unpublish | Archive — but no "Assign" shortcut, leaving no alternative UI entry point.

## Fix Applied

Two complementary changes:

**1. Step1BasicSettings.tsx — allow navigation without saving for active exams**

Changed `handleSubmit` to short-circuit when `isReadOnly=true`: calls `onDone(exam.id, resolvedSectionId)` directly, skipping validation and all API mutations. This preserves the ISS-034 read-only field state (all inputs remain `disabled={isReadOnly}`) and banner, while unblocking the Next button for navigation purposes.

The Next button's `disabled` prop was changed from `disabled={isPending || isReadOnly}` to `disabled={isPending}`, because for active exams the button's only job is navigation (no save), so there is no reason to disable it.

**2. ExamsListPage.tsx — direct "Assign" shortcut for active exams**

Added an "Assign" button (with `Users` icon) in the actions column for active exams. The link routes to `/admin/exams/{id}/edit?step=3`.

**3. ExamWizard/index.tsx — support `?step=N` URL query parameter**

Added `useSearchParams` to read an optional `step` query param. When the wizard is opened in edit mode with `?step=3`, it initialises `useState(initialStep)` at step 3, allowing the "Assign" button to land the admin directly on the Assignments tab without clicking through Step 1 and Step 2.

**4. Locale files — new `exam_list.manage_assignments` key**

Added the new i18n key to all three locale files (en/kk/ru).

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/pages/ExamWizard/Step1BasicSettings.tsx` | Short-circuit `handleSubmit` for active exams; remove `isReadOnly` from button `disabled` prop |
| `frontend/src/pages/ExamWizard/index.tsx` | Read `?step` search param; initialise wizard step from URL |
| `frontend/src/pages/admin/ExamsListPage.tsx` | Add "Assign" button for active exams linking to `?step=3` |
| `frontend/src/locales/en.json` | Add `exam_list.manage_assignments` key |
| `frontend/src/locales/kk.json` | Add `exam_list.manage_assignments` key |
| `frontend/src/locales/ru.json` | Add `exam_list.manage_assignments` key |
| `frontend/src/pages/ExamWizard/Step1BasicSettings.test.tsx` | Update test: Next button enabled for active exams; add navigation-without-save test |
| `frontend/src/pages/admin/ExamsListPage.test.tsx` | Add tests: Assign button present for active; absent for draft; href includes `?step=3` |

## Regression Test

- `frontend/src/pages/ExamWizard/Step1BasicSettings.test.tsx`
  - `keeps the Next button enabled for active exams to allow navigation` — ensures button is never disabled for active exams
  - `calls onDone directly without saving when Next is clicked for an active exam` — ensures no API call is made and navigation fires immediately
- `frontend/src/pages/admin/ExamsListPage.test.tsx`
  - `shows Assign button for active exams`
  - `does not show Assign button for draft exams`
  - `Assign button links to wizard edit page at step 3 for active exams`

## Resolution Results
- Tests: 248 passed, 0 failed (frontend `npm test`)
- Backend: all packages pass (`go test ./...`)
- Migration applied: no (frontend-only fix)
- Build clean: yes
