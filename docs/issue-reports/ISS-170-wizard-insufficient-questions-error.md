---
id: ISS-170
title: Exam wizard shows nothing for INSUFFICIENT_ADAPTIVE_QUESTIONS (details is an object)
status: resolved
severity: medium
layer: frontend
module: exams
tags: [INSUFFICIENT_ADAPTIVE_QUESTIONS, INSUFFICIENT_QUESTIONS, ExamApiError, unsatisfiedRules, Step4Review]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: frontend/src/pages/ExamWizard/Step4Review.errors.test.tsx
---

## Symptom
Publishing an adaptive exam with too few questions per difficulty displayed nothing in the wizard (BA review #168, GH #170).

## Root Cause
Backend returns `details` as an object `{rule_id, difficulty, required, available}` for INSUFFICIENT_ADAPTIVE_QUESTIONS, but `ExamApiError` stored `error.details` blindly as `unsatisfiedRules` (typed as array). Step4Review took the 422 + truthy `unsatisfiedRules` branch, where `.length > 0` on an object is false, so no banner rendered and the general-error branch was skipped. INSUFFICIENT_QUESTIONS has no details (handled by PR #142 only in Step 4; other steps showed raw message).

## Fix Applied
- `ExamApiError` keeps raw `details: unknown`; `unsatisfiedRules` only set when details is an array.
- New `src/api/examErrors.ts`: `parseInsufficientDetails`, `safeMessage`, `describeExamError` (localized, interpolated counts/difficulty/rule, safe fallbacks, never `[object Object]`).
- Step4Review and Step2QuestionRules use `describeExamError`; error blocks have `role="alert"`; 422 array branch guarded by `Array.isArray`.
- New i18n keys in en/ru/kk: `exam.wizard.step4.adaptiveInsufficient{,Counts,Detailed}`, `genericError`.

## Files Changed
| File | Change |
|------|--------|
| frontend/src/api/exams.ts | ExamApiError carries `details` |
| frontend/src/api/examErrors.ts | new helper |
| frontend/src/pages/ExamWizard/Step4Review.tsx, Step2QuestionRules.tsx | use helper, role=alert |
| frontend/src/locales/{en,ru,kk}.json | new keys |

## Regression Test
`src/api/examErrors.test.ts` (helper), `src/pages/ExamWizard/Step4Review.errors.test.tsx` (both codes x object/string/missing details, array details).

## Resolution Results
- Tests: 517 passed, 0 failed (73 files); tsc clean; check:i18n OK
- Migration applied: no
- Build clean: yes (tsc)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
