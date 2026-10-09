---
id: ISS-163
title: Exam results CSV export returns 200 text/csv with empty body
status: resolved
severity: high
layer: backend
module: reports
tags: [min(uuid), GetExamQuestions, StreamExamResultsCSV, ExamResultsCSV, FR-BB54, results/export]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: backend/internal/reports/handler_test.go
---

## Symptom
GET /api/v1/admin/exams/{id}/results/export returns 200 text/csv with an empty body (GitHub #163).

## Root Cause
`GetExamQuestions` (reports/repository.go) used `ROW_NUMBER() OVER (ORDER BY MIN(sq.question_id))`; PostgreSQL has no `min(uuid)` aggregate, so the query failed. The handler had already set Content-Type/Content-Disposition (status 200 implicit), so the service error could only be logged: the client got an empty 200 CSV file.

## Fix Applied
- SQL: order by `MIN(sq.sort_order), sq.question_id::text` (also gives the semantically right question order: the session question order, with a deterministic tie-break).
- Handler: `ExamResultsCSV` and `UserRecordCSV` render into an in-memory buffer (`writeBufferedCSV`); CSV headers/200 are sent only on success, otherwise a 500 JSON error envelope (`INTERNAL_ERROR`).
- Audit: other `min/max/sum/avg` uses in backend/internal are on int/numeric/timestamp columns (sort_order, score, deadline, submitted_at); no other uuid aggregates found.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/reports/repository.go | remove min(uuid) |
| backend/internal/reports/handler.go | buffered CSV, 500 on error |
| backend/internal/reports/*_test.go | regression tests |
| frontend/e2e/downloads-bearer.spec.ts | assert 200, text/csv, header row |

## Regression Test
handler_test.go: TestExamResultsCSV_500_ServiceError_NoCSVHeaders, TestUserRecordCSV_500_..., TestExamResultsCSV_200_HeaderAndDataRows; service_test.go: TestStreamExamResultsCSV_HeaderAndDataRows; repository_completion_test.go: TestGetExamQuestions_NoUUIDAggregate. Real-Postgres verification is left to UAT (label needs-live-db).

## Resolution Results
- Tests: go test -p 1 ./... all pass
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
