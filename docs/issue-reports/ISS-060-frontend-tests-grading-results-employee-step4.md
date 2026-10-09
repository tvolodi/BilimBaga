---
id: ISS-060
title: No frontend tests for grading UI, result components, employee record, Step 4 eligible counts
status: resolved
severity: low
layer: frontend
module: exams
tags: [vitest, coverage, grading, results, employee-record, Step4Review, eligible-counts, FR-BB47, FR-BB45, FR-BB58, FR-BB315]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-051]
regression_test: frontend/src/pages/ExamWizard/Step4Review.test.tsx
---

## Symptom
GitHub #41 (from docs/handoffs/ba-drift-check/report.md): grading components/pages (FR-BB47), result components (FR-BB45), employee record components/page (FR-BB58) and the ExamWizard Step 4 eligible counts (FR-BB315) had zero test files, violating the project policy that every implementation ships with tests. Line coverage of these files was 0% (frontend overall 51.96%).

## Root Cause
Those features were implemented without tests; the missing-tests rule was not enforced at the time.

## Fix Applied
Added behaviour-level vitest + Testing Library tests (real i18n, API hooks mocked at module level) mapped to ACs. No product code changed; no product bug was exposed.

## Files Changed
| File | Change |
|------|--------|
| frontend/src/components/grading/GradingComponents.test.tsx | GradingNavigation, GradingQueueTable, SubmitAllGradesButton (FR-BB47 AC-2..4, 6..8) |
| frontend/src/pages/admin/GradingQueuePage.test.tsx | loading/error/empty, pagination, exam filter (FR-BB47) |
| frontend/src/pages/admin/GradingDetailPage.test.tsx | one-at-a-time nav, pre-fill, submit gating, sequential submit + navigate, failure stops (AC-4,5,7,8,9,11) |
| frontend/src/components/results/ResultComponents.test.tsx | PassFailBanner, ScoreDial (clamping), TimeTakenBadge, QuestionBreakdownTable (FR-BB45) |
| frontend/src/components/employees/EmployeeComponents.test.tsx | ExamStatusChip, EmployeeInfoHeader, TrackProgressCards, SessionHistoryTable, skeleton (FR-BB58 AC-2,3,4,7,8,12) |
| frontend/src/pages/admin/EmployeeRecordPage.test.tsx | skeleton, error+retry, pagination, export, values profile (FR-BB58 AC-6,11,12; FR-BB75) |
| frontend/src/pages/ExamWizard/Step4Review.test.tsx | eligible count badge colours, loading, error, rule_id matching (FR-BB315) |

## Regression Test
The seven files above plus an AC-6 block in frontend/src/test/QuestionGrader.test.tsx (74 new tests).

## Resolution Results
- Tests: 416 passed, 0 failed (66 files; baseline 342 / 59 files)
- Coverage (lines): 51.96% -> 59.28%
- Migration applied: no
- Build clean: yes (tsc, eslint, check:i18n)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |

## Follow-ups noticed
- FR-BB47 AC-9: `GradingDetailPage` sets a local-state success toast and immediately navigates to `/admin/grading`, so the toast unmounts and is never seen (product gap; not fixed here, filed as a separate GitHub issue). The navigation itself is tested.
- FR-BB47 AC-1 (role guard) is covered by route-level tests elsewhere (`src/lib/routeRoles.test.tsx`, `RequireRole.test.tsx`), not by these files.
