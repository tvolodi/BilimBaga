---
slug: analytics-and-reporting
title: "Analytics and Reporting — UAT Scenario"
feature: analytics-and-reporting (FR-BB51, FR-BB52, FR-BB53, FR-BB54, FR-BB55, FR-BB56, FR-BB57, FR-BB58, FR-BB59, FR-BB19)
version: 1
created: 2026-06-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Preconditions

- The platform is running at `http://localhost:5173`.
- Super Admin: `admin@test.com` / `Admin1234!`.
- At least one completed session exists for "UAT Security Assessment" (run Employee Exam Taking UAT first).
- At least one exam assignment with a past deadline exists (to populate the "Overdue employees" section).
- The employee `uat.employee@test.com` has at least one completed session.

---

## Scenario 1: Admin Dashboard Shows KPIs and Recent Activity

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Log in as `admin@test.com` / `Admin1234!` | Admin Dashboard is the landing page | |
| 2 | Super Admin | Assert: KPI cards are visible — Total employees, Active exams, Completion rate, Pass rate | All 4 KPI cards are present and show non-empty values | |
| 3 | Super Admin | Assert: a bar chart (or equivalent chart) showing completion rates per exam is visible | Chart is rendered with at least one bar | |
| 4 | Super Admin | Assert: "Recent activity" section shows at least one completed session entry | At least one row/entry is visible in recent activity | |
| 5 | Super Admin | Assert: recent activity entry shows employee name, exam name, score, and pass/fail | Row data is populated correctly | |

---

## Scenario 2: Per-Exam Analytics Page

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Navigate to Exams, open "UAT Security Assessment" | Exam detail page visible | |
| 2 | Super Admin | Click the "Analytics" tab (or navigate to the analytics page for this exam) | Per-exam analytics page is visible | |
| 3 | Super Admin | Assert: score distribution histogram or chart is visible | Chart is rendered | |
| 4 | Super Admin | Assert: summary statistics are shown — pass rate, average score, total attempts, unique participants | All statistics are visible (may show low values given test data) | |
| 5 | Super Admin | Assert: per-question statistics table is visible with at least one row showing question stem preview, correct rate, and average time | Table is present and populated | |
| 6 | Super Admin | Click "Export" (CSV or export button) | Browser initiates a file download | |
| 7 | Super Admin | Assert: downloaded file is a CSV and contains rows with employee name, score, and pass/fail columns | File contents are valid CSV with expected columns | |

---

## Scenario 3: Employee Record View and Export

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Navigate to Users, find "UAT Employee", click "View record" | Employee Record page is visible | |
| 2 | Super Admin | Assert: session history table shows at least one row with exam name, date, score, and pass/fail badge | Table row is present and populated | |
| 3 | Super Admin | Click "Export" (CSV for this employee's record) | Browser initiates CSV download | |
| 4 | Super Admin | Assert: downloaded CSV contains at least one row with the employee's session data | CSV has expected content | |

---

## Scenario 4: Audit Log — Filter and Export

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Navigate to Settings → Audit Log (or Audit in sidebar) | Audit log page is visible with a paginated table of events | |
| 2 | Super Admin | Assert: table shows columns for actor, action, entity, IP, and date | Columns are visible | |
| 3 | Super Admin | Apply filter: Actor name = `admin` | Rows filtered to show only events by admin actors | |
| 4 | Super Admin | Assert: at least one row is shown (login events, user creation, etc.) | At least one row visible | |
| 5 | Super Admin | Click on a row to expand it | Full metadata JSON is visible in an expanded section | |
| 6 | Super Admin | Click "Export to CSV" | Browser initiates CSV download | |
| 7 | Super Admin | Assert: downloaded CSV contains the same rows as the filtered view (actor = admin only) | CSV rows match the filtered view | |

---

## Scenario 5: Department Admin Cannot See Other Departments' Data

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Create a Department Admin account `dept.admin@test.com` / `DeptAdmin1!` in department "UAT Engineering" if it doesn't exist | Account created | |
| 2 | Dept Admin | Log in as `dept.admin@test.com` / `DeptAdmin1!` | Admin shell visible | |
| 3 | Dept Admin | Navigate to Users | Only users in "UAT Engineering" are visible; users in other departments are not shown | |
| 4 | Dept Admin | Navigate to Dashboard (if accessible) | KPIs and data are scoped to UAT Engineering employees only; employees from other departments are not shown | |

---

## Acceptance Criteria Coverage

| AC# | Criterion | Covered by Scenario |
|-----|-----------|---------------------|
| 1 | Dashboard shows completion rate and pass rate for last 30 days | Scenario 1, Steps 2–3 |
| 2 | Per-exam analytics shows score distribution histogram and highlights low-correct-rate questions | Scenario 2, Steps 3–5 |
| 3 | Dashboard "overdue employees" table lists employees past deadline | Scenario 1, Step 2 (overdue section) |
| 4 | Admin can download CSV of all sessions for a specific exam | Scenario 2, Steps 6–7 |
| 5 | Audit log shows all login events filtered by user name | Scenario 4, Steps 3–4 |
| 6 | Audit log CSV export contains same rows as filtered view | Scenario 4, Steps 6–7 |
| 7 | Department Admin cannot see other departments' data | Scenario 5, Steps 3–4 |
| 8 | AI insight summary appears after clicking "Generate AI insights" | Not covered (Phase 7 feature) |
