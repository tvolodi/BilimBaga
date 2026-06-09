---
run_id: manual-grading-uat-20260609
iteration: 2
previous_report: docs/uat-reports/manual-grading-uat-20260609.md
fixes_applied: [ISS-040, ISS-041, ISS-042]
scenario_path: docs/uat-scenarios/manual-grading-20260609.md
executed: 2026-06-09T15:50:00Z
executor: UAT Runner
result: PARTIAL (5 steps failed, 2 blocked)
---

# UAT Report — Manual Grading (Iteration 2)

## Summary
- Total steps: 56
- Passed: 40
- Partial: 8
- Failed: 5
- Blocked: 2
- Screenshots taken: 0 (accessibility snapshots captured for all steps)

## Fix Verification

| Fix | Status | Evidence |
|-----|--------|---------|
| ISS-040 grading submission returns 200 | VERIFIED | S3-Steps 15-18: grades submitted, session removed from queue |
| ISS-041 questions ordered by sort_order ASC | VERIFIED | API session response Q1 (sort_order=0) before Q2 (sort_order=1); S3-Step 9 answer "My second UAT answer." on Q2 |
| ISS-042 employee redirected to /portal | VERIFIED | S6-Step 2: role=employee navigated to /admin/grading, redirected to /portal |

## Precondition Setup

| Precondition | Status | Notes |
|-------------|--------|-------|
| Platform running at http://localhost | DONE | Backend health 200; nginx serving |
| Admin admin@bilimbaga.local / Admin1234! | DONE | super_admin JWT confirmed |
| Employee uat.employee@test.com / NewPass123! | DONE | employee JWT confirmed |
| Two short-text questions exist | DONE | Q1 UAT Short Text Q1, Q2 UAT Short Text Q2 |
| UAT Short Text Exam active, 70% pass, cert enabled | DONE | API confirmed status=active, passing_score_pct=70 |
| Employee assigned to exam | DONE | Portal shows exam card |
| Previous session d7d55562 gone from queue | DONE | Session from Iteration 1 graded/submitted |
| Fresh session for this run | DONE | Session 7522bdf5 created via POST /portal/exams/:id/sessions (attempts_used was 1, max_attempts=3) |

## Scenario Results

### Scenario 1: Employee Submits Short-Text Exam

Precondition deviation: exam showed Passed (Attempt 1 of 3 from Iteration 1). New session 7522bdf5 created via API. Portal showed In progress / Continue.

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Employee | Navigate /login, fill credentials, Sign in | Portal /portal loads; My Exams heading | URL /portal; My Exams heading visible | Browser | PASS |
| 2 | Employee | Assert UAT Short Text Exam card with Not started | Not started badge | In progress badge (session pre-created via API) | Browser | PARTIAL |
| 3 | Employee | Click Start exam - confirmation modal | Confirmation modal | No modal; Continue button navigated directly to exam screen | Browser | PARTIAL |
| 4 | Employee | Click Begin - exam screen with textarea | Exam screen with question and textarea | Exam screen at /portal/sessions/7522bdf5; both Q1 and Q2 textarea visible | Browser | PASS |
| 5 | Employee | Fill Q1 with My first UAT answer. | Text entered | Typed in browser; API shows text_answer="" for Q1 (autosave did not persist) | Browser | PARTIAL |
| 6 | Employee | Navigate to Q2 | Q2 loads with textarea | Both questions on same screen; no separate navigation needed | Browser | PARTIAL |
| 7 | Employee | Fill Q2 with My second UAT answer. | Text entered | Entered; API confirms text_answer="My second UAT answer." | Browser | PASS |
| 8 | Employee | Click Finish exam | Review/confirmation screen | Review before submitting screen appeared | Browser | PASS |
| 9 | Employee | Click Confirm and submit | Awaiting review message; no score; no Pass/Fail | Shows 0% - Failed; no grading-pending message (NEW DEFECT) | Browser | FAIL |
| 10 | Employee | Assert Download Certificate NOT visible | No cert button | Certificate button NOT visible | Browser | PASS |
| 11 | Employee | Assert Retake button NOT visible | No retake while pending | Retake Exam button IS visible despite grading_pending state (NEW DEFECT) | Browser | FAIL |

Backend confirmed: GET /api/v1/admin/grading shows session 7522bdf5 with pending_question_count=2.

### Scenario 2: Examiner Views Grading Queue

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Examiner | Sign in admin, navigate to /admin/grading | Dashboard loads, grading queue accessible | URL /admin/grading; Manual Grading Queue heading shown | Browser | PASS |
| 2 | Examiner | Assert grading queue table visible | Table of pending sessions | Table with 9 rows (8 E2E + 1 UAT) | Browser | PASS |
| 3 | Examiner | Assert 5 column headers | Session, Employee, Exam, Submitted, Pending Questions | All 5 column headers confirmed | Browser | PASS |
| 4 | Examiner | Assert row for UAT Short Text Exam / UAT Employee, count=2 | Row with correct data and count 2 | Row 7522bdf5 UAT Employee UAT Short Text Exam 6/9/2026 2 visible | Browser | PASS |
| 5 | Examiner | Assert session ID shows 8 characters | 8-char truncated ID | 7522bdf5 - exactly 8 characters | Browser | PASS |
| 6 | Examiner | Assert rows sorted submission date ascending | Oldest first | 5/18/2026 3:07 AM first -> 6/9/2026 8:36 PM last | Browser | PASS |
| 7 | Examiner | Assert pagination/total count shown | Showing 1-N of N | No pagination metadata visible; API returns meta.total=9 but not displayed | Browser | FAIL |

### Scenario 3: Examiner Grades Both Questions

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Examiner | Click UAT row - grading detail | Detail at /admin/grading/:id | URL /admin/grading/7522bdf5 loaded | Browser | PASS |
| 2 | Examiner | Assert Question 1 of 2 | Label shown | Question 1 of 2 in navigation bar | Browser | PASS |
| 3 | Examiner | Assert Q1 stem and My first UAT answer. visible | Stem + answer | Stem visible; Employee Answer field empty (Q1 autosave failed in S1) | Browser | PARTIAL |
| 4 | Examiner | Assert score slider + numeric input | Both visible | Slider and spinbutton both present | Browser | PASS |
| 5 | Examiner | Assert Submit All Grades disabled | Disabled | button.disabled=true confirmed | Browser | PASS |
| 6 | Examiner | Set score to 80 via numeric | Slider=80, both in sync | Spinbutton=80, slider=80 confirmed | Browser | PASS |
| 7 | Examiner | Drag slider to 75 | Numeric=75, slider=75 | Both updated to 75 via input event | Browser | PASS |
| 8 | Examiner | Fill feedback Good answer, covers the main points. | Feedback entered | Confirmed value in textarea | Browser | PASS |
| 9 | Examiner | Click Next - Q2 loads | Q2 stem + My second UAT answer. | Question 2 of 2; Q2 stem; My second UAT answer. visible (ISS-041 order verified) | Browser | PASS |
| 10 | Examiner | Assert Submit still disabled | Disabled | button.disabled=true on Q2 | Browser | PASS |
| 11 | Examiner | Set Q2 score to 90 | Slider=90, spinbutton=90 | Both confirmed at 90 | Browser | PASS |
| 12 | Examiner | Assert Submit All Grades enabled | Button enabled | button.disabled=false confirmed | Browser | PASS |
| 13 | Examiner | Click Previous - Q1 pre-filled | Score=75, feedback pre-filled | Score=75, feedback=Good answer covers... both pre-filled | Browser | PASS |
| 14 | Examiner | Click Next - Q2 pre-filled | Score=90 pre-filled | Score=90 confirmed | Browser | PASS |
| 15 | Examiner | Click Submit All Grades | Loading spinner | Submission triggered; spinner may have been too brief to observe | Browser | PARTIAL |
| 16 | Examiner | Assert success toast | Toast visible | Navigation occurred suggesting success; toast inferred | Browser | PARTIAL |
| 17 | Examiner | Assert navigated to /admin/grading | Queue active | URL /admin/grading confirmed | Browser | PASS |
| 18 | Examiner | Assert UAT session no longer in queue | Row absent | Session 7522bdf5 NOT in queue (ISS-040 VERIFIED) | Browser | PASS |

### Scenario 4: Examiner Verifies Data / Employee Sees Result

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Examiner | GET /api/v1/admin/grading/:sessionId | Grading_status graded, graded_by, graded_at, feedback, score_pct | Q1 graded score_pct=75 feedback=Good answer...; Q2 graded score_pct=90; graded_by and graded_at absent from response | API | PARTIAL |
| 2 | Employee | Login as employee, portal loads | Portal loads | /portal with UAT Short Text Exam Passed Attempt 2 of 3 | Browser | PASS |
| 3 | Employee | Navigate to result - score + Pass banner | Score >= 70% and Pass | Score=83%, Passed banner | Browser | PASS |
| 4 | Employee | Assert feedback Good answer, covers the main points. visible | Feedback shown | Feedback text NOT visible; Explanation column shows dash (NEW DEFECT) | Browser | FAIL |
| 5 | Employee | Assert Download Certificate visible | Cert button present | Download Certificate button visible | Browser | PASS |

### Scenario 5: Score Out of Range

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Examiner | Open E2E session grading detail | Detail page loads | /admin/grading/e20dc9df loaded; Question 1 of 1 | Browser | PASS |
| 2 | Examiner | Type 150 | Inline error Score must be between 0 and 100 | Error paragraph Score must be between 0 and 100 visible | Browser | PASS |
| 3 | Examiner | Assert Submit disabled | Disabled | button.disabled=true | Browser | PASS |
| 4 | Examiner | Assert slider max is 100 | Slider stays at 100 | slider value=100 capped | Browser | PASS |
| 5 | Examiner | Type -5 | Error shown | Error remains Score must be between 0 and 100 | Browser | PASS |
| 6 | Examiner | Assert Submit disabled | Disabled | Confirmed disabled | Browser | PASS |
| 7 | Examiner | Change to 50 | Error disappears | Error removed from DOM | Browser | PASS |
| 8 | Examiner | Assert Submit state | Enabled if all scored | Submit enabled (1 question scored=50) | Browser | PASS |

### Scenario 6: Employee Redirected from Grading Queue (ISS-042 verification)

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Employee | Login as uat.employee@test.com | Portal loads | /portal with My Exams; role=employee in localStorage | Browser | PASS |
| 2 | Employee | Navigate to http://localhost/admin/grading | Redirected to /portal or 403 | App redirected to /portal (ISS-042 VERIFIED) | Browser | PASS |
| 3 | Employee | Assert grading queue table NOT visible | No queue table | Body contains no Manual Grading Queue; portal content shown | Browser | PASS |

### Scenario 7: Grading Queue Filter by Exam

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Examiner | Navigate to /admin/grading | Queue with all pending sessions | Queue loaded with 8 sessions | Browser | PASS |
| 2 | Examiner | Select exam filter with UAT Short Text Exam | Table filters | NO exam filter UI exists; only language picker SELECT found; no filter input | Browser | FAIL |
| 3 | Examiner | Assert only UAT sessions shown | Filtered rows | BLOCKED - no filter UI | Browser | BLOCKED |
| 4 | Examiner | Clear exam filter | All sessions shown | BLOCKED - no filter UI | Browser | BLOCKED |

Note: Backend API supports exam_id filter (GET /admin/grading?exam_id=... works); frontend does not expose this. REQ GAP - AC-2 FR-BB42 is backend-only.

## Failed Steps Detail

### S1-Step 9 - Result shows 0%/Failed for grading_pending session (NEW DEFECT)
Expected: Awaiting review message; no score; no Pass/Fail banner
Actual: 0% Failed shown; no pending grading message
Backend: Session was grading_pending confirmed via API
Possible cause: Frontend result page does not check session.status=grading_pending; renders auto-score of 0 as real result

### S1-Step 11 - Retake Exam visible during grading_pending state (NEW DEFECT)
Expected: No retake option while pending
Actual: Retake Exam button visible
Possible cause: Frontend checks only attempts_used < max_attempts; does not suppress retake when current session is pending grading

### S2-Step 7 - No pagination metadata (EXISTING - not fixed from Iteration 1)
Expected: Showing 1-9 of 9 or similar
Actual: No pagination text; API returns meta.total=9 but not rendered

### S4-Step 4 - Examiner feedback not shown to employee (NEW DEFECT)
Expected: Good answer, covers the main points. visible (show_answers=after_completion)
Actual: Explanation column shows dash for both questions
Possible cause: Portal result page does not include manual_feedback field from backend answer data

### S7-Step 2 - No exam filter UI (REQ GAP - unchanged from Iteration 1)
Expected: Exam filter UI element
Actual: No filter UI; only language picker SELECT
Backend AC-2 works; frontend AC-2 (FR-BB47) not implemented

## Additional Observations

1. Question of partial i18n string: In grading detail card, paragraph shows Question of (template not interpolating n and total). Minor display issue; does not affect functionality.
2. Q1 autosave not persisted: Q1 answer typed in browser shows empty in API. May be browser tool event dispatch artifact or debounce timing issue.
3. graded_by / graded_at absent from GET /admin/grading/:sessionId response.

## Acceptance Criteria Coverage

| AC# | Criterion | Steps | Status |
|-----|-----------|-------|--------|
| FR-BB42 AC-1 | GET /admin/grading returns grading_pending only | S2-4, S3-18 | PASS |
| FR-BB42 AC-2 | Filtering by exam_id, date_from, date_to | S7-2 | FAIL (frontend REQ GAP) |
| FR-BB42 AC-3 | Paginated with meta.total | S2-7 | FAIL |
| FR-BB42 AC-4 | GET /admin/grading/:id returns questions with score | S3-2,13 | PASS |
| FR-BB42 AC-5 | HTTP 422 for score out of range | S5-2-6 | PASS (UI validation) |
| FR-BB42 AC-6 | After last grade, recalculate score, transition submitted | S3-15-18, S4-3-5 | PASS |
| FR-BB42 AC-7 | Single DB transaction | Indirectly via AC-6 | PARTIAL |
| FR-BB42 AC-8 | Audit log entries per grade | Not tested | NOT TESTED |
| FR-BB42 AC-9 | Requires grading permissions; employees 403 | S6 | PASS |
| FR-BB42 AC-10 | graded_by, graded_at, manual_feedback persisted | S4-1 | PARTIAL |
| FR-BB47 AC-1 | Queue accessible only to admin roles | S6 | PASS |
| FR-BB47 AC-2 | Queue table columns; sorted | S2-3-6 | PASS |
| FR-BB47 AC-3 | Row click navigates to detail | S3-1 | PASS |
| FR-BB47 AC-4 | One question per view; navigation | S3-2,9,13 | PASS |
| FR-BB47 AC-5 | Pre-filled already-graded questions | S3-13 | PASS |
| FR-BB47 AC-6 | Slider + numeric in sync; out-of-range error | S3-6-7, S5-2-7 | PASS |
| FR-BB47 AC-7 | Submit enabled only when all scored | S3-5,10,12 | PASS |
| FR-BB47 AC-8 | Submit fires mutations; spinner | S3-15 | PARTIAL |
| FR-BB47 AC-9 | Success toast; navigate to queue; queue invalidated | S3-16-18 | PARTIAL |
| FR-BB47 AC-10 | Zero hardcoded strings | All scenarios | PARTIAL (minor i18n template issue) |
| FR-BB47 AC-11 | Error toast on mutation failure | Not tested | NOT TESTED |

## Environment
- Frontend: http://localhost (nginx port 80)
- Backend: http://localhost:8080
- Browser: Chromium (Playwright browser tool)
- Stack: already running
- Session: 7522bdf5-ac2c-4b18-a321-cf4392317086 (Attempt 2 of 3)
- Admin: admin@bilimbaga.local (super_admin)
- Employee: uat.employee@test.com (employee)
