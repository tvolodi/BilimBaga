---
id: ISS-191
title: CSV exports vulnerable to formula injection; results CSV omits auto_submitted sessions and returns 200 for unknown exam (also #178)
status: resolved
severity: high
layer: backend
module: reports
tags: [csv, formula-injection, CWE-1236, auto_submitted, StreamExamResultsCSV, StreamUserRecordCSV, audit export, questions export, "#178", "#163"]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-163]
regression_test: backend/internal/api/csv_test.go, backend/internal/reports/csv_injection_test.go, backend/internal/audit/csv_injection_test.go, backend/internal/questions/csv_injection_test.go
---

## Symptom
- Text cells beginning with `=`, `+`, `-`, `@`, TAB or CR (e.g. a user named `=HYPERLINK(...)`) were written verbatim into CSV exports (FR-BB54 AC-8 amendment, BA decision on #163).
- #178: `GET /admin/exams/{id}/results/export` omitted `auto_submitted` sessions (analytics counts include them) and an unknown exam id returned a 200 header-only CSV.

## Root Cause
Every CSV writer passed raw strings to `encoding/csv`. The results export SQL filtered `status IN ('submitted','grading_pending')` and the service never checked that the exam exists.

## Fix Applied
- New shared helper `internal/api/csv.go`: `CSVSafe` (prefix `'` when a text cell starts with `= + - @ TAB CR`; also escapes a cell that already starts with `'` + dangerous char so it stays invertible), `CSVSafeRecord`, `CSVUnsafe` (inverse, used by the questions CSV importer so export -> import round-trips). Rule documented in the helper: only user-controlled text cells are guarded; server-formatted numeric cells (score_pct, time_taken_seconds, question scores such as `-1`) are never touched.
- Writers covered: results CSV (employee_name, department), user-record CSV (exam_title, status), audit export (all text columns except timestamp), questions export CSV (all cells via `rowToCSV`). There is no separate "users export" endpoint in the codebase; the user-record export is the user-facing one. `users` only has a CSV importer (reader).
- #178 statuses: results export now uses `submitted`, `auto_submitted`, `grading_pending` (identical to the analytics per-exam queries; `graded` named in FR-BB54 AC-8 does not exist in the `session_status` enum). Applied to `StreamExamResultSessions` and `GetExamQuestions` (so question columns include questions only seen in auto-submitted sessions).
- #178 404: `StreamExamResultsCSV` calls `GetExamTitle` first; `ErrNotFound` -> handler replies 404 `NOT_FOUND` (code per FR-BB54 AC-8) before any CSV headers are sent.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/api/csv.go, csv_test.go | new helper + tests |
| backend/internal/reports/service.go | guard text cells, exam existence check |
| backend/internal/reports/repository.go | status filter includes auto_submitted |
| backend/internal/reports/handler.go | ErrNotFound -> 404 |
| backend/internal/audit/handler.go | guard text cells |
| backend/internal/questions/import_export_handler.go | guard on export, CSVUnsafe on import |
| tests: reports/audit/questions `csv_injection_test.go` | per-writer + #178 tests |

## Regression Test
See frontmatter. Covers `=HYPERLINK(...)`, `+1`, `-2+3`, `@SUM(A1)`, `\t=1`, `\r=1`, normal, empty, unicode, numeric cells unchanged, auto_submitted in SQL, unknown exam 404.

## Resolution Results
- Tests: full `go test -p 1 ./...` green; build and vet clean (schemaguard included)
- Migration applied: no
- Build clean: yes
- Needs live DB: yes (SQL status filter) -> label `needs-live-db`

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
