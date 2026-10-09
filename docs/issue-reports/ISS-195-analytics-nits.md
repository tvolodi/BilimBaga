---
id: ISS-195
title: Analytics/CSV consistency nits (grouped, from PR #194 review)
status: resolved
severity: low
layer: backend
module: reports
tags: [auto_submitted, completion-rate, dashboard, user-progress]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-165, ISS-191]
regression_test: backend/internal/reports/repository_autosubmitted_test.go
---

## Symptom
Dashboard, user-progress and completion-rate queries counted only `submitted`/`grading_pending` sessions,
while results CSV/analytics (PR #194) also include `auto_submitted`.

## Root Cause
Hard-coded status lists in `backend/internal/reports/repository.go` were never updated when `auto_submitted` was added to the CSV/analytics queries.
Real `session_status` enum values (migrations 014, 015): `in_progress`, `submitted`, `auto_submitted`, `grading_pending`.

## Fix Applied
Seven status predicates now use `IN ('submitted','auto_submitted','grading_pending')`: GetCompletionRateByExam,
GetRecentActivity, GetAvgScoreByTrack (was `= 'submitted'`), GetUserTrackActivity, GetUserRequiredExams,
GetDashboardCompletionRatesForRange, GetTopBottomQuestions. deptscope placeholders untouched. No migration.

## Per-item outcome
| # | Item | Outcome |
|---|------|---------|
| 1 | Completed-definition consistency | Done, tests added; SQL unverified against live Postgres (needs-live-db) |
| 2 | FR-BB54 AC-8 doc lists non-existent `graded` status | Code audit: no session-status `graded` in code or tests. `GradingStatusGraded` in sessions/grading.go is a per-answer grading state, not a session status. Doc fix left to BA |
| 3 | Exam-existence check lacks tenant filter | Not applicable: single tenant per deployment, `exams` has no tenant_id column; none added; reports code does not reference it; schemaguard green |
| 4 | Unknown user gives empty record CSV | Already done in PR #182; covered by handler_test.go (USER_NOT_FOUND, ~line 497/593) and scope_test.go:174 |

## Files Changed
| File | Change |
|------|--------|
| backend/internal/reports/repository.go | status predicates include auto_submitted |
| backend/internal/reports/repository_autosubmitted_test.go | new: SQL assertions for 7 queries, service pass-through |

## Regression Test
repository_autosubmitted_test.go (fake-driver SQL text assertions; no legacy list, no tenant_id, no 'graded', scope expanded).

## Resolution Results
- Tests: go test -p 1 ./... all pass; go vet and staticcheck clean
- Migration applied: no
- Build clean: yes
