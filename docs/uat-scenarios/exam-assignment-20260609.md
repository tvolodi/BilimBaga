---
slug: exam-assignment
title: "Exam Assignment — UAT Scenario"
feature: exam-assignment (FR-BB33, FR-BB34, FR-BB61)
version: 1
created: 2026-06-09
author: Business Analyst
---

## Preconditions

- The platform is running at `http://localhost`.
- Super Admin: `admin@test.com` / `Admin1234!`.
- An Active exam named "UAT Security Assessment" exists (run Exam Configuration UAT first if needed).
- An Employee account exists: `uat.employee@test.com` / `NewPass123!`, in department "UAT Engineering".
- The employee is NOT yet assigned to "UAT Security Assessment".

---

## Scenario 1: Assign Exam to Individual Employee — Employee Sees It in Portal

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Log in as `admin@test.com` / `Admin1234!`, navigate to Exams | Exams list visible | |
| 2 | Super Admin | Open "UAT Security Assessment" | Exam detail page visible with status "Active" | |
| 3 | Super Admin | Click "Assign" (or navigate to Assignments tab) | Assignment form / modal appears | |
| 4 | Super Admin | Select scope "Individual", search for and select `uat.employee@test.com` | Employee is selected | |
| 5 | Super Admin | Leave deadline blank, confirm assignment | Success message shown; assignment row appears in the assignments list | |
| 6 | Employee | Log in as `uat.employee@test.com` / `NewPass123!` | Employee Portal loads | |
| 7 | Employee | Assert: "UAT Security Assessment" exam card is visible | Exam card is visible with status "Not started", no deadline countdown | |
| 8 | Employee | Assert: card shows time limit (30 min) and max attempts (2) | Time limit and attempt info are visible on the card | |

---

## Scenario 2: Assignment with Deadline — Deadline Shown on Card

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Navigate to Exams → "UAT Security Assessment" → Assignments | Assignments list visible | |
| 2 | Super Admin | Click "Assign", select "Individual", select `uat.employee@test.com`, set deadline to tomorrow's date at 23:59, confirm | Assignment updated with deadline | |
| 3 | Employee | Reload the Employee Portal | Exam card for "UAT Security Assessment" shows a deadline countdown (e.g. "Due in 23h 59m") | |

---

## Scenario 3: Remove Assignment — Exam Disappears from Portal

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Navigate to Exams → "UAT Security Assessment" → Assignments | Assignment for `uat.employee@test.com` is listed | |
| 2 | Super Admin | Click "Remove" next to the assignment for `uat.employee@test.com` | Confirmation prompt appears | |
| 3 | Super Admin | Confirm removal | Assignment row disappears from the list | |
| 4 | Employee | Reload the Employee Portal | "UAT Security Assessment" exam card is no longer visible | |

---

## Scenario 4: Assignment Completion Status Visible to Admin

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super Admin | Re-assign "UAT Security Assessment" to `uat.employee@test.com` (no deadline) | Assignment created | |
| 2 | Super Admin | Navigate to Assignments for "UAT Security Assessment" | Completion table shows `uat.employee@test.com` with status "Not started" | |
| 3 | Employee | Navigate to the Employee Portal; assert the exam card is visible | Card is visible | |

---

## Acceptance Criteria Coverage

| AC# | Criterion | Covered by Scenario |
|-----|-----------|---------------------|
| 1 | Assigning to department makes exam card visible to all members | Not covered (requires multiple employees in department; tested in extended run) |
| 2 | Assignment with deadline shows countdown on exam card | Scenario 2, Step 3 |
| 3 | After deadline passes, card shows "Expired" and Start is disabled | Not covered (requires waiting for deadline; tested in extended run) |
| 4 | Completion table shows correct status per assignee | Scenario 4, Steps 1–3 |
| 5 | Removing assignment removes card; completed sessions remain in history | Scenario 3, Steps 1–4 |
| 6 | Attempts exhausted: card shows "Attempts exhausted" | Covered indirectly by Employee Exam Taking UAT (after 2 attempts) |
