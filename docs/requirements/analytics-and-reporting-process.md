---
slug: analytics-and-reporting
title: "Analytics and Reporting"
type: process-description
status: draft
created: 2026-06-09
related_requirements: [FR-BB51, FR-BB52, FR-BB53, FR-BB54, FR-BB55, FR-BB56, FR-BB57, FR-BB58, FR-BB59, FR-BB19, FR-BB74, FR-BB75]
---

## Business Goal

The organisation needs data-driven insight into the effectiveness of its assessment programme: which exams are too hard or too easy, which employees are at risk of non-compliance, what overall competency levels look like across departments and tracks, and a complete audit trail of all administrative actions. This process covers how admins access, interpret, and export data from the platform. Success means decision-makers can identify compliance gaps, improve question quality, and provide evidence of training completion — all without needing database access.

## Actors

| Actor | Role |
|-------|------|
| Super Admin | Access to all analytics, all exports, and the full audit log |
| Department Admin | Access to analytics and exports scoped to their department |
| Examiner | Access to per-exam analytics for exams they manage |
| AI Assistant | Generates natural-language insight summaries on admin request (Phase 7) |

## Process Steps

### Step 1 — Monitor Overall Programme Health (Super Admin)

1. Admin logs in and is directed to the Admin Dashboard.
2. The dashboard displays KPI cards:
   - Total active employees.
   - Exams currently active.
   - Completion rate for the last 30 days.
   - Pass rate for the last 30 days.
3. A bar chart shows completion rates per active exam.
4. An "Overdue employees" table lists employees who have passed their assignment deadline without completing the exam (up to 20 most urgent).
5. A "Recent activity" feed shows the last 20 completed sessions with employee name, exam, score, and pass/fail.
6. Average score by track (security, safety, loyalty/values) over the last 90 days.
7. **Expected outcome**: Admin can identify programme health and overdue compliance risks at a glance.

### Step 2 — Analyse a Specific Exam (Examiner / Admin)

1. Admin navigates to Exams → selects an exam → Analytics tab.
2. The per-exam analytics page shows:
   - Score distribution histogram (10% buckets from 0–100).
   - Pass rate, average score, median score, total attempts, unique participants.
   - Per-question statistics table: question stem preview, correct rate (%), average time spent, answer distribution.
   - Questions with correct rate < 40% are highlighted as "Consider revising".
   - Questions with correct rate > 95% are highlighted as "Consider removing (too easy)".
3. Admin uses this to identify poorly performing questions and decide whether to revise or remove them.
4. **Expected outcome**: Admin can make evidence-based decisions about question quality.

### Step 3 — Request AI Insight Summary (Admin, Phase 7)

1. On the per-exam analytics page, admin clicks "Generate AI insights".
2. The system sends anonymised aggregate stats to the AI service.
3. The AI returns 3–5 natural-language bullet observations (e.g. "Question 7 has a 23% correct rate; the stem may be ambiguous").
4. Insights are displayed on the page and cached for 24 hours.
5. Admin can click "Regenerate" to force a fresh analysis.
6. **Expected outcome**: Admin receives plain-language recommendations without needing to interpret raw statistics.

### Step 4 — Review an Employee's Record (Admin)

1. Admin navigates to Users → selects an employee → "View record".
2. The Employee Record page shows:
   - Session history table: exam name, date, score, pass/fail badge, certificate link, time taken.
   - Per-track progress summary: security / safety / loyalty completion status, questions answered, last activity.
3. Admin can download a CSV of all sessions for this employee.
4. **Expected outcome**: Admin can confirm an individual employee's compliance and download evidence for HR.

### Step 5 — Generate Loyalty Profile Narrative (Admin, Phase 7)

1. For loyalty/values assessment exams, admin opens a session result on the Employee Record page.
2. Admin clicks "Generate AI loyalty narrative".
3. The system sends the employee's Likert responses to the AI.
4. The AI returns a paragraph-length narrative of the employee's values profile.
5. The narrative is labelled "AI-generated summary" and is visible only to department admin and above.
6. **Expected outcome**: Department admins gain qualitative insight into an employee's values alignment alongside the numeric score.

### Step 6 — Export Data (Admin)

Available exports:

- **Per-exam results CSV**: all sessions for one exam — employee name, department, date, score, passed, time taken, per-question scores.
- **Per-employee record CSV**: all sessions for one employee.
- **Dashboard PDF**: company logo, date range, completion rates table, pass rates table, top/bottom questions.

Admin navigates to the relevant page (exam analytics, employee record, or dashboard) and clicks the "Export" button.

The browser downloads the file immediately.

7. **Expected outcome**: Admin can include data in external reports, share with HR, or archive for compliance audits.

### Step 7 — Review the Audit Log (Super Admin)

1. Super Admin navigates to Settings → Audit Log.
2. The audit log shows a paginated table of all administrative actions: who did what, to which entity, from which IP, and when.
3. Super Admin can filter by:
   - Date range.
   - Actor (search by name).
   - Action type (multi-select: login, user created, exam published, session graded, etc.).
   - Entity type.
4. Clicking a row expands the full metadata JSON.
5. Super Admin can export the filtered log to CSV.
6. **Expected outcome**: Super Admin has a tamper-evident trail for security, compliance, and incident investigation.

## Business Rules

- Analytics data is read-only; admins cannot edit, delete, or suppress session records.
- Department Admins see analytics only for employees and exams within their department scope.
- Audit log entries are append-only; no admin (including Super Admin) can delete or edit them.
- The audit log CSV export applies the same filters currently active on the screen.
- AI insight summaries are cached for 24 hours to avoid excessive API cost; the cache is invalidated when "Regenerate" is clicked.
- Loyalty profile narratives are visible only to Department Admin and above; employees cannot see the AI narrative about themselves.
- All exports (CSV and PDF) reflect the data as of the moment the download is triggered; they are not live documents.
- The dashboard's "overdue employees" list is capped at 20 entries; the export is uncapped.

## Acceptance Criteria (business language)

1. The admin dashboard shows completion rate and pass rate for the last 30 days after logging in.
2. The per-exam analytics page shows the score distribution histogram and highlights questions with a correct rate below 40%.
3. The dashboard "overdue employees" table lists employees who have passed their deadline without completing the exam.
4. An admin can download a CSV of all sessions for a specific exam from the analytics page.
5. The audit log shows all login events for a specific user when filtered by that user's name.
6. The audit log CSV export contains the same rows as the filtered view on screen.
7. A Department Admin cannot see sessions or analytics for employees in other departments.
8. The AI insight summary appears on the per-exam analytics page after clicking "Generate AI insights" (Phase 7).

## Out of Scope

- Real-time push notifications for analytics events (dashboard is refreshed on page load).
- Custom report builder or ad-hoc query interface.
- Integration with external BI tools (Tableau, Power BI).
- Compliance regulation mapping (the platform does not tag sessions to specific legal requirements).
