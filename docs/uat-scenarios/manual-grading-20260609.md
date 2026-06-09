---
slug: manual-grading
title: "Manual Grading — UAT Scenario"
feature: manual-grading (FR-BB42, FR-BB47)
version: 2
created: 2026-06-09
author: Business Analyst
---

## Preconditions

- The platform is running at `http://localhost` (nginx port 80 → api:8080).
- Super Admin / Examiner: `admin@test.com` / `Admin1234!` (role: super_admin; holds grading:read and grading:write permissions).
- Employee: `uat.employee@test.com` / `NewPass123!` (role: employee).
- **Two active short-text questions** exist in the question bank (category "UAT Short Text" or any active category). If they do not exist:
  1. Via API: `POST /api/v1/admin/questions` × 2 with `type: "short_text"`, any category/difficulty, then transition each `draft → review → active`.
- **"UAT Short Text Exam"** exists, is Active, contains those 2 short-text questions via manual question rules, 70% passing score, `show_answers = "after_completion"`, `certificate_enabled = true`, no time limit or long time limit (≥60 min). If it does not exist:
  1. Create via `POST /api/v1/admin/exams` with the above settings; add manual rules; publish.
- The exam is **assigned** to `uat.employee@test.com`. If not assigned: `POST /api/v1/admin/exams/:id/assignments`.
- `uat.employee@test.com` has **NOT** submitted a session for "UAT Short Text Exam" for this run. (If a prior session exists and attempts are exhausted, create a new test exam or reset attempts.)

---

## Scenario 1: Employee Submits Short-Text Exam — Session Enters Grading Queue

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Navigate to `http://localhost/login`; fill email `uat.employee@test.com`, password `NewPass123!`; click Login | Employee Portal (`/portal`) loads; "My Exams" heading visible | |
| 2 | Employee | Assert "UAT Short Text Exam" card is visible with status "Not started" | Card visible with "Not started" badge | |
| 3 | Employee | Click "Start exam" on the "UAT Short Text Exam" card | Confirmation modal appears with exam details | |
| 4 | Employee | Click "Begin" (or "Start") in the confirmation modal | Exam-taking screen loads; first short-text question visible with a textarea input | |
| 5 | Employee | Fill the textarea for question 1 with `My first UAT answer.` | Text is entered in the textarea | |
| 6 | Employee | Navigate to question 2 (via Next button or question navigator) | Question 2 loads with its own textarea | |
| 7 | Employee | Fill the textarea for question 2 with `My second UAT answer.` | Text is entered | |
| 8 | Employee | Click "Finish exam" | Review/confirmation screen appears | |
| 9 | Employee | Click "Confirm and submit" (or equivalent submit confirmation) | Result screen loads; message indicates grading is in progress (e.g. "Awaiting review" or "Grading in progress"); **no score percentage** and **no Pass/Fail banner** are shown | |
| 10 | Employee | Assert: "Download Certificate" button is NOT visible | Certificate button absent | |
| 11 | Employee | Assert: "Retake" / "Start new attempt" button is NOT visible | No retake option shown while grading is pending | |

---

## Scenario 2: Examiner Views Grading Queue — Correct Columns and Data

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Navigate to `http://localhost/login`; fill email `admin@test.com`, password `Admin1234!`; click Login | Admin dashboard loads | |
| 2 | Examiner | Navigate to `http://localhost/admin/grading` | Grading queue page loads; a table of pending sessions is visible | |
| 3 | Examiner | Assert: table has columns for session ID (truncated), employee name, exam name, submission date, and pending question count | All five column headers are visible | |
| 4 | Examiner | Assert: a row for "UAT Short Text Exam" / `uat.employee@test.com` is present with pending question count "2" | Row visible with correct employee name, exam name, and count | |
| 5 | Examiner | Assert: the session ID in the row shows only 8 characters (not the full UUID) | Truncated 8-character session ID displayed | |
| 6 | Examiner | Assert: rows are sorted by submission date ascending (oldest first — only one row visible, or earliest row is first) | Oldest submission appears at top of table | |
| 7 | Examiner | Assert: pagination or total count information is shown (e.g. "Showing 1–1 of 1") | Pagination metadata visible | |

---

## Scenario 3: Examiner Grades Both Questions — Happy Path (Including Submit Button State and Pre-Fill)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Click the queue row for the `uat.employee@test.com` "UAT Short Text Exam" session | Grading detail page loads at `/admin/grading/:sessionId` | |
| 2 | Examiner | Assert: "Question 1 of 2" indicator is visible | "Question 1 of 2" label shown | |
| 3 | Examiner | Assert: the question stem and the employee's answer text "My first UAT answer." are visible | Question stem and answer text displayed | |
| 4 | Examiner | Assert: a score slider (range 0–100) and a numeric input field are both present | Slider and number field visible side-by-side | |
| 5 | Examiner | Assert: "Submit All Grades" button is **disabled** | Button disabled (no scores entered yet) | |
| 6 | Examiner | Set score to `80` via the numeric input field | Slider repositions to 80; numeric field shows `80`; both remain in sync | |
| 7 | Examiner | Drag the slider to `75` | Numeric field updates to `75`; slider is at 75; both in sync | |
| 8 | Examiner | Fill the feedback field with `Good answer, covers the main points.` | Feedback text entered | |
| 9 | Examiner | Click "Next" button | Question 2 of 2 loads; question 2 stem and "My second UAT answer." visible | |
| 10 | Examiner | Assert: "Submit All Grades" button is still **disabled** (question 2 not yet scored) | Button remains disabled | |
| 11 | Examiner | Set score to `90` for question 2 | Slider at 90; numeric field shows `90` | |
| 12 | Examiner | Assert: "Submit All Grades" button is now **enabled** | Button becomes enabled | |
| 13 | Examiner | Click "Previous" button | Question 1 reloads; score field shows `75` and feedback field shows `Good answer, covers the main points.` (pre-filled) | |
| 14 | Examiner | Click "Next" to return to question 2 | Question 2 reloads with score `90` pre-filled | |
| 15 | Examiner | Click "Submit All Grades" | Loading spinner is shown during submission | |
| 16 | Examiner | Assert: after submission completes, a success toast notification appears | Success toast visible | |
| 17 | Examiner | Assert: examiner is navigated back to `/admin/grading` queue | Queue page is active | |
| 18 | Examiner | Assert: the "UAT Short Text Exam" session for `uat.employee@test.com` is **no longer listed** in the queue | Session row absent from queue | |

---

## Scenario 4: Examiner Verifies Graded Data Persisted — Employee Sees Final Result

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Call `GET /api/v1/admin/grading/:sessionId` via API (using the session ID from Scenario 3) | Response includes both questions with `grading_status != "pending_manual"`, `graded_by` (examiner UUID), `graded_at` (UTC timestamp), `manual_feedback` set; `score_pct` values match submitted scores | |
| 2 | Employee | Navigate to `http://localhost/login`; log in as `uat.employee@test.com` / `NewPass123!` | Portal loads | |
| 3 | Employee | Navigate to the result for "UAT Short Text Exam" (e.g. click "View result" on the exam card, or visit `/portal/sessions/:sessionId/result`) | Result screen shows a score percentage (≥70%) and a **Pass** banner | |
| 4 | Employee | Assert: feedback text `Good answer, covers the main points.` is visible on the result screen (since show_answers = "after_completion") | Feedback for question 1 visible | |
| 5 | Employee | Assert: "Download Certificate" button is visible | Certificate button present | |

---

## Scenario 5: Score Out of Range — Inline Error and Submit Blocked

> **Setup**: Navigate to any session currently in the grading queue (create a second short-text session via API if needed, or reuse an existing one before it is fully graded).

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Open a grading detail page for any pending session | Grading detail page loads; "Question 1 of …" visible | |
| 2 | Examiner | Clear the score numeric input and type `150` | An inline error message is shown (e.g. "Score must be between 0 and 100") | |
| 3 | Examiner | Assert: "Submit All Grades" button is **disabled** | Button is disabled while invalid score is present | |
| 4 | Examiner | Assert: slider does not advance past 100 | Slider capped at 100 regardless of typed value | |
| 5 | Examiner | Clear the score field and type `-5` | Inline error message shown for negative value | |
| 6 | Examiner | Assert: "Submit All Grades" button remains **disabled** | Button still disabled | |
| 7 | Examiner | Change the score to `50` | Inline error disappears | |
| 8 | Examiner | Assert: "Submit All Grades" button state reflects whether all other questions also have scores | Button enabled if all scored, disabled otherwise | |

---

## Scenario 6: Employee Redirected from Grading Queue (Access Control)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Log in as `uat.employee@test.com` / `NewPass123!` | Portal (`/portal`) loads | |
| 2 | Employee | Navigate to `http://localhost/admin/grading` | Employee is redirected away from the grading queue (e.g. back to `/portal` or shown a 403 / "Access denied" message) | |
| 3 | Employee | Assert: grading queue table is **not visible** | No queue table shown | |

---

## Scenario 7: Grading Queue Filter by Exam

> **Setup**: Queue must have at least one `grading_pending` session. Use the session from Scenario 1 if available, or create a new one.

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Examiner | Navigate to `http://localhost/admin/grading` | Grading queue with all pending sessions visible | |
| 2 | Examiner | Select or fill the Exam filter with "UAT Short Text Exam" | Table updates; only sessions for "UAT Short Text Exam" are shown | |
| 3 | Examiner | Assert: sessions for other exams (if any) are **not shown** | Only filtered exam rows visible | |
| 4 | Examiner | Clear the exam filter | All pending sessions are shown again | |

---

## Acceptance Criteria Coverage

### FR-BB42 — Manual Grading Queue (Backend)

| AC# | Criterion (from FR-BB42) | Covered by Scenario |
|-----|--------------------------|---------------------|
| AC-1 | `GET /admin/grading` returns only sessions with `status = grading_pending` | Scenario 2, Step 4 (only newly-submitted session appears; already-submitted session from Scenario 3 Step 18 disappears) |
| AC-2 | Supports filtering by `exam_id`, `date_from`, `date_to` | Scenario 7 (exam filter applied and cleared) |
| AC-3 | Paginated with `page`/`per_page`; returns `meta.total` | Scenario 2, Step 7 (total count visible in pagination metadata) |
| AC-4 | `GET /admin/grading/:sessionId` returns only `short_text` questions; already-graded included with current score | Scenario 3, Steps 2–3 and Step 13 (pre-filled scores) |
| AC-5 | HTTP 422 / `INVALID_SCORE` for `score_pct` outside [0, 100] | Scenario 5, Steps 2–6 (UI validates and blocks submit; backend would return 422 for out-of-range values) |
| AC-6 | After last grade, system recalculates score and transitions session to `submitted` | Scenario 3, Steps 15–18 and Scenario 4, Steps 3–5 (session leaves queue; employee sees final score) |
| AC-7 | Final score recalculation in single DB transaction | Verified indirectly: AC-6 behavior (score + status both updated atomically); no partial states observed |
| AC-8 | Each grade writes audit log entry (`action = answer.grade`) | Scenario 4, Step 1 (API response confirms `graded_by`, `graded_at` set; audit log entries expected per grade) |
| AC-9 | Requires `grading:read`/`grading:write`; employees get 403 | Scenario 6 (employee redirected from `/admin/grading`) |
| AC-10 | `graded_by`, `graded_at`, `manual_feedback` persisted | Scenario 4, Step 1 (API response verified) |

### FR-BB47 — Frontend Manual Grading UI

| AC# | Criterion (from FR-BB47) | Covered by Scenario |
|-----|--------------------------|---------------------|
| AC-1 | Queue accessible only to examiner/dept_admin/super_admin; employees redirected | Scenario 6 (employee navigates to queue, is redirected) |
| AC-2 | Queue table columns: session ID (8 chars), employee name, exam name, submission date, pending count; sorted submission date asc | Scenario 2, Steps 3–6 |
| AC-3 | Clicking queue row navigates to grading detail page | Scenario 3, Step 1 |
| AC-4 | Detail page shows one question at a time with "Question X of Y" indicator; Previous/Next navigation | Scenario 3, Steps 2, 9, 13 |
| AC-5 | Already-graded questions show current score and feedback as pre-filled | Scenario 3, Step 13 (navigate back to Q1 after grading; score and feedback pre-filled) |
| AC-6 | Score input: slider + numeric field in sync; value outside 0–100 disables Submit and shows inline error | Scenario 3, Steps 6–7 (sync); Scenario 5, Steps 2–7 (out-of-range error + submit disabled) |
| AC-7 | "Submit All Grades" enabled only when every question has a score | Scenario 3, Steps 5, 10, 12 |
| AC-8 | Clicking "Submit All Grades" fires POST mutations sequentially; loading spinner shown | Scenario 3, Steps 15 |
| AC-9 | On success: success toast shown; examiner navigated back to queue; queue invalidated and refetched | Scenario 3, Steps 16–18 |
| AC-10 | Zero hardcoded strings; all text via i18n | Observed throughout all scenarios: all button labels, column headers, toast messages, and error messages must render (non-empty) in the current locale without raw key strings visible (e.g. no `grading.submit_all_grades` shown literally) |
| AC-11 | If any grade mutation fails: error toast shown; remaining mutations not fired; Submit re-enabled | Scenario 5, Step 8 (partial: Submit re-enabled after error entry); full network-fail path requires API mock and is noted as out-of-scope for this run — flag as PARTIAL |
