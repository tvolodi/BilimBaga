---
id: ISS-033
title: "UAT Defect: Exam Configuration — Manual-mode exam rules bypass publish validation"
status: resolved
severity: medium
layer: backend
module: exams
tags: [uat, exam-configuration]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: []
regression_test: null
---

## Symptom
UAT Scenario: `Exam Configuration`, Step AC#4
Actor: Admin
Action: Creates an exam with a manual-mode rule specifying 50 questions but selects 0 questions, then publishes
Expected: Publish is rejected with a validation error indicating the manual rule has insufficient questions selected
Actual: The backend skips validation for manual-mode rules entirely. In `backend/internal/exams/service.go:196`, the check `if rule.Mode == "manual" { continue }` causes the publish endpoint to accept a manual rule with 50 count and 0 selected questions without returning an error.
Screenshot: none

## Root Cause
In `backend/internal/exams/service.go` the `Publish()` method iterated over all question rules and used `continue` to skip any rule with `mode == "manual"`. This meant manual-mode rules were never validated before publish: an exam with a manual rule requiring 50 questions but 0 selected could be published without error. The repository already had `CountAvailableForManualRule` (used by `GetEligibleCounts`) but the publish validation path never called it.

## Fix Applied
Replaced the `if rule.Mode == "manual" { continue }` guard with an active validation branch: for manual-mode rules, `CountAvailableForManualRule(ctx, rule.ID)` is called and, if the count of linked active questions is less than `rule.Count`, a `RuleUnsatisfiedDetail` is appended to `unsatisfied` (same `PublishValidationError` path as random rules). The fix mirrors the pattern already used in `GetEligibleCounts`.

No new repository method was needed — `CountAvailableForManualRule` already existed.

## Files Changed
- `backend/internal/exams/service.go` — publish validation loop now covers manual rules
- `backend/internal/exams/service_test.go` — replaced stale `TestPublish_ManualRulesSkippedInValidation` (tested the bug) with four regression tests covering ISS-033

## Regression Test
Four new service-level tests added to `service_test.go`:
- `TestPublish_ManualRule_SufficientQuestions_Succeeds` — happy path: 3 of 3 selected → publishes OK
- `TestPublish_ManualRule_ZeroSelectedQuestions_ReturnsUnsatisfied` — 0 of 3 selected → `EXAM_RULES_UNSATISFIED`
- `TestPublish_ManualRule_InsufficientSelectedQuestions_ReturnsUnsatisfied` — 2 of 50 selected → `EXAM_RULES_UNSATISFIED`
- `TestPublish_MixedRules_BothUnsatisfied_ReportsBoth` — one random + one manual, both unsatisfied → error lists both

## Resolution Results
All 4 new tests PASS. Full `go test ./...` PASS (25 packages). Build clean.
