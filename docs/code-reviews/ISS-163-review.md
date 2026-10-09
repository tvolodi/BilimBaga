# Code Review: ISS-163 (results CSV export)

Result: PASS

Scope: staged changes in backend/internal/reports (handler.go, repository.go, 3 test files), frontend/e2e/downloads-bearer.spec.ts.

## Findings

- [Medium] backend/internal/reports/handler.go (writeBufferedCSV) — The whole CSV is buffered in memory. This is acceptable for per-exam exports but loses streaming; consider a size note or cap if exports grow large.
- [Medium] backend/internal/reports/handler.go (writeBufferedCSV) — The final `w.Write` error is returned and then triggers `api.WriteError` after headers and status 200 were already sent. That produces a superfluous write and a "superfluous WriteHeader" log on client disconnect. It is harmless but could be logged only.
- [Low] backend/internal/reports/handler.go — `filename` is built from examID/userID taken from the URL. Both are UUID route params, so there is no header injection in practice. Sanitising them would be defence in depth.
- [Low] frontend/e2e/downloads-bearer.spec.ts — The row cell-count check splits on commas and would be fooled by quoted commas. It only asserts >=, so it is safe.

## Verification

- SQL: `session_questions.sort_order` exists (migration 014, INT NOT NULL). `GROUP BY sq.question_id` with `MIN(sq.sort_order)` and `sq.question_id::text` is valid in PostgreSQL. The `min(uuid)` error is removed. Parameterized, no concatenation. Ordering is deterministic (sort_order, then id text).
- Handler: on failure nothing is written to w, so 500 with the `{data:null,error:{code,message}}` envelope via api.WriteError is correct. The error is logged with slog and not leaked. The success path sets the CSV headers and returns 200. Applied to both ExamResultsCSV and UserRecordCSV. RBAC is unchanged (router-level).
- Errors are wrapped; no secrets, `os.Getenv`, or debug output.
- Tests: new handler tests (500 with no CSV headers or partial body, 200 header and data rows), a repository test asserting no aggregate on the uuid column, and a service happy-path test with rows. `go test -p 1 ./internal/reports/` passes (ok).
- E2E: the Bearer assertion is retained and now also asserts status 200, content-type, and a non-empty body with the header row. Not run (per constraints).

## AC Coverage

- Root cause (min(uuid) SQL error): covered
- Failure returns an error status instead of an empty 200: covered
- Regression tests at repo, handler, service and e2e levels: covered

Summary: zero Critical or High findings; the fix is correct, tested, and tests pass.
