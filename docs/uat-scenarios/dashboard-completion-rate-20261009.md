---
slug: dashboard-completion-rate
title: "Dashboard completion_rate_by_exam Includes Unassigned Active Exams — UAT Scenario"
feature: dashboard-completion-rate (FR-BB51 AC-2; GitHub issue #38, parent #34)
version: 1
created: 2026-10-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Code that must be merged before running

- **Issue #38** (`fix(reports): completion_rate_by_exam omits active exams without assignments`; LEFT JOIN in `GetCompletionRateByExam` plus test). Status ready, no PR at authoring time.
- Rebuild the API. No frontend change is required; the UI effect (S4) relies on `AdminDashboardPage.tsx`, which uses `completion_rate_by_exam.length` as the active-exam count and renders the chart only when the list is non-empty.
- Expected pre-fix baseline: an active exam with zero assignments is ABSENT from `completion_rate_by_exam`; S1 steps 4-7 and S4 steps 2-3 FAIL.

## Spec reference

FR-BB51 AC-2: `completion_rate_by_exam` includes every active exam in the tenant; `assigned_count` = distinct assigned users; `completed_count` = those with at least one `submitted` or `grading_pending` session; `passed_count` = those with at least one `passed=true` session. Endpoint `GET /api/v1/admin/dashboard` (reports:read: super_admin, department_admin, examiner).

## Preconditions and accounts

| Account | Role | Password | Source |
|---------|------|----------|--------|
| `admin@bilimbaga.local` | super_admin | `Admin1234!` (or current) | seeded |
| `uat.examiner@test.com` | examiner | `NewPass123!` | create per `route-guards-20261009.md` S0 |
| `uat.deptadmin@test.com` | department_admin | `NewPass123!` | same |
| `uat.employee@test.com` | employee | `NewPass123!` | earlier UAT |

- Platform at `http://localhost`; `{admin_token}` from login.
- Create exams via UI (`/admin/exams`) or API as admin (see `exam-configuration-20260609.md`; an exam needs at least one question to be activated, reuse questions from `question-authoring-20260609.md`). Confirm the exam status enum values from the exams model first and record them.
- Test data:

| Name | State | Assignments |
|------|-------|-------------|
| `UAT-Dash-Unassigned` | active | none |
| `UAT-Dash-Assigned` | active | `uat.employee@test.com`, no session started |
| `UAT-Dash-Draft` | draft (not activated) | none |

- Before creating anything, save the baseline `GET /admin/dashboard` (`baseline_list`, `baseline_active_count`).
- DB access (`psql`) optional, for S2 step 5.

## Scenario S1: API - unassigned active exam appears with zeros

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Record baseline dashboard JSON | Captured | |
| 2 | Admin | Create and activate `UAT-Dash-Unassigned` (no assignments) | Active | |
| 3 | Admin | Create, activate, assign `UAT-Dash-Assigned` to `uat.employee@test.com` | Active, 1 assignment | |
| 4 | Admin | `GET /api/v1/admin/dashboard` | HTTP 200; `completion_rate_by_exam` contains `UAT-Dash-Unassigned` | |
| 5 | Admin | Inspect that entry | `assigned_count=0`, `completed_count=0`, `passed_count=0`; any rate/percentage field is `0` (not null, NaN or error) | |
| 6 | Admin | Inspect `UAT-Dash-Assigned` | `assigned_count=1`, `completed_count=0`, `passed_count=0` | |
| 7 | Admin | Compare with baseline | List grew by exactly 2; earlier entries unchanged | |
| 8 | Admin | Check duplicates | Each exam id appears exactly once (LEFT JOIN must not duplicate rows) | |

## Scenario S2: Counting rules unchanged (regression)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Also assign `UAT-Dash-Assigned` to a second employee | `assigned_count=2` on next fetch | |
| 2 | Employee | Take and submit `UAT-Dash-Assigned` | Submitted | |
| 3 | Admin | Fetch dashboard | `completed_count=1`; `passed_count` 1 if passed else 0; `assigned_count` 2 | |
| 4 | Employee | Retake if allowed (second submitted session, same user) | `completed_count` stays 1 (distinct users) | |
| 5 | Tester | For a pre-existing exam with several sessions compare with DB `COUNT(DISTINCT user_id)` for assigned/completed/passed | Equal | |

## Scenario S3: Non-active exams excluded

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Create `UAT-Dash-Draft` (not activated) | Exists | |
| 2 | Admin | Fetch dashboard | `UAT-Dash-Draft` NOT listed | |
| 3 | Admin | Deactivate or archive `UAT-Dash-Unassigned` | Inactive | |
| 4 | Admin | Fetch dashboard | It disappears from the list | |
| 5 | Admin | Re-activate | Reappears with zeros | |

## Scenario S4: Dashboard UI

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Open `/admin/dashboard` | Renders without error | |
| 2 | Admin | Find the completion chart | `UAT-Dash-Unassigned` shown with 0 / 0%; no NaN width or overflow | |
| 3 | Admin | "Active exams" KPI | Equals list length = baseline + new active exams | |
| 4 | Admin | Total-assigned KPI | Sum of `assigned_count`; unassigned exam contributes 0 | |
| 5 | Admin | Tooltips/hover, if any | Show 0 correctly | |
| 6 | Admin | Edge: only unassigned active exam(s) exist (deactivate others or fresh tenant) | Chart section shown (list non-empty), no error; with no active exams, empty state rather than error | |
| 7 | Admin | Switch to ru and kk | Labels translated, no raw keys | |
| 8 | Examiner, dept admin | Open `/admin/dashboard` | Same data, no 403 | |
| 9 | Employee | `GET /admin/dashboard` with employee token | 403 | |

## Scenario S5: Export consistency (optional)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | `GET /api/v1/admin/dashboard/export` | 200 PDF; if it uses the same query, text lists `UAT-Dash-Unassigned` with 0; else record "not applicable" | |

## Cleanup

Delete or archive `UAT-Dash-*` exams and remove the extra assignment so later scenarios' counts are undisturbed.

## Pass / Fail criteria

- PASS: unassigned active exam present with zero counts, no duplicates, non-active exams excluded, counts equal to DB for existing exams, UI renders.
- FAIL (defect): unassigned active exam missing; duplicate rows; NULL/NaN; draft or archived exam listed; counts inflated by the join; 500.
- ENV ISSUE: #38 not merged; exams cannot be created or activated.

## Acceptance Criteria Coverage

| FR | AC# | Covered by |
|----|-----|------------|
| FR-BB51 | AC-2 (every active exam; count definitions) | S1, S2, S3 |
| FR-BB51 | reports:read access | S4 8-9 |
| Dashboard UI | rendering of the list | S4 |

## Out of Scope

Overdue-employee, recent-activity and average-score groups; query performance; PDF layout.

## Addendum (BA, after PRs #77/#83/#86 merged; see docs/requirements/conformance/PR77-PR83-PR86-conformance-20261009.md section 5)
- Dashboard PDF export is mandatory: expect 200, `application/pdf`, and the unassigned exam listed with zero counts (previously a swallowed 500).
- Also verify both CSV exports and date-range variants of the PDF; an employee token gets 403 on every export.
- After each export the api logs must contain no `reports: ... failed` line.
- AI insight smoke test (FR-BB74): the insight endpoint returns 200 without SQL errors.
- Live-check in `cert-public-verification-20261009.md`: a malformed code (`not-a-uuid`) must show "not found or invalid", not "temporarily unavailable" (known p2 gap from PR #86 until the handler validates the UUID).
