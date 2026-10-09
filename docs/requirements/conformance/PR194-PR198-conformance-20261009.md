# Conformance Review: PR #194, PR #198

- Date: 2026-10-09
- Reviewer: Business Analyst (static code review; no live stack, no UAT executed, tests not re-run)
- Base: origin/main at 96bf413 (worktree `ba`, branch swarm/ba-sync-194-198)
- Scope: PR #194 (6f3f417, ISS-191, GitHub #178 / formula-injection part of #163), PR #198 (96bf413, ISS-146 `make migrate`).
- Compared against: FR-BB54 AC-8 (amended), FR-BB12 AC-1/AC-4/AC-7, FR-BB11 AC-7, `api-conventions.md` sections 9 and 14, UAT scenario `results-csv-export-20261009.md`.
- Severity: p1 only for security, data exposure or a broken core flow; p2 behaviour/spec defect; p3 minor or documentation.

**p1 gaps: none.**

## 1. PR #194 against FR-BB54 AC-8

| # | Requirement | Result | Evidence | Severity |
|---|-------------|--------|----------|----------|
| 1 | Formula guard on results export (`employee_name`, `department`) | PASS | `reports/service.go:434-435` | - |
| 2 | Guard on user-record export (`exam_title`, `status`) | PASS | `reports/service.go:511,517` | - |
| 3 | Guard on audit export (all text columns, timestamp excluded) | PASS | `audit/handler.go:119-125` | - |
| 4 | Guard on questions CSV export (all cells) | PASS | `questions/import_export_handler.go:466` (`CSVSafeRecord`) | - |
| 5 | Users export | N/A, not covered because it does not exist: only `POST /users/import` reads CSV (`users/handler.go:307`) | - | - |
| 6 | Guard characters `= + - @` | PASS, wider: also TAB and CR; prefix `'` | `api/csv.go:7,26-37` | - |
| 7 | Numeric / server-formatted cells not mangled | PASS (`score_pct`, `time_taken_seconds`, question scores raw) | `reports/service.go:431-445` | - |
| 8 | Round trip export then import | PASS for questions CSV (`CSVUnsafe` on import). Note: a cell that starts with `'` plus a dangerous char is double-escaped by design | `questions/import_export_handler.go:263`, `api/csv.go:26-57` | - |
| 9 | Statuses `submitted`, `auto_submitted`, `graded` | PASS with deviation: `graded` is not a `session_status` value; `grading_pending` is included instead (matches analytics) | `reports/repository.go:783,813`; `GetExamQuestions` also updated | p3 (spec wording) |
| 10 | Unknown exam id gives 404 `NOT_FOUND` | PASS for a well-formed id | `reports/handler.go:183-186`, `service.go:342`, `repository.go:448-458` | - |
| 11 | 404 for a malformed (non-UUID) id | GAP (G2): no UUID validation; Postgres rejects `id = $1` and the error is wrapped (`service.go:346`), so the handler answers 500 `INTERNAL_ERROR`. Not verified live | `reports/handler.go:169-189` | p2 |
| 12 | Exam lookup is tenant-scoped | GAP (G4): `GetExamTitle` is `WHERE id = $1` with no tenant or department scope, so an exam of another tenant/department is "found" and yields a header-only 200 instead of 404. Single-tenant deployments are unaffected; no data is leaked (rows stay scoped by `@SCOPE@`, only existence is confirmed) | `reports/repository.go:448-450` | p3 |
| 13 | Bounded buffering: results export has a documented page/row cap | GAP (G1): the whole result set is read into slices and a `bytes.Buffer` with no cap; the service comment still says "without buffering the full dataset" | `reports/handler.go:140-165`, `reports/service.go:336-338,377-396` | p2 |
| 14 | Failure after headers sent is logged | PASS: buffered path sends nothing on failure and logs `slog.Error`; audit/questions exports still stream (header first, write errors ignored: `audit/handler.go:96,117`) | `reports/handler.go:186-188` | p3 |
| 15 | No BOM | PASS (none written) | `service.go:361` | - |
| 16 | Leading-space / other-prefix values (for example `" =1+1"`, `|cmd`) | By design not guarded; spreadsheets do not evaluate them. `|` and `%` (DDE-era) are not in the AC list | `api/csv.go:7-22` | p3 |
| 17 | Tests | Present for helper and each writer (`api/csv_test.go`, `reports|audit|questions/csv_injection_test.go`); SQL status filter needs a live DB (label `needs-live-db`) | commit 6f3f417 | - |

### Not covered / residual
- Questions JSON export (not CSV, so not applicable). Short-text answers are not part of any CSV export, so the guard does not apply to them.
- `GET /admin/dashboard/export` is PDF.
- Frontend: nothing else produces CSV; no client-side CSV builder was found in the docs (not re-grepped in code).
- Header rows are static strings and need no guard.

## 2. PR #198 against FR-BB11 / FR-BB12

| # | Requirement | Result | Evidence | Severity |
|---|-------------|--------|----------|----------|
| 1 | FR-BB11 AC-7 `migrate` target runs pending migrations | PASS now | `Makefile:14-15`; image `ENTRYPOINT ["/app/api"]` (`backend/Dockerfile:26`) so the argument is `migrate` | - |
| 2 | FR-BB12 AC-1 startup migration still happens before listening | PASS, unchanged | `backend/cmd/api/main.go:92` | - |
| 3 | `migrate` never starts server, schedulers, bootstrap | PASS | `cmd/api/run.go:31-43,70-86` | - |
| 4 | Exit codes: 0 ok, 1 failure, 2 usage | PASS | `run.go:31-47`, `run_test.go` | - |
| 5 | FR-BB12 AC-7 exit 0 on up-to-date DB | PASS (`ErrNoChange` is success) | `internal/db/migrate.go:32-34` | - |
| 6 | FR-BB12 AC-7 logs "no new migrations" | GAP (G3): `migrate` always prints `migrations applied`; `RunMigrations` returns nil silently on no change. Test strategy line "logs a no-change message" is also unmet | `run.go:40`, `db/migrate.go:32` | p3 |
| 7 | FR-BB12 AC-9 dirty flag aborts | PASS, also on the subcommand (error propagates, exit 1) | `db/migrate.go:35-38`, `run.go:36-38` | - |
| 8 | Docs stale on "no migrate subcommand" | FIXED in this review: `api-conventions.md` section 14 row `make migrate`; FR-BB11 sample Makefile (`./bin/api migrate` was wrong: binary is `/app/api`); FR-BB12 AC-7 status note. README was already updated by the PR | docs | - |
| 9 | Mentions of `make migrate` that remain valid | FR-BB16 AC-3, FR-BB65 prerequisites, UAT scenarios `role-management`, `overdue-reminder`, `account-recovery` (the last says "`make migrate` is unreliable per README": now outdated, but harmless, and that file is outside this task's edit list) | grep | p3 |

## 3. Gap list

| ID | Gap | Severity | Suggested owner |
|----|-----|----------|-----------------|
| G1 | Results CSV buffering has no row/size cap (AC-8 "bounded"); stale "without buffering" comment | p2 | Issue Resolution: cap (for example a maximum row count returning 413/422, or stream after a first-page probe) |
| G2 | Non-UUID exam id on `/admin/exams/{id}/results/export` is likely 500, not 404 (same class as other `{id}` routes) | p2 | Issue Resolution: validate UUID, return 404 `NOT_FOUND` |
| G3 | `migrate` does not report "no new migrations" (FR-BB12 AC-7) | p3 | Issue Resolution or accept and amend AC-7 |
| G4 | `GetExamTitle` is not tenant/department scoped (existence oracle only) | p3 | with G2 |
| G5 | `graded` in FR-BB54 AC-8 is not a real status | p3 | requirement wording: replace with `grading_pending` (done in the AC-8 status note) |

## 4. Not verifiable statically
- Actual HTTP status for a non-UUID id (G2) and the `auto_submitted` SQL filter against real data (needs live Postgres).
- That Excel / LibreOffice render the `'` prefix as literal text without displaying the apostrophe in all versions (UAT S6 step 3).
- `docker compose run --rm api migrate` end to end (depends on the built image and compose `entrypoint`; compose file was not inspected for an `entrypoint` or `command` override, only the Dockerfile).
- Whether department_admin scoping changes the 404 behaviour for an exam outside its subtree.

## 5. Documents changed in this review
`docs/requirements/FR-BB54.Export-API.md` (AC-8 status), `docs/requirements/api-conventions.md` (section 9 CSV rules, section 14 `make migrate`), `docs/requirements/FR-BB11.Project-scaffold.md` (sample Makefile), `docs/requirements/FR-BB12.Database-bootstrap.md` (AC-7 status), `docs/uat-scenarios/results-csv-export-20261009.md` (v2: baseline notes removed, S5 rewritten, S6 added), this report.
