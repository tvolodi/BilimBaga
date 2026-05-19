---
id: ISS-010
title: Exam wizard step 3 — sections URL empty ID, assignment 422, i18n type label
status: resolved
severity: high
layer: frontend, backend
module: exams
tags: [exam-wizard, assignments, sections, i18n, draft-status, ErrNotActive]
created: 2026-05-19
resolved: 2026-05-19
recurrence_count: 1
related_issues: []
regression_test: backend/internal/exams/service_test.go
---

## Symptom
Four errors observed in step 3 ("Назначения") of the exam creation wizard (`/admin/exams/new`):

1. `POST /api/v1/exams//sections` → 405 — double slash, empty exam ID
2. `POST /api/v1/exams/{id}/assign` → 422 "Exams must be in active status before they can be assigned."
3. `i18n: key 'exam.assignment.type (ru)' returned an object instead of string` — console warning on the assignment type `<Label>`
4. Red banner: "Exams must be in active status before they can be assigned." (same root as #2)

## Root Cause

### Issue #1 — Empty examId in `useAddSection`
`Step1BasicSettings.tsx` called `useAddSection(exam?.id ?? '')`. On new-exam creation `exam` is `null`, so the hook closed over an empty string. The `addSection.mutateAsync` call (which fires *after* `createExam` returns) still used the empty string, producing `/api/v1/exams//sections`.

### Issues #2 & #4 — Backend blocks draft-exam assignments
`service.go CreateAssignment` had `if exam.Status != "active" { return nil, ErrNotActive }`. New exams are created in `draft` status; the wizard's step 3 happens before step 4 (publish). Wizard flow intends assignments to be configured pre-publish, so the restriction was too strict.

### Issue #3 — `exam.assignment.type` i18n object used as string
All three locale files have `exam.assignment.type` as a nested object (`{ user, department, all }`). `Step3Assignments.tsx` used `t('exam.assignment.type')` as a plain string for a `<Label>`, triggering the react-i18next "returned an object" warning. No separate label key existed.

## Fix Applied

**`frontend/src/api/exams.ts`** — Refactored `useAddSection` to take `examId` as part of the mutation variables instead of a hook parameter, so the correct ID is available at call time regardless of when the hook was initialized.

**`frontend/src/pages/ExamWizard/Step1BasicSettings.tsx`** — Updated `useAddSection()` call (no arg) and `addSection.mutateAsync({ examId: created.id, sort_order: 0 })` to pass the freshly-created exam ID.

**`backend/internal/exams/service.go`** — Changed status check from `!= "active"` to `!= "active" && != "draft"` (i.e., only reject `archived` exams).

**`frontend/src/pages/ExamWizard/Step3Assignments.tsx`** — Changed `t('exam.assignment.type')` → `t('exam.assignment.typeLabel')` for the label element.

**`frontend/src/locales/en.json`, `ru.json`, `kk.json`** — Added `exam.assignment.typeLabel` string key ("Assignment type" / "Тип назначения" / "Тапсырма түрі").

**`backend/internal/exams/service_test.go`** — Updated `TestCreateAssignment_ExamNotActive_Returns422` to seed an `archived` exam (previously `draft`, which now succeeds). Added `TestCreateAssignment_DraftExam_Succeeds` regression test.

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/api/exams.ts` | `useAddSection` takes `examId` in mutation vars |
| `frontend/src/pages/ExamWizard/Step1BasicSettings.tsx` | Updated hook call + mutateAsync arg |
| `frontend/src/pages/ExamWizard/Step3Assignments.tsx` | `typeLabel` key for Label |
| `frontend/src/locales/en.json` | Added `exam.assignment.typeLabel` |
| `frontend/src/locales/ru.json` | Added `exam.assignment.typeLabel` |
| `frontend/src/locales/kk.json` | Added `exam.assignment.typeLabel` |
| `backend/internal/exams/service.go` | Allow `draft` + `active` for assignment creation |
| `backend/internal/exams/service_test.go` | Fixed existing test; added regression test |

## Regression Test
`backend/internal/exams/service_test.go` — `TestCreateAssignment_DraftExam_Succeeds` verifies that a draft-status exam can receive assignments.

## Resolution Results
- Backend tests: all pass (no FAIL lines)
- Frontend TypeScript: clean (no errors)
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-19 | Initial report from browser screenshot | Fixed all four issues |
