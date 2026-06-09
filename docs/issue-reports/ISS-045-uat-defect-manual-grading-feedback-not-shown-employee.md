---
id: ISS-045
title: "UAT Defect: Manual Grading — Examiner manual_feedback not displayed on employee result screen"
status: resolved
severity: medium
layer: frontend
module: grading
tags: [uat, manual-grading, manual_feedback, QuestionBreakdownTable, session_question_scores]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: []
regression_test: null
---

## Symptom
UAT Scenario: Scenario 4, Step 4
Actor: Employee
Action: Employee opens the result/answer-review screen after a manually graded session where `show_answers = 'after_completion'` and the examiner has saved feedback text in `session_question_scores.manual_feedback`.
Expected: The examiner's feedback (e.g., "Good answer, covers the main points.") is visible on the per-question answer-review section.
Actual: Feedback text is not rendered. The frontend result/answer-review component does not display the `manual_feedback` field even though the backend stores and returns it.
Screenshot: none

## Root Cause
Three-layer omission:
1. `GetQuestionBreakdown` SQL query in `repository.go` did not SELECT `sqs.manual_feedback`.
2. `questionBreakdownRow` struct in `repository.go` had no `ManualFeedback` field.
3. `QuestionBreakdownItem` in `model.go` and the frontend `api/sessions.ts` type both lacked `manual_feedback`.
4. `QuestionBreakdownTable.tsx` had no column or cell for examiner feedback.

## Fix Applied
1. Added `sqs.manual_feedback` to the `GetQuestionBreakdown` SELECT query.
2. Added `ManualFeedback *string` to `questionBreakdownRow` (db struct) in `repository.go`.
3. Added `ManualFeedback *string` to `QuestionBreakdownItem` in `model.go`.
4. Propagated `ManualFeedback: b.ManualFeedback` in `buildSessionResult` in `service.go`.
5. Added `manual_feedback: string | null` to `QuestionBreakdownItem` in `frontend/src/api/sessions.ts`.
6. Added "Examiner Feedback" column to `QuestionBreakdownTable.tsx`, rendering feedback in blue when present.
7. Added `result.examiner_feedback` i18n key in en.json, ru.json, kk.json.

## Files Changed
| File | Change |
|------|--------|
| `backend/internal/sessions/repository.go` | Added `ManualFeedback *string` to `questionBreakdownRow`; added `sqs.manual_feedback` to SQL |
| `backend/internal/sessions/model.go` | Added `ManualFeedback *string` to `QuestionBreakdownItem` |
| `backend/internal/sessions/service.go` | Propagate `b.ManualFeedback` in breakdown construction in `buildSessionResult` |
| `frontend/src/api/sessions.ts` | Added `manual_feedback: string \| null` to `QuestionBreakdownItem` |
| `frontend/src/components/results/QuestionBreakdownTable.tsx` | Added "Examiner Feedback" column with feedback rendering |
| `frontend/src/locales/en.json` | Added `result.examiner_feedback` key |
| `frontend/src/locales/ru.json` | Added `result.examiner_feedback` key |
| `frontend/src/locales/kk.json` | Added `result.examiner_feedback` key |

## Regression Test
None added (covered by build + type-check + 254 passing unit tests).

## Resolution Results
- Tests: 254 passed, 0 failed
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
