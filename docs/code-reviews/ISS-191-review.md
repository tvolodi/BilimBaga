# Code Review: ISS-191 (CSV formula-injection guard) + #178 (results CSV statuses / 404)

Run ID: iss-191 | Reviewer: Code Reviewer subagent | Date: 2026-10-09

## Result: PASS

`go vet` and `go test -p 1` pass for `internal/api`, `internal/reports`, `internal/audit`, `internal/questions`.

## Findings

- [Medium] backend/internal/reports/repository.go:430 (GetExamTitle), service.go:316 - Tenant scoping: the new existence check is `SELECT title FROM exams WHERE id=$1` with no tenant filter, and `StreamExamResultSessions` (line ~785) also ignores `tenantID`. An examiner of tenant A can distinguish "exists in another tenant" (200) from "does not exist" (404) and, for the sessions query, export another tenant's results if tenant isolation relies on schema-per-tenant search_path this is a non-issue, otherwise it is a leak. Pre-existing pattern (GetExamTitle is used the same way at service.go:143), so not introduced here. -> Verify tenancy model; if shared schema, scope both queries by tenant.
- [Medium] backend/internal/reports/repository.go lines 179, 329, 379, 689, 730, 879, 942 - Status filter consistency: the per-exam analytics queries (461-576) use `('submitted','auto_submitted','grading_pending')` and now match the results export, but the dashboard / user-progress / completion-rate queries still use `('submitted','grading_pending')` (line 379 only `'submitted'`). The issue report's claim "identical to the analytics queries" is true only for the per-exam set. Pre-existing divergence; not blocking. -> Follow-up issue to unify via a shared constant.
- [Low] FR-BB54 AC-8 lists `graded` as an included status; `session_status` enum has no `graded` (only `in_progress|submitted|auto_submitted`). Implementation correctly omits it; recommend amending the AC text.
- [Low] backend/internal/reports/service.go:316 - Extra DB round trip and small TOCTOU between exam lookup and stream; acceptable. User-record export for an unknown user still returns an empty CSV (AC only mandates 404 for unknown exam).
- [Low] backend/internal/api/csv.go CSVUnsafe - A hand-authored import cell that legitimately begins with `'` + `= + - @` loses its quote. Documented invertible design, acceptable; round-trip test covers exported data.
- [Low] backend/internal/audit/handler.go - `csvWriter.Write` errors still ignored (pre-existing).
- [Low] Frontend: `downloadFile` surfaces the 404 code (`NOT_FOUND`) but `downloadErrorKey` maps everything except 401 to the generic `download.failed` message. Works, just not specific. No change required.
- [Low] Tests: `TestResultsExportQueries_IncludeAutoSubmitted` asserts on the SQL string via the fake DB (verifies text, not behavior); no handler-through-real-service 404 test (handler test uses mock service, service test uses mock repo; together they cover the path). No test that audit timestamp column is untouched beyond format; fine.

## Checklist notes

1. Guard completeness: all `csv.NewWriter` sites in backend/internal are covered: reports/service.go (results, user record), audit/handler.go, questions/import_export_handler.go. `users/handler.go` only uses `csv.NewReader` (import). No frontend code builds CSV (frontend only downloads blobs). Static headers are not guarded (constants).
2. Numeric cells: score_pct, passed, time_taken_seconds, per-question scores (e.g. `-1`) and timestamps are written raw; test asserts `-1` stays `-1`. Questions export `correct` column is digits/commas and unaffected by the guard.
3. Consumers: questions importer applies `CSVUnsafe` to every cell, so export -> import round-trips (tested). No e2e/seed code parses these CSVs (grep of frontend/e2e and e2e found none). Frontend only downloads. Docs: FR-BB54 AC-8 already prescribes the `'` prefix.
4. Status filter: see Medium finding above; `GetExamQuestions` and `StreamExamResultSessions` are consistent with each other and with per-exam analytics.
5. 404: `ErrNotFound` -> 404 `NOT_FOUND` matches FR-BB54 AC-8; returned before any CSV headers via `writeBufferedCSV` (buffered, nothing written on error). Frontend handles via generic error alert.
6. Test quality: good coverage (unit for helper, per-writer integration with injection payloads, TAB/CR, Unicode, empty, round trip, input non-mutation). Minor gaps listed in Low.

## AC Coverage (FR-BB54 AC-8 amendment)
- CSV via encoding/csv: covered
- Statuses submitted/auto_submitted included: covered (`graded` n/a)
- Unknown exam id -> 404 NOT_FOUND: covered
- Cells starting `= + - @` prefixed with `'`: covered (also TAB/CR)

## Summary
No Critical or High findings; implementation is complete and correct across all CSV writers, with only pre-existing tenancy/status-consistency items recommended as follow-ups.
