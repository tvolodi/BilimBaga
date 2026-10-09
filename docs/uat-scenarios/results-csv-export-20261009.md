---
slug: results-csv-export
title: "Exam Results CSV Export Returns Real Data — UAT Scenario"
feature: results-csv-export (FR-BB54, FR-BB52; GitHub issue #163, follow-up #178)
version: 1
created: 2026-10-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Code that must be merged before running

- **PR #174** (ISS-163, commit `9bf06a7`): **MERGED** in `origin/main`. Fix: `GetExamQuestions` orders by `MIN(sort_order), question_id::text` instead of `MIN(uuid)` (PostgreSQL has no `min(uuid)`); `ExamResultsCSV` and `UserRecordCSV` buffer the CSV and return a 500 JSON error on failure instead of an empty 200; `e2e/downloads-bearer.spec.ts` reads the file from the browser download event.
- Follow-ups in **#178** (developer-filed, not fixed): (1) `auto_submitted` (timed-out) sessions are not in the CSV although per-exam analytics counts them; (2) an unknown but well-formed exam id answers 200. Both are encoded below as **known-gap** checks (S5), recorded as KNOWN-GAP, not new defects.
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
| 4 | SA | DB (or `GET /admin/exams/{id}/analytics`): count sessions `status IN ('submitted','grading_pending')` for `UAT Csv Exam` | 4 (this is `N`, the reference count for S2) | |
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
| 4 | SA | Count data rows (parse with a real CSV parser, not by newline) | Exactly `N` = 4 rows (count equals sessions with status submitted or grading_pending) | |
| 5 | SA | Every row has the same column count as the header | True (RFC 4180 parse, including the quoted name rows) | |
| 6 | SA | Row for EMP-2 and EMP-3 names | `"UAT Csv Two, Jr."` parsed back as one field with a comma; `UAT Csv "Three"` parsed with escaped quotes (`""`) | |
| 7 | SA | Row values | `score_pct` numeric with 2 decimals; `passed` is `true`/`false`; timestamps are UTC ISO 8601; `time_taken_seconds` integer > 0 and consistent with `submitted_at - started_at`; department `UAT Csv Dept` for EMP-1/2, empty for EMP-3 | |
| 8 | SA | EMP-3 grading_pending row | `score_pct`/`passed` follow FR-BB54 AC-4: the short-text question cell is EMPTY (not `0`); other question cells filled | |
| 9 | SA | Cross-check against `GET /admin/exams/{Csv}/analytics` | Per-user pass/fail and scores in CSV equal the values shown for the same sessions (spot check all 4) | |
| 10 | Tester | API log since marker | No `ERROR reports: exam results CSV export failed` and no `min(uuid)`; no stack trace | |
| 11 | SA | Repeat step 1 twice more | Identical header and row count (deterministic order of question columns) | |
| 12 | SA | Add a fifth session (EMP-2, second attempt, submit) and re-export | Rows = 5 (N tracks sessions) | |
| 13 | SA | `GET /admin/exams/{Empty}/results/export` | 200, header row only, 0 data rows (not 0 bytes, not 500) | |
| 14 | SA | Export for an exam with 0 questions if one can exist (draft with no rules) | Either header with 7 fixed columns and 0 rows, or a documented error; never a 200 with an empty body | |

## Scenario S3: Errors surface as errors (no empty 200)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | SA | `GET /admin/exams/not-a-uuid/results/export` | 404 `NOT_FOUND` (malformed path id, api-conventions 2.1) JSON | |
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

## Scenario S5: Known gaps (#178), recorded not failed

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | SA | Let one session on `UAT Csv Timeout Exam` time out so it becomes `auto_submitted`; compare analytics attempts with CSV rows | Analytics counts N+1 while CSV has N rows: record KNOWN-GAP (#178 item 1). If the CSV includes it, PASS | |
| 2 | SA | `GET /admin/exams/{random well-formed UUID}/results/export` | KNOWN-GAP: 200 with header-only CSV (#178 item 2). Desired: 404 `NOT_FOUND`/`ERR_NOT_FOUND`. If 404, PASS | |

## Pass / fail / env-issue criteria

- **PASS:** S1-S4 all as expected; S5 either behaves as desired or exactly as documented (KNOWN-GAP).
- **FAIL (defect):** any 200 with an empty body; wrong header/columns; data row count differs from sessions with status submitted or grading_pending; broken quoting; employee obtains data; missing Bearer yields CSV; an internal error reported as 200; log shows `min(uuid)` error.
- **ENV ISSUE:** stack not reachable; cannot create sessions because exam start fails for reasons outside this feature (route to Infrastructure Configuration or Issue Resolution as appropriate).

## Pre-fix baseline (expected on code before PR #174; already merged in current `main` `a45b92b`)

On pre-fix code, for any exam with attempts: S2 step 1 FAIL (200, 0 bytes, `Content-Disposition` present), steps 2 PASS (headers written first), 3-9 FAIL (no body), step 10 FAIL (`pq: function min(uuid) does not exist` in log), step 13 PASS only for zero-attempt exams where no uuid aggregate runs (header-only 85 bytes), S3 step 2 FAIL (empty 200 instead of 500), S3 step 3 PASS (record export worked: 19 rows in the issue). If a Runner sees this on current `main`, the build predates `9bf06a7`: ENV ISSUE (stale image) first, then DEFECT if rebuilt.

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
| FR-BB54 AC-8 | Streamed, not buffered | **Conflicts with PR #174**: handler now buffers; see gap 1 |
| authenticated-downloads scenario | Bearer required, no cookie/URL token | S1 steps 1, 2, 6 |
| api-conventions 2.1 | `MISSING_TOKEN`, `INVALID_TOKEN`, `FORBIDDEN`, `NOT_FOUND` | S1, S3 |

## Gaps and REQ GAP candidates

1. **FR-BB54 AC-8 contradicts the fix:** it requires direct streaming "not buffered in memory", but PR #174 deliberately buffers to be able to return a 500. Requirement needs to be amended (buffer with a size/row cap, or fetch-first-then-stream).
2. **Row-inclusion rule undefined:** which session statuses appear in the CSV (`submitted`, `grading_pending`, `auto_submitted`, `in_progress`, abandoned) is not in FR-BB54; #178 item 1 is a requirement gap, not only a bug. Also unspecified for superseded retakes.
3. **Unknown exam id:** FR-BB54 does not state 404 vs empty CSV (#178 item 2).
4. **Encoding:** UTF-8 BOM for Excel on Windows with Cyrillic/Kazakh names is unspecified; CSV formula injection (cells starting with `=`, `+`, `-`, `@` in employee names or short-text answers) is unspecified. Candidate security AC: prefix with `'`.
5. **`passed` for `grading_pending`** sessions and the `score_pct` semantics (percent of max vs points) are not defined; S2 step 8 verifies consistency with analytics only.
6. **Department scoping** of this export for `department_admin` is delivered by PR #182, covered in `dept-admin-scoping-20261009.md`.
7. Fault injection (S3 step 2) is not possible on `qa`; mark SKIPPED there.
