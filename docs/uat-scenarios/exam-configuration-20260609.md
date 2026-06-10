---
slug: exam-configuration
title: "Exam Configuration and Publishing — UAT Scenario"
feature: exam-configuration (FR-BB31, FR-BB32, FR-BB312, FR-BB315, FR-BB318)
version: 1
created: 2026-06-09
author: Business Analyst
---

## Preconditions

- The platform is running at `http://localhost:5173`.
- Super Admin account: `admin@test.com` / `Admin1234!`.
- At least 5 Active questions exist in category "UAT Security" with difficulty "Medium". If not, run the Question Authoring UAT scenario first and create/approve them.
- No exam named "UAT Security Assessment" exists yet.

---

## Scenario 1: Create, Configure, and Publish an Exam (Happy Path)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Log in as `admin@test.com` / `Admin1234!`, navigate to Exams | Exams list page is visible | |
| 2 | Examiner | Click "New Exam" (or "Create exam") | Exam creation form / step 1 of the wizard is visible | |
| 3 | Examiner | Fill "Title" with `UAT Security Assessment` | Title field populated | |
| 4 | Examiner | Fill "Time limit" with `30` (minutes) | Time limit set to 30 | |
| 5 | Examiner | Fill "Passing score" with `70` (%) | Passing score set to 70% | |
| 6 | Examiner | Fill "Max attempts" with `2` | Max attempts set to 2 | |
| 7 | Examiner | Set "Show answers" to "After completion" | Show answers policy set | |
| 8 | Examiner | Set "Tab switch behaviour" to "Warn employee" | Tab switch behaviour set | |
| 9 | Examiner | Enable "Certificate" toggle | Certificate is enabled | |
| 10 | Examiner | Click "Save" / "Next" to proceed to question rules step | Moves to step 2 (Question Rules); step 1 settings are saved | |
| 11 | Examiner | Click "Add rule", select mode "Random", select category "UAT Security", select difficulty "Medium", set count to `3` | A random rule row appears: "3 Medium UAT Security questions"; eligible question count indicator shows ≥ 3 | |
| 12 | Examiner | Click "Save" / "Next" to proceed to review step | Moves to the review/publish step | |
| 13 | Examiner | Assert: review summary shows title "UAT Security Assessment", time limit 30 min, passing score 70%, 2 attempts | Summary is visible and correct | |
| 14 | Examiner | Click "Publish" | Exam status changes to "Active"; success message is shown; exam appears in the exams list with status "Active" | |

---

## Scenario 2: Publish Fails When Rule Cannot Be Satisfied

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Create a new exam titled `UAT Unsatisfiable Exam`, any basic settings | Draft exam created | |
| 2 | Examiner | Add a random rule: category "UAT Security", difficulty "Hard", count `50` | Rule row appears; eligible question count indicator shows a warning (fewer than 50 hard UAT Security questions exist) | |
| 3 | Examiner | Attempt to publish | Error message is shown naming the unsatisfied rule; exam remains in "Draft" status | |

---

## Scenario 3: Active Exam Cannot Be Edited — Unpublish Required

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Navigate to Exams, open "UAT Security Assessment" (Active) | Exam detail page is visible | |
| 2 | Examiner | Assert: edit controls for title, time limit, etc. are disabled or absent; a message states editing requires unpublishing | Edit fields are read-only or hidden; "Unpublish" button is visible | |
| 3 | Examiner | Click "Unpublish" | Confirmation prompt appears | |
| 4 | Examiner | Confirm unpublish | Exam status returns to "Draft"; edit controls are now enabled | |
| 5 | Examiner | Change title to `UAT Security Assessment v2`, save | Title is updated to "UAT Security Assessment v2" | |
| 6 | Examiner | Re-publish the exam | Exam status returns to "Active" with the updated title | |

---

## Scenario 4: Archive an Exam

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Open the Active exam "UAT Security Assessment v2" | Exam detail page with status "Active" | |
| 2 | Examiner | Click "Archive" | Confirmation prompt appears | |
| 3 | Examiner | Confirm archive | Exam status changes to "Archived"; exam does not appear in the assignable exams list | |

---

## Acceptance Criteria Coverage

| AC# | Criterion | Covered by Scenario |
|-----|-----------|---------------------|
| 1 | Examiner creates draft exam with title, 30-min limit, 70% passing, 2 attempts | Scenario 1, Steps 1–9 |
| 2 | Random rule shows warning when fewer eligible questions than count requested | Scenario 2, Step 2 |
| 3 | Publish with valid rules changes status to Active | Scenario 1, Step 14 |
| 4 | Publish with unsatisfied rule shows specific error; exam stays Draft | Scenario 2, Step 3 |
| 5 | Active exam edit controls are disabled; unpublish is required | Scenario 3, Steps 1–4 |
| 6 | Unpublishing returns exam to Draft without affecting in-progress sessions | Scenario 3, Steps 3–4 |
| 7 | Archiving removes from assignable list; historical session data accessible | Scenario 4, Steps 1–3 |
