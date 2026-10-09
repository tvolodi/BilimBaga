---
id: ISS-151
title: Adaptive exam with zero rules can be published and then cannot start
status: resolved
severity: medium
layer: backend
module: exams
tags: [ErrNoQuestionRules, Publish, adaptive, INSUFFICIENT_QUESTIONS]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-132]
regression_test: backend/internal/exams/service_test.go
---

## Symptom
Follow-up to ISS-132 (PR #142): non-adaptive exams with no rules were refused on publish, but an adaptive exam with zero rules published (the adaptive per-rule loop is vacuous) and then could not start.

## Root Cause
`service.Publish` guarded the empty check with `!e.Adaptive && len(rules) == 0`, so adaptive exams skipped it.

## Fix Applied
Dropped the `!e.Adaptive` condition: any exam with zero rules returns `ErrNoQuestionRules` (handler: 422 `INSUFFICIENT_QUESTIONS`, same as non-adaptive). Rules that cannot produce questions were already refused by the adaptive >=5-per-difficulty check (FR-BB72) and the unsatisfied-rules check. No frontend change: the code/message path is shared with the non-adaptive case.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/exams/service.go | empty-rules check applies to adaptive too |
| backend/internal/exams/service_test.go | adaptive zero rules refused; adaptive valid rule publishes |
| backend/internal/exams/handler_test.go | wrapped ErrNoQuestionRules -> 422 INSUFFICIENT_QUESTIONS |

## Regression Test
`TestPublish_Adaptive_NoRules_ReturnsErrNoQuestionRules`, `TestPublish_Adaptive_WithValidRule_Succeeds`, `TestPublish_AdaptiveNoRules_Returns422InsufficientQuestions`.

## Resolution Results
- Tests: full backend suite passed (go test -p 2 ./...)
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
