---
id: ISS-38
title: Dashboard completion_rate_by_exam omits active exams with no assignments
status: resolved
severity: medium
layer: backend
module: reports
tags: [GetCompletionRateByExam, GetDashboardCompletionRatesForRange, LEFT JOIN, resolved_assignments, FR-BB51]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: backend/internal/reports/repository_completion_test.go
---

## Symptom
FR-BB51 AC-2: an active exam with no assignments is absent from `completion_rate_by_exam` instead of listed with zero counts (docs/handoffs/ba-drift-check/report.md; UAT scenario docs/uat-scenarios/dashboard-completion-rate-20261009.md). GitHub issue #38.

## Root Cause
`GetCompletionRateByExam` (and the sibling `GetDashboardCompletionRatesForRange` used by the PDF export) used `FROM exams e JOIN resolved_assignments ra`, an inner join that drops exams having no resolved assignment rows.

## Fix Applied
Changed both to `LEFT JOIN resolved_assignments ra ON ra.exam_id = e.id`. Filters (`e.status = 'active'`, `e.tenant_id`) are on the base table `exams` so they remain in WHERE without degrading the join; the date-range filter was already in the ON clause of the `exam_sessions` join. COUNT(DISTINCT ra.user_id) yields 0 for NULL rows. Division by zero is already guarded in the consumers (pdf.go and CompletionBarChart.tsx check `assigned_count > 0`); the API returns raw counts only. No migration.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/reports/repository.go | INNER JOIN -> LEFT JOIN in two queries |
| backend/internal/reports/fakedb_test.go | fake database/sql driver (copied from audit pattern) |
| backend/internal/reports/repository_completion_test.go | regression tests |

## Regression Test
repository_completion_test.go: repo-level (zero-count row mapped; SQL asserts LEFT JOIN, no inner join, filters on base table; empty slice), service passthrough, handler JSON zero fields. Caveat: the fake driver does not execute SQL, so the join semantics are asserted on query text, not against Postgres.

## Resolution Results
- Tests: go test -p 2 ./... all packages ok; go vet clean
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
