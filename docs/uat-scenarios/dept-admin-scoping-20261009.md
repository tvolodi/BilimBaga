---
slug: dept-admin-scoping
title: "Department Admin Scoping (Record, Record CSV, Exam Analytics, Dashboard) — UAT Scenario"
feature: dept-admin-scoping (FR-BB51, FR-BB52, FR-BB53, FR-BB54; GitHub issue #165; extends analytics-and-reporting S5 / AC-7)
version: 1
created: 2026-10-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Code that must be merged before running

- **PR #182** (ISS-165): `department_admin` scoped to own department subtree (`internal/deptscope`) on record, record export, progress, exam analytics, results CSV, dashboard (+ PDF), AI insights, session result/certificate, grading queue/detail/answers. **OPEN at authoring time (not in `origin/main`, head `a45b92b`).** Until it is merged, run the scenario in **baseline mode** (see "Pre-fix baseline").
- Already merged and required: PR #174 (results CSV non-empty; needed for S4), PR #162 (admin force-change; accounts created below start with `force_password_change=true`).
- PR #180 (server-side force-change enforcement) is OPEN; if it is merged, every account created in S0 must call `POST /auth/change-password` before any other call (step S0.6 covers this either way).
- Follow-ups #183, #184 (filed by the developer, content not part of this scenario) are out of scope.
- Migrations apply at API startup. A rebuilt API is sufficient. Live-DB behaviour (recursive subtree SQL, dashboard HAVING) was **not verified** by the developer; this scenario is its first real-Postgres check.

## Decision applied (from issue #165 comment, proposal by developer)

Exam-level analytics for `department_admin`: aggregates are computed over participants of the caller's own department subtree only. `super_admin` (and `examiner`) unchanged: see everything. The issue text says "403, or scoped lists"; this scenario accepts, per surface, exactly what is written in the expected column. If the product owner chooses "aggregate across all" for exam analytics instead, S3 must be revised (REQ GAP 1).

## Accounts and data (QA starts with an EMPTY database)

Only `admin@bilimbaga.local` exists. Its current password is `{ADMIN_PW}` (supplied to the Runner from environment/secret store; never hard-code, never commit). If `force_password_change=true` for it, change it first (S0.1). All other accounts below are created by S0.

| Alias | Email | Role | Department | Password after S0 |
|-------|-------|------|-----------|-------------------|
| SA | `admin@bilimbaga.local` | super_admin | n/a | `{ADMIN_PW}` |
| DA-A | `uat.da.a@test.com` | department_admin | Dept A | `UatScope123!` |
| DA-B | `uat.da.b@test.com` | department_admin | Dept B | `UatScope123!` |
| EMP-A1 | `uat.emp.a1@test.com` | employee | Dept A | `UatScope123!` |
| EMP-A2 | `uat.emp.a2@test.com` | employee | Dept A1 (child of A) | `UatScope123!` |
| EMP-B1 | `uat.emp.b1@test.com` | employee | Dept B | `UatScope123!` |
| EMP-B2 | `uat.emp.b2@test.com` | employee | Dept B | `UatScope123!` |
| EXM | `uat.examiner@test.com` | examiner | none | `UatScope123!` |

Departments: `UAT Dept A`, `UAT Dept A1` (parent = Dept A), `UAT Dept B`. Platform at `http://localhost`, API at `http://localhost/api/v1` (QA: its public URL). `ids` below mean UUIDs captured at creation.

Exam data: two published exams, both assigned to all four employees:
- `UAT Scope Exam 1` (shared exam: both departments have attempts)
- `UAT Scope Exam 2` (attempted only by EMP-B1/EMP-B2, so Dept A has zero participants)

Session counts to create (all submitted, passing): EMP-A1 x2 on Exam 1, EMP-A2 x1 on Exam 1, EMP-B1 x3 on Exam 1 + x1 on Exam 2, EMP-B2 x2 on Exam 1 + x1 on Exam 2. Totals: Exam 1 = 8 sessions (A-subtree 3, B 5); Exam 2 = 2 sessions (A 0, B 2). Add one overdue assignment (deadline in the past, no session) for EMP-A1 and one for EMP-B2 on a third exam `UAT Scope Overdue`.

## Scenario S0: Setup

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | SA | `POST /auth/login` as admin; if `data.user.force_password_change=true`, `POST /auth/change-password` | Token obtained; flag cleared | |
| 2 | SA | `POST /departments {name:"UAT Dept A"}`, `{name:"UAT Dept B"}`, then `{name:"UAT Dept A1", parent_id:<A>}` | 201 each; ids captured (`GET /departments` shows the tree) | |
| 3 | SA | `GET /users/roles`; capture ids for employee, department_admin, examiner | Roles listed | |
| 4 | SA | `POST /users` for each account in the table (`email`, `full_name`, `role_id`, `department_id`) | 201; response carries `temporary_password` | |
| 5 | Each | `POST /auth/login` with the temporary password, then `POST /auth/change-password` to `UatScope123!` | 200; later login returns `force_password_change=false` | |
| 6 | SA | Create the question(s), exams, rules, publish and assign exactly as `frontend/e2e/fixtures/seed.ts` does (`POST /questions`, `/questions/{id}/status` to review then active, `POST /exams`, `/exams/{id}/rules`, `/exams/{id}/publish`, `/exams/{id}/assign`); set a past deadline on the Overdue assignments | Exams published; assignments exist (`GET /exams/{id}/assignments`) | |
| 7 | Employees | Take each exam through the portal API (`POST /portal/exams/{id}/sessions`, `PUT .../answers/{qid}`, `POST .../submit`) to produce the session counts above | Session counts match the plan; record in a table `sessions_by_user` | |
| 8 | SA | Capture ground truth: `GET /admin/users/{id}/record` for each employee, `GET /admin/exams/{id}/analytics` for Exam 1 and 2, `GET /admin/dashboard` | Baseline totals recorded (Exam 1 attempts 8, Exam 2 attempts 2, 10 sessions total) | |

## Scenario S1: Employee record and record CSV (API)

Login as DA-A (token `TA`).

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | DA-A | `GET /users` | Only Dept A and A1 users (EMP-A1, EMP-A2, DA-A); no Dept B user (control: this already worked pre-fix) | |
| 2 | DA-A | `GET /admin/users/{EMP-A1}/record` | 200; 2 sessions | |
| 3 | DA-A | `GET /admin/users/{EMP-A2}/record` (child department) | 200; 1 session (descendants included) | |
| 4 | DA-A | `GET /admin/users/{EMP-B1}/record` | 403 `FORBIDDEN` (or 404 `NOT_FOUND`); no session data in the body | |
| 5 | DA-A | `GET /admin/users/{EMP-B1}/progress` | 403 or 404, no data | |
| 6 | DA-A | `GET /admin/users/{EMP-A1}/record/export` | 200 `text/csv`, `Content-Disposition: attachment; filename="record-{id}-{YYYYMMDD}.csv"`; header + 2 rows | |
| 7 | DA-A | `GET /admin/users/{EMP-B1}/record/export` | 403 or 404; body is NOT CSV and contains no session row | |
| 8 | DA-B | Symmetric: DA-B fetches EMP-A1 record and record/export | 403 or 404; DA-B own EMP-B1 record returns 4 sessions | |
| 9 | DA-A | Record/export/progress for a random valid UUID not in DB; and for malformed id `not-a-uuid` | 404 `NOT_FOUND`; malformed id 404 (path id, per api-conventions section 2.1); no 500 | |
| 10 | SA | Records of EMP-A1 and EMP-B1 | 200 each, full data (unchanged) | |
| 11 | EXM | Record of EMP-B1 | 200 (examiner unchanged per PR #182 note) | |

## Scenario S2: Per-exam analytics and results CSV (API)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | DA-A | `GET /admin/exams/{Exam1}/analytics` | 200; attempts/participants computed over Dept A subtree only: total attempts = 3, unique participants = 2; NOT 8 | |
| 2 | DA-A | `GET /admin/exams/{Exam2}/analytics` | 200 with zero attempts (empty/zero state, no error), or 403/404; in no case 2 attempts or any Dept B figure | |
| 3 | DA-A | `GET /admin/exams/{Exam1}/results/export` | 200 `text/csv`; header + exactly 3 data rows (EMP-A1 x2, EMP-A2 x1); no EMP-B1/B2 name in any row | |
| 4 | DA-A | `GET /admin/exams/{Exam2}/results/export` | Header only (0 rows) or 403/404; no Dept B row | |
| 5 | DA-B | Exam 1 analytics and CSV | attempts = 5; CSV header + 5 rows, all Dept B names | |
| 6 | SA | Exam 1 analytics and CSV | attempts = 8; CSV header + 8 rows (unchanged) | |
| 7 | Any scoped | Per-question statistics in step 1 | Correct-rate/average-time figures derive only from Dept A attempts (spot check: differ from SA figures when B answers differ; if identical by chance, change one B answer and rerun) | |

## Scenario S3: Dashboard (API)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | DA-A | `GET /admin/dashboard` | 200; assigned/completed totals, exam completion bars, recent activity and overdue list contain Dept A subtree employees only (recent activity: 3 sessions, only A names; overdue: EMP-A1 only, not EMP-B2) | |
| 2 | DA-A | Compare with SA response | Differs from SA (SA overdue contains both EMP-A1 and EMP-B2; recent activity 10 entries) | |
| 3 | DA-A | Exam list in dashboard completion chart | Exams with zero Dept A participants either absent or shown at 0%; no counts from Dept B | |
| 4 | DA-A | `GET /admin/dashboard/export` (PDF) | 200 `application/pdf`; text of the PDF (extract) shows no Dept B employee names and tables agree with step 1 totals | |
| 5 | DA-B | `GET /admin/dashboard` | Dept B data only (recent activity 7 sessions; overdue EMP-B2 only) | |
| 6 | SA | `GET /admin/dashboard` | All data (10 sessions, both overdue employees) | |
| 7 | Anonymous | Any endpoint of S1-S3 without `Authorization` | 401 `MISSING_TOKEN` | |
| 8 | EMP-A1 | Any endpoint of S1-S3 | 403 `FORBIDDEN` | |

## Scenario S4: UI surfaces

Fresh browser context per role.

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | DA-A | Log in, open `/admin` (dashboard) | KPI cards, charts, recent activity, overdue table show Dept A subtree only; matches S3 step 1 | |
| 2 | DA-A | `/admin/users`, open "View record" for EMP-A1 | `/admin/users/{id}/record` shows 2 sessions; "Export" downloads CSV with header + 2 rows | |
| 3 | DA-A | Browse directly to `/admin/users/{EMP-B1}/record` | No Dept B session data rendered; localized error/forbidden state, no crash, no blank flash of data | |
| 4 | DA-A | `/admin/reports` then the exam analytics page of Exam 1 | Attempts = 3; histogram and per-question table based on Dept A; "CSV" download has header + 3 rows | |
| 5 | DA-A | Browse directly to the analytics page of Exam 2 | Zero-state or forbidden message; no Dept B figures | |
| 6 | DA-B | Repeat 1-4 with Dept B ids | Dept B only | |
| 7 | SA | Repeat 1, 2, 4 | Full data | |
| 8 | DA-A | Switch locale en, ru, kk on the dashboard and the denied state | No raw i18n keys | |

## Pass / fail / env-issue criteria

- **PASS:** every S1-S3 expectation holds in API and UI, with the exact counts above, and SA/EXM views are unchanged.
- **FAIL (defect):** DA-A obtains any Dept B record, record CSV row, exam participant figure, results CSV row, dashboard row or PDF content (this is the issue #165 symptom); SA/EXM view loses data; a 500 or empty-200 on any scoped endpoint; DA-A loses access to its own subtree (e.g. EMP-A2 in child department hidden).
- **ENV ISSUE:** stack unreachable; seed of S0.6-S0.7 fails for reasons unrelated to scoping (route to Infrastructure Configuration); recursive-subtree SQL error with a Postgres message (this is a defect, not an env issue, because the developer could not verify it live).

## Pre-fix baseline (before PR #182 merges, expected observations on `main` `a45b92b`)

S1 steps 1 PASS (already scoped); step 4 and 7 FAIL (200 with Dept B data), step 5 FAIL, step 6 PASS only for own user; S2 steps 1 and 3 FAIL (attempts 8, CSV 8 rows, all departments); S3 step 1 FAIL (identical to SA); step 4 FAIL; S4 steps 3 and 5 FAIL. SA, EXM, employee-403 and 401 steps PASS. A Runner on pre-fix code should record these as "baseline reproduced" and mark the scenario BLOCKED on PR #182, not DEFECT-new.

## Acceptance criteria coverage

| Source | Criterion | Covered by |
|--------|-----------|-----------|
| analytics S5 / AC-7 | Department Admin cannot see other departments' data | S1, S2, S3, S4 |
| Issue #165 (1) | Record of another department's employee refused/scoped | S1 steps 4, 5, 8 |
| Issue #165 (2) | Record CSV scoped | S1 steps 6, 7 |
| Issue #165 (3) | Per-exam analytics scoped to department participants | S2 steps 1, 2, 5, 7 |
| Issue #165 (4) | Dashboard recent activity / overdue / totals scoped | S3 steps 1-6, S4 step 1 |
| PR #182 note | Descendant departments included | S1 step 3 |
| PR #182 note | Results CSV, PDF scoped | S2 steps 3, 4; S3 step 4 |
| PR #182 note | super_admin / examiner unchanged | S1 steps 10, 11; S2 step 6; S3 step 6 |
| FR-BB54 AC-1 | Employee 403 | S3 step 8 |
| FR-BB54 AC-2, AC-10 | CSV Content-Type, filename | S1 step 6, S2 step 3 |
| api-conventions 2.1 | 401 `MISSING_TOKEN`, 403 `FORBIDDEN`, 404 `NOT_FOUND` | S1 steps 4, 9; S3 steps 7, 8 |

## Gaps and REQ GAP candidates

1. **Exam-analytics policy is only a developer proposal** (issue comment), not in FR-BB52 or FR-BB51. Document "department_admin sees own-subtree aggregates" in FR-BB52, plus what a zero-participant exam returns (zero-state 200 vs 403/404).
2. **Scoped-denied status undefined:** 403 vs 404 for another department's record is not specified anywhere (FR-BB53 lacks it). The scenario accepts either; the requirement should pick one (information disclosure argues for 404).
3. **Role name drift:** PR #182 lists `hr_admin` as unchanged; route-guards scenario states `hr_admin` was renamed to `examiner` (migration 005). Requirement text should drop `hr_admin`.
4. **department_admin with no department** (NULL): behaviour unspecified (no data vs everything). Add as FR-BB53 AC.
5. FR-BB51/53/54 contain no department-scoping AC at all (they say "scoped to tenant"); AC-7 exists only in the analytics scenario.
6. S0 bodies for exams/questions are defined by `seed.ts` rather than a requirement doc; if seed.ts changes, S0.6 drifts.
