---
id: ISS-228
title: Legacy blank-option questions pass ->review; AI Generate e2e locator strict-mode violation
status: resolved
severity: low
layer: backend
module: questions
tags: [TransitionStatus, validateOptionTexts, ai-generate, strict-mode]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-173]
regression_test: backend/internal/questions/option_validation_test.go
---

## Symptom
(2) draft->review returned 200 for choice questions with blank default-locale option text; only ->active rejected them.
(3) e2e `getByRole('button', { name: /generate|ai/i })` hits a strict-mode violation when a question title contains "ai".
(1) Blank option text in non-default locales: needs BA decision, left untouched.

## Root Cause
(2) The option-text check in `TransitionStatus` was gated on `newStatus == "active"`. (3) Over-broad regex locator.

## Fix Applied
(2) Gate is now `review || active`, same validator, same 422 ERR_VALIDATION `{data:null,error}` format. (3) Added `data-testid="ai-generate-button"` to the page-level button; e2e specs use `getByTestId`.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/questions/service.go | validate options on ->review |
| backend/internal/questions/option_validation_test.go | review service + handler tests; draft->review removed from "ignored" list |
| frontend/src/pages/admin/questions/QuestionBankPage.tsx | data-testid |
| frontend/e2e/{question-management,question-bank,ai-assist}.spec.ts | testid locator |

## Regression Test
option_validation_test.go: TestTransitionStatus_ReviewBlankOptionText, TestQHandlerTransitionStatus_ReviewBlankOptions_Returns422.

## Resolution Results
- Tests: questions package passes; tsc/eslint clean; e2e not run (stack belongs to UAT)
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
