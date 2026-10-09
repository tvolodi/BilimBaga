---
id: ISS-75
title: Dashboard PDF export returns 500 "failed to build dashboard report"
status: resolved
severity: high
layer: backend
module: reports
tags: [tenant_id, "column does not exist", BuildDashboardReport, GetDashboardCompletionRatesForRange, GetTopBottomQuestions, dashboard/export]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: backend/internal/reports/schema_guard_test.go
---

## Symptom
GET /api/v1/admin/dashboard/export -> 500 `{INTERNAL_ERROR "failed to build dashboard report"}` for super_admin, department_admin, examiner. The error was not logged (handler.go DashboardExportPDF).

## Root Cause
SQL referenced a `tenant_id` column that does not exist on `exams` or `exam_sessions`. The only migrated `tenant_id` column in all 30 up-migrations (incl. every ALTER TABLE) is `audit_log.tenant_id` (007_audit_log.up.sql:7). exams = 012_exams.up.sql (+026 adaptive), exam_sessions = 014 (+015, 018, 026): none has it. Postgres answers `column "tenant_id" does not exist` -> BuildDashboardReport (service.go:493-500) wraps it -> handler returned a generic 500 without logging.

Offending predicates (repository.go, before fix):
- GetDashboardCompletionRatesForRange: `WHERE e.tenant_id = $1` (~line 888)  -> first query BuildDashboardReport runs
- GetTopBottomQuestions: `WHERE es.tenant_id = $1` (~line 943)
- Same bug class in the CSV exports (FR-BB54): GetExamQuestions (~753), StreamExamResultSessions (~783), StreamUserRecordSessions (~838) used `es.tenant_id = $2`. Their handler errors were also swallowed (`_ = err`).

Tenancy reality: single tenant per deployment. `auth.TenantContext()` (router.go:49) injects the constant "public"; tenant branding lives in the `tenant_config` key/value table (002), not per-row. So row-level tenant predicates have no meaning for exams/sessions.

Ruled out: empty DB (queries return empty slices, PDF builder handles empty data; handler test covers), nil pointers, PDF resources (pdf.go uses built-in Helvetica only, no font files; Dockerfile copies binary + migrations only, nothing else needed), all other columns in the dashboard queries (exams.title/status, exam_assignments, exam_sessions.status/passed/submitted_at, 'grading_pending' enum value (015), session_question_scores.score/max_score, question_translations.stem, questions.default_locale) match the migrated schema.

## Fix Applied
- Removed the tenant predicates from the 5 queries and renumbered placeholders (Postgres rejects unused parameters); interface signatures keep `tenantID` (documented as unused) to avoid touching mocks/service.
- handler.go: log (slog.Error) BuildDashboardReport, PDF generation and the two CSV stream errors.
- Delivered on the PR #77 branch (swarm/38-reports-left-join) on top of the #38 LEFT JOIN change; the two completion-rate queries now carry both changes.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/reports/repository.go | drop tenant_id predicates, renumber args |
| backend/internal/reports/handler.go | log swallowed errors |
| backend/internal/reports/schema_guard_test.go | new schema guard |
| backend/internal/reports/handler_test.go | error-logged 500 test, PDF body 200 test |
| frontend/e2e/downloads-bearer.spec.ts | assert 200 + application/pdf |

## Regression Test
schema_guard_test.go scans migrations and fails if repository.go references tenant_id. handler_test.go: TestDashboardExportPDF_500_ErrorIsLogged, TestDashboardExportPDF_200_PDFBody.

## Resolution Results
- Tests: go vet clean; go test -p 2 ./... all packages ok; tsc --noEmit clean
- Migration applied: no
- Build clean: yes
- Not verified: live DB (no stack allowed). Also found, NOT fixed: backend/internal/ai/repository.go:178 `SELECT ... tenant_id FROM exams` (GetExamInsightData) has the same nonexistent-column defect.

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
