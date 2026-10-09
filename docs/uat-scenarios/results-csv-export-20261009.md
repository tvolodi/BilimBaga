---
slug: results-csv-export
title: "Exam Results CSV Export Returns Real Data — UAT Scenario"
feature: results-csv-export (FR-BB54, FR-BB52; GitHub issue #163, follow-up #178)
version: 2 (2026-10-09: PR #194 merged; S5 now asserts the fixes; S6 guard added)
created: 2026-10-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Code that must be merged before running

- **PR #174** (ISS-163, commit `9bf06a7`): **MERGED** in `origin/main`. Fix: `GetExamQuestions` orders by `MIN(sort_order), question_id::text` instead of `MIN(uuid)` (PostgreSQL has no `min(uuid)`); `ExamResultsCSV` and `UserRecordCSV` buffer the CSV and return a 500 JSON error on failure instead of an empty 200; `e2e/downloads-bearer.spec.ts` reads the file from the browser download event.
- **PR #194** (ISS-191, commit `6f3f417`, closes #178 and the formula-injection part of #163): **MERGED**. `auto_submitted` sessions are now in the CSV (status set `submitted`, `auto_submitted`, `grading_pending`), an unknown exam id returns 404 `NOT_FOUND`, and text cells are guarded against spreadsheet formulas on all four CSV exports (S5, S6).
- Real Postgres is required (the original failure appears only on a real database).

## Accounts and data (QA starts with an EMPTY database)

Only `admin@bilimbaga.local` exists (password `{ADMIN_PW}` from the environment; never hard-coded; change it first if `force_password_change=true`).

| Alias | Email | Role | Password after S0 |
|-------|-------|------|-------------------|
| SA | `admin@bilimbaga.local` | super_admin | `{ADMIN_PW}` |
| EXM | `uat.csv.examiner@test.com` | examiner | `UatCsv123!` |
| EMP-1 | `uat.csv.emp1@test.com` (full name `UAT Csv One`) | employee | `UatCsv123!` |
| EMP-2 | `uat.csv.emp2@test.com` (full name `UAT Csv Two, Jr.`; the comma tests CSV quoting) | employee | `UatCsv123!` |
| EMP-3 | `uat.csv.emp3@test.com` (full name `UAT Csv "Three"`; quotes test CSV escaping) | employee | `UatCsv123!` |

Departments: `UAT Csv Dept` (employees EMP-1, EMP-2) and EMP-3 with no department (department cell must be empty, not `null`).

Exams:
- `UAT Csv Exam` (published, 3 active single-choice questions + 1 short-text question so one session stays `grading_pending`): sessions EMP-1 x2 (both submitted, one fail, one pass), EMP-2 x1 (submitted), EMP-3 x1 (short-text answered, left `grading_pending`). Total CSV rows expected = 4 sessions (3 submitted + 1 grading_pending).
- `UAT Csv Empty Exam` (published, assigned, **zero** sessions): header-only expectation.
- `UAT Csv Timeout Exam` (short duration, one session left to auto-submit): used by S5 step 1 only.

Create data through the same API calls as `frontend/e2e/fixtures/seed.ts` (`POST /questions` then `/questions/{id}/status` review then active, `POST /exams`, `/exams/{id}/rules`, `/exams/{id}/publish`, `/exams/{id}/assign`, then portal session create, `PUT answers`, `submit`). New users start with `force_password_change=true`; each must call `POST /auth/change-password` first (required if PR #180 is merged, harmless otherwise). Platform at `http://localhost`, API `http://localhost/api/v1`.

## Scenario S0: Setup

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | SA | Login (change password if forced); create department, users EXM, EMP-1..3 with the full names above; each user changes the temporary password | Accounts active | |
| 2 | SA | Create questions and exams as described; assign `UAT Csv Exam` and `UAT Csv Empty Exam` to all employees | Exams published (`GET /exams/{id}`), assignments listed | |
| 3 | Employees | Take sessions per plan | Session count table: `sessions_by_user` = EMP-1 2, EMP-2 1, EMP-3 1 (grading_pending) | |
| 4 | SA | DB (or `GET /admin/exams/{id}/analytics`): count sessions `status IN ('submitted','auto_submitted','grading_pending')` for `UAT Csv Exam` | 4 (this is `N`, the reference count for S2) | |
| 5 | Tester | Clear API log marker (note current log offset) | For the "no ERROR log" check in S2 | |

## Scenario S1: Bearer required and role checks

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Anonymous | `GET /admin/exams/{Csv}/results/export` with no Authorization header | 401 `MISSING_TOKEN`, JSON body, NOT a CSV | |
| 2 | Anonymous | Same with `Authorization: Bearer garbage` | 401 `INVALID_TOKEN` | |
| 3 | EMP-1 | Same with an employee token | 403 `FORBIDDEN`, JSON body, no CSV content | |
| 4 | EXM | Same with examiner token | 200 (examiner allowed) | |
| 5 | SA | Same with super_admin token | 200 | |
| 6 | SA | Browser navigation (cookie only, no Bearer) to the export URL, i.e. a plain `<a href>` | 401: confirms the UI must download with the Bearer header (FR authenticated-downloads) | |

## Scenario S2: Content of the results CSV (API)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | SA | `GET /admin/exams/{Csv}/results/export` with Bearer; capture status, headers and body bytes | 200; body length > 0 (the pre-fix symptom was 0 bytes) | |
| 2 | SA | Headers | `Content-Type: text/csv; charset=utf-8`; `Content-Disposition: attachment; filename="results-{examId}-{YYYYMMDD}.csv"` where the date is today in UTC; the id equals the requested exam | |
| 3 | SA | Parse the first line | Exactly `employee_name,department,started_at,submitted_at,score_pct,passed,time_taken_seconds,question_1_score,question_2_score,question_3_score,question_4_score` (7 fixed columns + one per exam question, N = 1-indexed position) | |
| 4 | SA | Count data rows (parse with a real CSV parser, not by newline) | Exactly `N` = 4 rows (count equals sessions with status submitted, auto_submitted or grading_pending) | |
| 5 | SA | Every row has the same column count as the header | True (RFC 4180 parse, including the quoted name rows) | |
| 6 | SA | Row for EMP-2 and EMP-3 names | `"UAT Csv Two, Jr."` parsed back as one field with a comma; `UAT Csv "Three"` parsed with escaped quotes (`""`) | |
| 7 | SA | Row values | `score_pct` numeric with 2 decimals; `passed` is `true`/`false`; timestamps are UTC ISO 8601; `time_taken_seconds` integer > 0 and consistent with `submitted_at - started_at`; department `UAT Csv Dept` for EMP-1/2, empty for EMP-3 | |
| 8 | SA | EMP-3 grading_pending row | `score_pct`/`passed` follow FR-BB54 AC-4: the short-text question cell is EMPTY (not `0`); other question cells filled | |
| 9 | SA | Cross-check against `GET /admin/exams/{Csv}/analytics` | Per-user pass/fail and scores in CSV equal the values shown for the same sessions (spot check all 4) | |
| 10 | Tester | API log since marker | No `ERROR reports: exam results CSV export failed` and no `min(uuid)`; no stack trace | |
| 11 | SA | Repeat step 1 twice more | Identical header and row count (deterministic order of question columns) | |
| 12 | SA | Add a fifth session (EMP-2, second attempt, submit) and re-export | Rows = 5 (N tracks sessions) | |
| 13 | SA | `GET /admin/exams/{Empty}/results/export` (exam exists, zero sessions) | 200, header row only, 0 data rows (not 0 bytes, not 500) | |
| 14 | SA | Export for an exam with 0 questions if one can exist (draft with no rules) | Either header with 7 fixed columns and 0 rows, or a documented error; never a 200 with an empty body | |

## Scenario S3: Errors surface as errors (no empty 200)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | SA | `GET /admin/exams/not-a-uuid/results/export` | 404 `NOT_FOUND` JSON is desired. Static review of PR #194 found no UUID check in the handler, so Postgres rejects the id and the likely result is 500 `INTERNAL_ERROR` (gap G2): record the actual status as KNOWN-GAP, PASS if 404 | |
| 2 | SA | Fault injection (optional, local only): temporarily break the query (e.g. revoke SELECT on `session_questions` for the app role, or stop the DB mid-request) and call the export | HTTP 500 with JSON `{data:null,error:{code:"INTERNAL_ERROR"...}}` (or `ERR_INTERNAL`); no `Content-Disposition: attachment`, no 200; restore afterwards | |
| 3 | SA | Same GET for `GET /admin/users/{EMP-1}/record/export` (the sibling export, same buffering fix) | 200 `text/csv`, `filename="record-{userId}-{YYYYMMDD}.csv"`, header `exam_title,started_at,submitted_at,score_pct,passed,time_taken_seconds,status` + 2 rows for EMP-1 | |

## Scenario S4: UI download

Fresh browser context, Super Admin.

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | SA | Open `/admin/reports`; find the row of `UAT Csv Exam`; click its CSV/export button (note: the first button on the page can belong to a zero-session exam, so choose by exam title) | Browser download event fires; file name `results-{examId}-{date}.csv` | |
| 2 | SA | Open the downloaded file | Header + 4 rows, same content as S2 (read the file from the download event; response bodies are consumed as a blob by the app) | |
| 3 | SA | Click the CSV button of `UAT Csv Empty Exam` | File downloaded, header line only (about 85 bytes), no error toast | |
| 4 | SA | Exam analytics page of `UAT Csv Exam`, click "Export" | Same file as step 2 | |
| 5 | SA | Open the CSV in a spreadsheet tool or re-parse | Names with comma/quotes show correctly; Cyrillic/Kazakh name (create `UAT Csv Тест Қазақ`) renders correctly (UTF-8; note whether a BOM is present, see gap 4) | |
| 6 | SA | Failure path in UI (stop API or revoke token before click) | Localized error message; no empty file saved, no crash; en, ru, kk | |
| 7 | EMP-1 | Try to open `/admin/reports` | Redirect to `/login` or `/portal` per route guards; no export reachable | |

## Scenario S5: auto_submitted rows and unknown exam (#178, fixed by PR #194)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | SA | Let one session on `UAT Csv Timeout Exam` time out so it becomes `auto_submitted`; export its results CSV and compare with `GET /admin/exams/{id}/analytics` attempts | CSV includes the timed-out session as a data row; row count equals analytics attempts | |
| 2 | SA | `GET /admin/exams/{random well-formed UUID}/results/export` | 404 `NOT_FOUND`, JSON envelope, no `Content-Disposition`, no CSV body | |
| 3 | SA | Same request with an examiner token and (if available) a `department_admin` token | Same 404 | |
| 4 | SA | `GET /admin/exams/{Empty}/results/export` right after step 2 | Still 200 header-only (an existing exam with no sessions must not become 404) | |

## Scenario S6: Formula-injection guard (FR-BB54 AC-8, PR #194)

Setup: create users whose full names are `=HYPERLINK("http://x","a")`, `+1+1`, `-2+3`, `@SUM(A1)`, and one starting with a TAB character followed by `=1` (via `POST /users`), each with a submitted session on `UAT Csv Exam`; one department named `=cmd|' /C calc'!A0`.

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | SA | Export the results CSV, parse with a real CSV parser | Each dangerous `employee_name` and the `department` value has a single leading `'` (for example `'=HYPERLINK(...)`, `'+1+1`, `'-2+3`, `'@SUM(A1)`, TAB case prefixed); the rest of the value is unchanged | |
| 2 | SA | Same file, numeric columns | `score_pct`, `time_taken_seconds` and `question_N_score` are NOT prefixed (a value like `-1` or `0.00` stays plain); timestamps and `passed` unchanged | |
| 3 | SA | Open the file in Excel or LibreOffice | The dangerous names render as literal text, nothing is evaluated, no external-link or DDE prompt | |
| 4 | SA | Normal names (`UAT Csv One`, Cyrillic name) and empty department of EMP-3 | Unchanged (no `'` added; empty stays empty) | |
| 5 | SA | `GET /admin/users/{dangerous user}/record/export` after giving the user an exam titled `=Exam` | `exam_title` is `'=Exam`; `status` unchanged | |
| 6 | SA | `GET /audit/export` after the user creations above | `actor_name`, `entity_id` and `metadata_json` cells that start with a dangerous character are prefixed with `'`; `timestamp` is not | |
| 7 | SA | `GET /questions/export` with `Accept: text/csv` after creating a question whose stem starts with `=1+1` and a tag `@tag` | Those cells are prefixed with `'` | |
| 8 | SA | Re-import the questions export through `POST /questions/import?dry_run=true` | The prefix is removed on import (stem back to `=1+1`), so export then import round-trips; a stem that genuinely starts with `'=` also round-trips | |
| 9 | Tester | Coverage check: list every CSV download endpoint (`/audit/export`, `/questions/export`, `/admin/exams/{id}/results/export`, `/admin/users/{id}/record/export`) | All four exercised above; there is no users CSV export (import only) | |

## Pass / fail / env-issue criteria

- **PASS:** S1-S6 all as expected; S3 step 1 may be recorded as KNOWN-GAP (gap G2) if it returns 500.
- **FAIL (defect):** any 200 with an empty body; wrong header/columns; data row count differs from sessions with status submitted, auto_submitted or grading_pending; an `auto_submitted` session missing; a dangerous text cell written without the `'` prefix; a numeric cell wrongly prefixed; broken quoting; employee obtains data; missing Bearer yields CSV; an internal error reported as 200; log shows `min(uuid)` error.
- **ENV ISSUE:** stack not reachable; cannot create sessions because exam start fails for reasons outside this feature (route to Infrastructure Configuration or Issue Resolution as appropriate).

## Pre-fix history (informational)

Before PR #174 (`9bf06a7`) the export returned an empty 200 for any exam with attempts (`min(uuid)` error). Before PR #194 (`6f3f417`) auto_submitted sessions were missing, an unknown exam gave a header-only 200 and text cells were written unguarded. Both PRs are merged in `origin/main` (96bf413 and later); a Runner that still sees those symptoms is on a stale image: ENV ISSUE (rebuild) first, then DEFECT.

## Acceptance criteria coverage

| Source | Criterion | Covered by |
|--------|-----------|-----------|
| #163 | Non-empty body for exams with attempts | S2 steps 1, 4, 11, 12; S4 step 2 |
| #163 | Header + one row per session | S2 steps 3-5 |
| #163 | Failures return an error status, not an empty 200 | S3 step 2; S4 step 6 |
| #163 | Record CSV unaffected | S3 step 3 |
| FR-BB54 AC-1 | Employee gets 403 | S1 step 3; S4 step 7 |
| FR-BB54 AC-2 | `Content-Type: text/csv; charset=utf-8`, `Content-Disposition: attachment` | S2 step 2 |
| FR-BB54 AC-3 | Columns and `question_{N}_score` | S2 steps 3, 7 |
| FR-BB54 AC-4 | Ungraded question cell empty, not zero | S2 step 8 |
| FR-BB54 AC-5 | Record CSV columns | S3 step 3 |
| FR-BB54 AC-10 | Deterministic filename | S2 step 2 |
| FR-BB54 AC-8 (amended) | Buffered with cap; statuses; 404; formula guard; no BOM | S2 steps 4, 12; S5; S6. The row/size cap is not implemented (static review, gap G1) and cannot be asserted here without a very large dataset |
| authenticated-downloads scenario | Bearer required, no cookie/URL token | S1 steps 1, 2, 6 |
| api-conventions 2.1 | `MISSING_TOKEN`, `INVALID_TOKEN`, `FORBIDDEN`, `NOT_FOUND` | S1, S3 |

## Gaps and REQ GAP candidates

Status after PR #194 (static review: `docs/requirements/conformance/PR194-PR198-conformance-20261009.md`):

1. **Bounded buffering (G1, open):** FR-BB54 AC-8 amended requires a documented row/size cap; the handler buffers the whole CSV with no cap (`reports/handler.go:155`). Not testable at UAT scale; recorded as a code gap.
2. Row-inclusion rule: RESOLVED. `submitted`, `auto_submitted`, `grading_pending` (`graded` is not a status). Superseded retakes are still listed as separate rows (one row per session).
3. Unknown exam id: RESOLVED (404 `NOT_FOUND`). A malformed id is still likely 500 (G2, S3 step 1).
4. Encoding: formula-injection guard RESOLVED (S6). A UTF-8 BOM is deliberately not required by AC-8; Excel on Windows may mis-detect Cyrillic/Kazakh text when the file is opened by double-click (S4 step 5 records the observation).
5. `passed` for `grading_pending` sessions and the `score_pct` semantics are not defined; S2 step 8 verifies consistency with analytics only.
6. Department scoping of this export for `department_admin` is delivered by PR #182, covered in `dept-admin-scoping-20261009.md`.
7. Fault injection (S3 step 2) is not possible on `qa`; mark SKIPPED there.
