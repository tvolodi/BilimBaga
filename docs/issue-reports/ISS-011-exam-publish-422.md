---
id: ISS-011
title: Publish exam returns 422 — "0 available" because questions are in draft status
status: resolved
severity: medium
layer: frontend
module: exams
tags: [422, publish, CountAvailableForRule, question-status, draft, active, validationWarning]
created: 2026-05-19
resolved: 2026-05-19
recurrence_count: 1
related_issues: [ISS-010]
regression_test: backend/internal/exams/service_test.go
---

## Symptom
User tries to publish an exam on the "Review & Publish" (Step 4) page. The frontend shows:

> "Some question rules cannot be satisfied. Fix them before publishing."
> "Rule 06f834c1…: needs 1, only 0 available"

Browser console:
1. `POST /api/v1/auth/refresh` → 401 (background token rotation — unrelated)
2. `POST /api/v1/exams/{id}/publish` → 422 Unprocessable Entity

Exam rule: random, 1 question, medium difficulty.

## Root Cause
`CountAvailableForRule` in `backend/internal/exams/repository.go` builds a query that filters
`q.status = 'active'`. All questions are created with `status = 'draft'` by default (via
`CreateQuestionFull` / `CreateFull` in the questions package). Users who have not explicitly
transitioned their questions through `draft → review → active` will always have 0 active questions,
causing every random rule to report `available = 0`.

The backend behavior is **correct and intentional**: only `active` questions are usable in exam
sessions (the sessions repository also uses `q.status = 'active'` when selecting questions at
session-start time). Publishing an exam with 0 active questions would allow users to publish but
then fail at session-start, which is a worse UX.

The 401 on `/auth/refresh` (Root Cause C) is a separate background token-rotation event and does
NOT affect the publish call — the access token is still valid at the time of the publish request.

The actual problem is **UX**: the error message "needs 1, only 0 available" gives the user no
indication of WHY the count is 0 or what they must do to fix it (navigate to the Question Bank and
activate questions).

## Fix Applied
- `frontend/src/pages/ExamWizard/Step4Review.tsx`: added a descriptive hint paragraph below the
  unsatisfied-rules list that is shown whenever at least one rule reports `available === 0`. The
  hint explains that only questions with `active` status are counted, and tells the user to open
  the Question Bank to activate their questions.
- `frontend/src/locales/{en,ru,kk}.json`: added `exam.wizard.validationWarningHint` i18n key with
  the hint text in all three supported locales.
- `backend/internal/exams/service_test.go`: added
  `TestPublish_ZeroAvailableActiveQuestions_ReturnsValidationError` to explicitly cover the
  ISS-011 scenario (random rule requiring 1 medium question, CountAvailableForRule returns 0,
  PublishValidationError is returned with correct rule detail including the `difficulty` filter).

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/pages/ExamWizard/Step4Review.tsx` | Added hint paragraph for 0-available rules |
| `frontend/src/locales/en.json` | Added `exam.wizard.validationWarningHint` |
| `frontend/src/locales/ru.json` | Added `exam.wizard.validationWarningHint` |
| `frontend/src/locales/kk.json` | Added `exam.wizard.validationWarningHint` |
| `backend/internal/exams/service_test.go` | Added ISS-011 regression test |

## Regression Test
`backend/internal/exams/service_test.go` →
`TestPublish_ZeroAvailableActiveQuestions_ReturnsValidationError`

Verifies: when a random rule requires 1 question of difficulty="medium" and CountAvailableForRule
returns 0, Publish returns a PublishValidationError with Details[0].Available == 0 and
Details[0].Filter containing the difficulty filter.

## Resolution Results
- Tests: all backend tests pass
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
| 2026-05-19 | Bug reported: 422 on exam publish with draft questions | UX fix + regression test |
