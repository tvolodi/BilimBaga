---
slug: employee-exam-taking
title: "Employee Exam Taking — UAT Scenario"
feature: employee-exam-taking (FR-BB34, FR-BB35, FR-BB37, FR-BB38, FR-BB39, FR-BB310, FR-BB311, FR-BB313, FR-BB314)
version: 1
created: 2026-06-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Preconditions

- The platform is running at `http://localhost:5173`.
- Employee: `uat.employee@test.com` / `NewPass123!`.
- Super Admin: `admin@test.com` / `Admin1234!`.
- "UAT Security Assessment" is Active, has 3 random-rule questions, 30-min time limit, 70% passing score, 2 max attempts, show_answers = "After completion", tab_switch = "Warn employee", certificate enabled.
- The exam is assigned to `uat.employee@test.com`.
- Preconditions from Exam Assignment UAT scenario have been completed.

---

## Scenario 1: Start Exam, Answer All Questions, Submit — Pass

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Log in as `uat.employee@test.com` / `NewPass123!` | Employee Portal is visible | |
| 2 | Employee | Assert "UAT Security Assessment" card shows status "Not started" and time limit "30 min" | Card is visible with correct info | |
| 3 | Employee | Click "Start exam" on the exam card | Confirmation modal appears showing time limit, passing score, and tab-switch policy | |
| 4 | Employee | Click "Confirm" (or "Start") in the modal | Exam-taking screen loads; first question is visible; timer shows approximately 30:00 | |
| 5 | Employee | Assert: exam title is shown in the top bar; progress counter shows "0 / 3 answered" (or similar) | Top bar and progress are correct | |
| 6 | Employee | Assert: question navigator panel is visible (sidebar or collapsible panel with question numbers) | Navigator is visible | |
| 7 | Employee | Select the first answer option on question 1 | Answer is highlighted; auto-save indicator shows "Saved" | |
| 8 | Employee | Navigate to question 2 using the question navigator | Question 2 is displayed; question 1 is marked as answered in the navigator | |
| 9 | Employee | Select an answer on question 2 | Answer is highlighted; auto-save indicator shows "Saved" | |
| 10 | Employee | Navigate to question 3 and select an answer | Answer is highlighted; auto-save indicator shows "Saved"; progress shows "3 / 3 answered" | |
| 11 | Employee | Click "Finish exam" | Review screen appears listing any unanswered or flagged questions | |
| 12 | Employee | Assert: no unanswered questions listed (all 3 are answered) | Review screen confirms all questions answered | |
| 13 | Employee | Click "Confirm and submit" | Session is submitted; employee is redirected to the Result screen | |
| 14 | Employee | Assert: Result screen shows a score percentage, and a Pass or Fail banner | Score and pass/fail banner are visible | |
| 15 | Employee | Assert: per-question review table is visible (since show_answers = "After completion") | Review table with question stems, employee answers, and correct answers is visible | |

---

## Scenario 2: Auto-Save Survives Browser Reload

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Start a second attempt of "UAT Security Assessment" (if attempts remain) or use a separate test exam | Exam-taking screen visible | |
| 2 | Employee | Answer question 1 | Answer saved; indicator shows "Saved" | |
| 3 | Employee | Close the browser tab (or navigate away) without submitting | Tab closed / navigation away | |
| 4 | Employee | Reopen `http://localhost:5173` and log in | Employee Portal visible | |
| 5 | Employee | Assert: "UAT Security Assessment" card shows status "In progress" with a "Continue" button | Card shows in-progress state | |
| 6 | Employee | Click "Continue" | Exam-taking screen reloads; question 1 still shows the previously selected answer; timer reflects remaining time (less than 30:00) | |

---

## Scenario 3: Tab Switch Warning

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Start the exam (new attempt or same in-progress session) | Exam-taking screen visible | |
| 2 | Employee | Switch to another browser tab (click a different tab) | A warning modal appears on return to the exam tab, informing the employee the tab switch was logged | |
| 3 | Employee | Dismiss the warning modal | Exam continues normally | |

---

## Scenario 4: Attempts Exhausted After Using Both Attempts

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Complete and submit both allowed attempts for "UAT Security Assessment" (or confirm both are consumed via admin) | Both attempts used | |
| 2 | Employee | Navigate to Employee Portal | "UAT Security Assessment" exam card shows "Attempts exhausted" (or similar) and the "Start" button is absent or disabled | |

---

## Acceptance Criteria Coverage

| AC# | Criterion | Covered by Scenario |
|-----|-----------|---------------------|
| 1 | Employee can start assigned exam and see all questions | Scenario 1, Steps 1–6 |
| 2 | Selecting an answer saves automatically; "Saved" indicator visible within 2 seconds | Scenario 1, Steps 7, 9, 10 |
| 3 | Closing and reopening browser restores answers and remaining time | Scenario 2, Steps 1–6 |
| 4 | Timer reaches zero → auto-submit → redirected to result | Not covered (requires waiting 30 min; tested in extended/timer-accelerated run) |
| 5 | Tab switch on auto-submit exam immediately ends session | Not covered (exam uses "Warn" not "Auto-submit"; tested with separate exam config) |
| 6 | After submitting objective-only exam, employee immediately sees score and pass/fail | Scenario 1, Steps 13–14 |
| 7 | Short-text exam shows "Grading in progress" | Not covered (exam has no short-text questions; tested in Manual Grading UAT) |
| 8 | Attempts exhausted: card shows message; Start button absent | Scenario 4, Step 2 |
