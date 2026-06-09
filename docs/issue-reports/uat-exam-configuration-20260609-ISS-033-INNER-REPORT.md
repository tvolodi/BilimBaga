# uat-exam-configuration-20260609 / ISS-033: Implementation Inner Report

**Date**: 2026-06-09
**Pipeline**: B
**Commit**: 420a9ea

## Summary
The `Publish()` service method in the exams domain silently skipped validation for manual-mode question rules (`if rule.Mode == "manual" { continue }`). This allowed an exam with a manual rule requiring N questions but zero selected questions to be published without error, violating the business rule that every rule must be satisfiable at publish time. The fix replaces the bypass with an active validation call to the existing `CountAvailableForManualRule` repository method; if the count of linked active questions is below `rule.Count`, the same `PublishValidationError` / `EXAM_RULES_UNSATISFIED` path is taken as for random rules.

## Files Changed
| File | Action |
|------|--------|
| `backend/internal/exams/service.go` | modified — publish validation loop now covers manual rules |
| `backend/internal/exams/service_test.go` | modified — replaced stale bypass test with 4 ISS-033 regression tests |
| `docs/issue-reports/ISS-033-uat-defect.md` | created — issue record with root cause, fix, and resolution results |

## Acceptance Criteria Verified
| AC | Verified By |
|----|-------------|
| AC#4: publish with unsatisfied manual rule rejected with error naming the failing rule; exam remains draft | test: TestPublish_ManualRule_ZeroSelectedQuestions_ReturnsUnsatisfied, TestPublish_ManualRule_InsufficientSelectedQuestions_ReturnsUnsatisfied |
| Happy path: sufficient selected questions → publish succeeds | test: TestPublish_ManualRule_SufficientQuestions_Succeeds |
| Mixed rules: both unsatisfied rules reported | test: TestPublish_MixedRules_BothUnsatisfied_ReportsBoth |

## Test Results
- Backend: all packages pass (`go test ./...` — 25 packages, 0 failures)
- Frontend: not applicable (no frontend changes)

## Migration Applied
none

## Known Limitations
The adaptive publish validation (FR-BB72 AC-2) still skips manual-mode rules when checking per-difficulty question counts. This is intentional — manual rules have explicit question lists so per-difficulty counts are not meaningful. No change was needed there.
