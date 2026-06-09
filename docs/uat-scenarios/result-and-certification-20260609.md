---
slug: result-and-certification
title: "Result Review and Certification — UAT Scenario"
feature: result-and-certification (FR-BB41, FR-BB43, FR-BB44, FR-BB45, FR-BB46)
version: 2
created: 2026-06-09
author: Business Analyst
---

## Preconditions

1. Platform is running at `http://localhost` (nginx port 80; API at `http://localhost/api/v1`).
2. **Super Admin**: `admin@bilimbaga.local` / `Admin1234!`.
3. **Employee**: `uat.employee@test.com` / `NewPass123!`.
4. A second **no-exam employee** `uat.noexam@test.com` / `NewPass123!` exists with no submitted sessions. If absent, create via admin in Scenario S0d.
5. **"UAT Result Exam"** (created in Scenario S0a) — `certificate_enabled=true`, `show_answers='after_completion'`, `passing_score=70`, `time_limit=30 min`, `max_attempts=3`, flat 5 single-choice questions with one correct answer each, no sections, assigned to `uat.employee@test.com`.
6. **"UAT No-Answers Exam"** (created in Scenario S0b) — `certificate_enabled=false`, `show_answers='never'`, `passing_score=70`, `max_attempts=2`, 5 single-choice questions, assigned to `uat.employee@test.com`.
7. **"UAT Pending Review Exam"** (created in Scenario S0c) — at least 1 short-text question, `certificate_enabled=false`, `show_answers='after_completion'`, `max_attempts=2`, assigned to `uat.employee@test.com`.
8. `uat.employee@test.com` has **at least one submitted passing session** on "UAT Result Exam". Execute Scenario S3 if not present.
9. Variable **`{passing_session_id}`** is recorded after Scenario S3 (from result page URL).
10. Variable **`{cert_code}`** is recorded after Scenario S2 (from Content-Disposition filename of the downloaded certificate).
11. Variable **`{employee_token}`** = JWT access token for `uat.employee@test.com` (obtained during login).
12. Variable **`{admin_token}`** = JWT access token for `admin@bilimbaga.local` (obtained during admin login).
13. Variable **`{inprogress_session_id}`** is recorded during Scenario S10b when an in-progress session is created.
14. Variable **`{noAnswers_session_id}`** is recorded after completing Scenario S5 (submitted session on "UAT No-Answers Exam").

---

## Scenario 1: View Result Screen After Passing — Download Certificate

---

## Scenario S0: Admin Setup — Create Test Exams

> Run once before all other scenarios. Skip individual sub-scenarios if the exam already exists and is Active.

### S0a — Create and Publish "UAT Result Exam"

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Navigate to `http://localhost/login`; fill email `admin@bilimbaga.local`, password `Admin1234!`; click Login | Admin dashboard visible at `/admin/dashboard` | |
| 2 | Admin | Navigate to `http://localhost/admin/exams`; click "New Exam" | Exam Wizard Step 1 visible | |
| 3 | Admin | Fill Title: `UAT Result Exam`; set Passing Score `70`; Time Limit `30`; Max Attempts `3` | Fields populated | |
| 4 | Admin | Set Certificate toggle to **enabled** | Certificate toggle is on | |
| 5 | Admin | Set Show Answers to `After completion` | Show Answers dropdown shows "After completion" | |
| 6 | Admin | Proceed through wizard; add 5 single-choice questions (each with exactly one correct answer marked); save | 5 questions listed; no section grouping | |
| 7 | Admin | Save exam and click "Publish"; confirm | Exam "UAT Result Exam" status shows "Active" | |
| 8 | Admin | Navigate to assignments for "UAT Result Exam"; assign `uat.employee@test.com` | Assignment row visible for uat.employee | |

### S0b — Create and Publish "UAT No-Answers Exam"

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Navigate to `http://localhost/admin/exams`; click "New Exam" | Exam Wizard Step 1 visible | |
| 2 | Admin | Fill Title: `UAT No-Answers Exam`; Passing Score `70`; Max Attempts `2`; Certificate toggle **OFF**; Show Answers `Never` | Fields set | |
| 3 | Admin | Add 5 single-choice questions; save; publish | Exam "UAT No-Answers Exam" is Active | |
| 4 | Admin | Assign `uat.employee@test.com` | Assignment visible | |

### S0c — Create and Publish "UAT Pending Review Exam"

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Create new exam titled `UAT Pending Review Exam`; add at least 1 short-text question; Certificate **OFF**; Show Answers `After completion`; Max Attempts `2` | Exam created | |
| 2 | Admin | Publish and assign to `uat.employee@test.com` | Exam Active; assignment visible | |

### S0d — Create "uat.noexam" Employee (if absent)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Navigate to `http://localhost/admin/users`; check whether `uat.noexam@test.com` exists | If present, skip. If absent, proceed | |
| 2 | Admin | Click "Invite User" (or "Create User"); fill email `uat.noexam@test.com`, role `employee`, set a known password `NewPass123!` | User created with no exam assignments | |

---

## Scenario S3: Employee — Take and Pass "UAT Result Exam"

> Skip if `uat.employee@test.com` already has a submitted, passing session on "UAT Result Exam".

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Navigate to `http://localhost/login`; fill `uat.employee@test.com` / `NewPass123!`; click Login | Employee portal visible at `/portal` | |
| 2 | Employee | Locate "UAT Result Exam" in the exams list; click "Start Exam" | Exam taking page visible with first question | |
| 3 | Employee | For each of the 5 questions, select the correct answer (as configured in S0a) | All 5 questions answered correctly | |
| 4 | Employee | Click "Submit Exam"; confirm submission | Navigated to result screen at `/portal/sessions/{sessionId}/result` | |
| 5 | Employee | Assert visible: score percentage ≥ 70% | Score shown (e.g. "100%") | |
| 6 | Employee | Record **`{passing_session_id}`** from the URL path (`/portal/sessions/{passing_session_id}/result`) | Session ID noted for subsequent scenarios | |

---

## Scenario S1: Employee — Result Screen Happy Path (FR-BB45)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Log in as `uat.employee@test.com`; navigate to `http://localhost/portal/sessions/{passing_session_id}/result` | Result page loads without error | |
| 2 | Employee | Assert visible: circular progress dial widget containing a numeric percentage (e.g. "100%") | Circular dial rendered with score percentage | |
| 3 | Employee | Assert visible: the dial fill colour matches the tenant's branding primary colour (not plain grey or black) | Dial colour is branded | |
| 4 | Employee | Assert visible: a green banner containing text "Passed" (or i18n equivalent) | Green pass banner displayed | |
| 5 | Employee | Assert not visible: a red banner containing text "Failed" | No failed banner on screen | |
| 6 | Employee | Assert visible: time taken displayed as `Xm Ys` (e.g. "2m 14s") | Time taken badge rendered in correct format | |
| 7 | Employee | Assert not visible: any section score cards (exam has no sections) | No section cards rendered | |
| 8 | Employee | Assert visible: question breakdown table with at least 5 rows | Breakdown table present | |
| 9 | Employee | Assert text contains: each breakdown row shows the employee's selected answer | Employee answer column populated per row | |
| 10 | Employee | Assert visible: correct answer text highlighted in green per row | Correct answer cell/text is green-highlighted | |
| 11 | Employee | Assert visible: "X / Y" points per row (e.g. "1 / 1") | Points earned / max points shown | |
| 12 | Employee | Assert visible: "Download Certificate" button (enabled, not disabled) | Certificate download button present and clickable | |
| 13 | Employee | Assert visible: "Retake Exam" button (attempt 1 of 3; retake is allowed) | Retake button present | |
| 14 | Employee | Assert not visible: "Results Pending Manual Review" notice | No pending notice on this submitted/graded result | |

---

## Scenario S2: Employee — Download Certificate and Verify PDF Content (FR-BB43, FR-BB44)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | On result page for `{passing_session_id}`, click "Download Certificate" | Browser initiates a file download | |
| 2 | Employee | Call API: `GET /api/v1/portal/sessions/{passing_session_id}/certificate` with `{employee_token}` | Response HTTP 200; `Content-Type: application/pdf`; `Content-Disposition: attachment; filename="certificate-{uuid}.pdf"` | |
| 3 | Employee | Record **`{cert_code}`** (the UUID in the Content-Disposition filename) | Verification code UUID noted | |
| 4 | Employee | Assert: downloaded filename matches pattern `certificate-{cert_code}.pdf` | Filename contains "certificate-" and ends ".pdf" | |
| 5 | Employee | Open the downloaded PDF; assert visible: heading "Certificate of Completion" (centred, large font) | Heading present and centred | |
| 6 | Employee | Assert visible: employee's full display name centred prominently below the heading | Employee name present and centred | |
| 7 | Employee | Assert visible: exam title "UAT Result Exam" | Exam title present in PDF | |
| 8 | Employee | Assert visible: score percentage (e.g. "100%") | Score present | |
| 9 | Employee | Assert visible: issue date in human-readable format (e.g. "June 9, 2026") | Issue date present | |
| 10 | Employee | Assert visible: company logo in the top-left area of the PDF, OR company name text if logo is absent | Logo or company name text at top-left | |
| 11 | Employee | Assert visible: QR code in the bottom-right area (visually ≥ 3 cm × 3 cm) | QR code rendered bottom-right | |
| 12 | Employee | Assert visible: verification code UUID text in the footer area | Verification code UUID text in footer | |
| 13 | Employee | Assert visible: signatory name and job title in the bottom-left area, above a horizontal rule | Signatory name and title present bottom-left | |
| 14 | Employee | Assert: PDF orientation is landscape (page is wider than it is tall) | A4 landscape orientation confirmed | |
| 15 | Employee | Click "Download Certificate" a second time | Browser downloads a second file | |
| 16 | Employee | Call API: `GET /api/v1/portal/sessions/{passing_session_id}/certificate` (second request) with `{employee_token}` | Response HTTP 200; Content-Disposition UUID is **identical** to the first request (same `{cert_code}`) | |
| 17 | Employee | Assert: both downloads produce the same verification code | Certificate was reused, not regenerated | |

---

## Scenario S3cert: Public Certificate Verification (FR-BB43 AC-7)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Call API (no auth, no cookies): `GET /api/v1/verify/{cert_code}` | HTTP 200; body: `{ "data": { "valid": true, "employee_name": "...", "exam_title": "UAT Result Exam", "score_pct": 100.0, "issued_at": "..." }, "error": null }` | |
| 2 | Tester | Assert text contains: `"valid": true` | `valid` is `true` | |
| 3 | Tester | Assert text contains: `"exam_title": "UAT Result Exam"` | Exam title present in verification response | |
| 4 | Tester | Assert text contains: `employee_name` is a non-empty string | Employee name present | |
| 5 | Tester | Assert text contains: `issued_at` is a valid UTC ISO 8601 timestamp (e.g. `2026-06-09T...Z`) | Issue date present | |
| 6 | Tester | Call API (no auth): `GET /api/v1/verify/00000000-0000-0000-0000-000000000000` | HTTP 200; body: `{ "data": { "valid": false }, "error": null }` | |
| 7 | Tester | Assert text contains: `"valid": false` | Invalid/unknown code returns `valid: false` | |

---

## Scenario S4: Employee — My Results Tab and History (FR-BB46)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Log in as `uat.employee@test.com`; navigate to `http://localhost/portal` | Employee portal visible with tab navigation | |
| 2 | Employee | Assert visible: "My Results" tab in portal navigation alongside "My Exams" | "My Results" tab is present | |
| 3 | Employee | Click "My Results" tab | Navigates to `http://localhost/portal/results` | |
| 4 | Employee | Assert visible: table header with columns Exam Name, Date, Score, Status, Time, Certificate | All 6 column headers present | |
| 5 | Employee | Assert visible: at least one row for "UAT Result Exam" | Row for "UAT Result Exam" is present | |
| 6 | Employee | Assert text contains: score column for the "UAT Result Exam" row shows a percentage value (e.g. "100.0%") | Score percentage shown | |
| 7 | Employee | Assert visible: green "Passed" badge in the Status column for the passing row | Green pass badge rendered | |
| 8 | Employee | Assert text contains: time taken in "Xm Ys" format in the Time column | Time formatted correctly | |
| 9 | Employee | Assert visible: "Download Certificate" button/link in the Certificate column for the passing "UAT Result Exam" row | Certificate link present (passed + cert_enabled) | |
| 10 | Employee | Click "Download Certificate" from the history row | Browser triggers file download | |
| 11 | Employee | Assert: current URL is still `http://localhost/portal/results` (no page navigation) | Stayed on My Results; no navigation | |
| 12 | Employee | Click the exam name "UAT Result Exam" link in the first column | Browser navigates to `/portal/sessions/{passing_session_id}/result` | |
| 13 | Employee | Assert: URL matches pattern `/portal/sessions/.+/result` | Result detail page loaded | |
| 14 | Employee | Navigate back to `http://localhost/portal/results` | My Results page loads | |
| 15 | Employee | Click the "Date" column sort header | URL updates to include `sort=date` and a `dir` parameter; table reorders | |
| 16 | Employee | Click the "Date" sort header a second time | `dir` parameter in URL toggles between `asc` and `desc`; table order reverses | |
| 17 | Employee | Click the "Score" column sort header | URL updates to include `sort=score`; table reorders by score value | |
| 18 | Employee | Assert text contains: pagination summary text matching "Showing X–Y of Z results" pattern | Pagination summary visible | |
| 19 | Employee | Assert disabled: "Previous" button (on page 1) | Previous button is not clickable | |
| 20 | Employee | Assert disabled: "Next" button (if total results < 20) | Next button disabled when only one page | |
| 21 | Employee | Navigate to `http://localhost/portal`; then navigate back to `http://localhost/portal/results` | Results table reloads and shows same data | |

---

## Scenario S5: Employee — show_answers=never Result Screen (FR-BB41 AC-3, FR-BB45 AC-5 negative)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Log in as `uat.employee@test.com`; navigate to `http://localhost/portal` | Employee portal visible | |
| 2 | Employee | Locate "UAT No-Answers Exam"; click "Start Exam" | Exam taking page visible | |
| 3 | Employee | Answer all 5 questions (any answers); click "Submit Exam"; confirm | Navigated to result screen for this session | |
| 4 | Employee | Record **`{noAnswers_session_id}`** from the URL | Session ID noted | |
| 5 | Employee | Assert visible: score percentage | Score displayed | |
| 6 | Employee | Assert visible: pass or fail banner | Pass/fail banner shown | |
| 7 | Employee | Assert not visible: question breakdown table | No review table rendered | |
| 8 | Employee | Assert not visible: correct answer column or any per-question details | No correct answers shown | |
| 9 | Employee | Assert not visible: "Download Certificate" button (cert_enabled=false for this exam) | No certificate button present | |
| 10 | Employee | Call API: `GET /api/v1/portal/sessions/{noAnswers_session_id}/result` with `{employee_token}` | Response body: `per_question_breakdown` key is **absent** from the `data` object | |

---

## Scenario S6: Admin — View Any Session Result (FR-BB41 AC-5, FR-BB43 AC-8)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Admin | Log in as `admin@bilimbaga.local` / `Admin1234!`; navigate to `http://localhost/admin/users` | Users list visible | |
| 2 | Admin | Find and click `uat.employee@test.com` | Employee record / detail page visible | |
| 3 | Admin | Assert visible: session history row for "UAT Result Exam" with score and pass/fail indicator | Session row present | |
| 4 | Admin | Call API: `GET /api/v1/admin/sessions/{passing_session_id}/result` with `{admin_token}` | HTTP 200; response `data` contains `per_question_breakdown` array | |
| 5 | Admin | Assert text contains: each breakdown item has `employee_answer`, `correct_answer`, `points_earned`, `max_points` fields | Full breakdown always returned for admin | |
| 6 | Admin | Assert text contains: `score_pct`, `passed`, `time_taken_seconds`, `submitted_at` all present in response | All summary fields present | |
| 7 | Admin | Call API: `GET /api/v1/admin/sessions/{passing_session_id}/certificate` with `{admin_token}` | HTTP 200; `Content-Type: application/pdf` | |
| 8 | Admin | Assert: response body is binary PDF data (starts with `%PDF-`) | Admin can download any session certificate | |

---

## Scenario S7: FR-BB41 AC-7 — No Sections → No per_section_scores in API and UI

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Navigate to `http://localhost/portal/sessions/{passing_session_id}/result` | Result page loaded | |
| 2 | Employee | Assert not visible: any section score card component on the result page | No section cards rendered | |
| 3 | Tester | Call API: `GET /api/v1/portal/sessions/{passing_session_id}/result` with `{employee_token}` | Response body: `per_section_scores` is `[]` (empty array) | |

---

## Scenario S8: FR-BB46 AC-6 — Empty My Results State

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Log in as `uat.noexam@test.com` / `NewPass123!` | Employee portal visible | |
| 2 | Employee | Click "My Results" tab | Navigates to `/portal/results` | |
| 3 | Employee | Assert visible: empty state component (illustration or message; no table with data rows) | Empty state shown instead of results table | |
| 4 | Employee | Assert not visible: a table containing session data rows | No data table rendered | |

---

## Scenario S9: FR-BB45 AC-8 — Pending Manual Review Notice

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Log in as `uat.employee@test.com`; navigate to `http://localhost/portal` | Portal visible | |
| 2 | Employee | Locate "UAT Pending Review Exam" (short-text exam); click "Start Exam" | Exam taking page visible | |
| 3 | Employee | Fill answer text for the short-text question(s); click "Submit Exam"; confirm | Navigated to result screen (`/portal/sessions/{pending_session_id}/result`) | |
| 4 | Employee | Assert visible: "Results Pending Manual Review" notice (or i18n equivalent, e.g. `result.pending_grading` key) | Pending grading notice rendered prominently | |
| 5 | Employee | Assert not visible: circular score dial | No score dial | |
| 6 | Employee | Assert not visible: pass/fail banner | No pass or fail banner | |
| 7 | Employee | Assert not visible: "Download Certificate" button | Certificate button absent | |
| 8 | Employee | Assert not visible: "Retake Exam" button | Retake button absent | |

---

## Scenario S10: API Error Cases (FR-BB41 ACs 1, 2, 9, 11; FR-BB43 ACs 1, 2, 3, 4)

> Use `Call API` steps with an HTTP client. Tokens are captured during login flows. `{other_session_id}` refers to any session ID that belongs to a different user; the admin may look one up via `GET /api/v1/admin/sessions` if needed.

### S10a — FR-BB41 AC-1: Session belonging to another user returns 403 for employee

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Obtain `{other_session_id}` — a submitted session belonging to any user other than `uat.employee@test.com` | Session ID available | |
| 2 | Tester | Call API: `GET /api/v1/portal/sessions/{other_session_id}/result` with `{employee_token}` | HTTP 403; body: `{ "data": null, "error": { "code": "SESSION_FORBIDDEN", ... } }` | |

### S10b — FR-BB41 AC-2: In-progress session returns 422 on portal result endpoint

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Employee | Log in as `uat.employee@test.com`; start "UAT No-Answers Exam" second attempt (or any available exam) — DO NOT SUBMIT | Exam taking page visible; session is in `in_progress` state | |
| 2 | Tester | Record **`{inprogress_session_id}`** from the exam-taking page URL | Session ID noted | |
| 3 | Tester | Call API: `GET /api/v1/portal/sessions/{inprogress_session_id}/result` with `{employee_token}` | HTTP 422; body: `{ "data": null, "error": { "code": "SESSION_IN_PROGRESS", ... } }` | |
| 4 | Employee | Navigate away from the exam (abandon without submitting) | In-progress session retained for S10d and S10h | |

### S10c — FR-BB41 AC-9: Non-existent session returns 404

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Call API: `GET /api/v1/portal/sessions/00000000-0000-0000-0000-000000000000/result` with `{employee_token}` | HTTP 404; body: `{ "data": null, "error": { "code": "SESSION_NOT_FOUND", ... } }` | |

### S10d — FR-BB41 AC-11: Admin result endpoint returns 422 for in-progress session

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Call API: `GET /api/v1/admin/sessions/{inprogress_session_id}/result` with `{admin_token}` | HTTP 422; body: `{ "data": null, "error": { "code": "SESSION_IN_PROGRESS", ... } }` | |

### S10e — FR-BB43 AC-1: Certificate for another user's session returns 403

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Call API: `GET /api/v1/portal/sessions/{other_session_id}/certificate` with `{employee_token}` | HTTP 403; body contains error code | |

### S10f — FR-BB43 AC-2: Certificate for cert_enabled=false exam returns 422

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Ensure `{noAnswers_session_id}` is set (from Scenario S5) | Session ID available | |
| 2 | Tester | Call API: `GET /api/v1/portal/sessions/{noAnswers_session_id}/certificate` with `{employee_token}` | HTTP 422; body: `{ "data": null, "error": { "code": "EXAM_NOT_CERTIFIABLE", ... } }` | |

### S10g — FR-BB43 AC-3: Certificate for a failed session returns 422

> If `uat.employee@test.com` has a submitted session with `passed=false` (e.g., from answering incorrectly on the No-Answers Exam), use its session ID as `{failed_session_id}`. Otherwise, have the employee retake an exam with wrong answers to create one.

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Obtain `{failed_session_id}` — a submitted session with `passed=false` for `uat.employee@test.com` | Session ID available | |
| 2 | Tester | Call API: `GET /api/v1/portal/sessions/{failed_session_id}/certificate` with `{employee_token}` | HTTP 422; body: `{ "data": null, "error": { "code": "SESSION_NOT_PASSED", ... } }` | |

### S10h — FR-BB43 AC-4: Certificate for in-progress session returns 422

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Call API: `GET /api/v1/portal/sessions/{inprogress_session_id}/certificate` with `{employee_token}` | HTTP 422; body: `{ "data": null, "error": { "code": "SESSION_NOT_SUBMITTED", ... } }` | |

---

## Scenario S11: FR-BB46 AC-11 / FR-BB45 AC-9 — API Error → Inline Error Message

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Using Playwright route interception, mock `GET /api/v1/portal/results` to return HTTP 500 | Network interception active | |
| 2 | Employee | Navigate to `http://localhost/portal/results` | Page loads with mocked failure | |
| 3 | Employee | Assert visible: inline error message text (e.g. "Failed to load results." or i18n equivalent `common.loadError`) | Error message rendered on page | |
| 4 | Employee | Assert not visible: a table with result rows | No data table rendered | |
| 5 | Tester | Remove Playwright route interception | Interception cleared | |
| 6 | Tester | Using Playwright route interception, mock `GET /api/v1/portal/sessions/{passing_session_id}/result` to return HTTP 500 | Network interception active | |
| 7 | Employee | Navigate to `http://localhost/portal/sessions/{passing_session_id}/result` | Page loads with mocked failure | |
| 8 | Employee | Assert visible: error alert or message (not a blank/broken page) | Error state rendered with appropriate message | |
| 9 | Tester | Remove route interception | Interception cleared | |

---

## Scenario S12: FR-BB41 AC-6, AC-10 — Per-Exam History Ordering and Pagination

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Tester | Obtain `{uatResultExamId}` from any API response or admin exam list for "UAT Result Exam" | Exam UUID noted | |
| 2 | Tester | Call API: `GET /api/v1/portal/exams/{uatResultExamId}/history?page=1&per_page=20` with `{employee_token}` | HTTP 200; `data.sessions` array returned | |
| 3 | Tester | Assert: every session in `data.sessions` has `status = 'submitted'`; no `in_progress` sessions included | In-progress sessions excluded | |
| 4 | Tester | Assert: sessions are ordered ascending by `started_at` (each entry's `started_at` ≥ previous entry) | Ascending chronological order confirmed | |
| 5 | Tester | Assert text contains: `data.meta.total` is a positive integer | Pagination metadata present | |

---

## Acceptance Criteria Coverage

| FR | AC# | Criterion | Covered by |
|----|-----|-----------|------------|
| FR-BB41 | AC-1 | Portal session result returns 403 for wrong user | S10a |
| FR-BB41 | AC-2 | Portal session result returns 422 SESSION_IN_PROGRESS | S10b Step 3 |
| FR-BB41 | AC-3 | show_answers=never → per_question_breakdown absent from response | S5 Steps 7–10 |
| FR-BB41 | AC-4 | show_answers=after_completion → full breakdown with all fields | S1 Steps 8–11 |
| FR-BB41 | AC-5 | Admin endpoint always includes full breakdown regardless of show_answers | S6 Steps 4–6 |
| FR-BB41 | AC-6 | History ascending by started_at; in_progress excluded | S12 Steps 3–4 |
| FR-BB41 | AC-7 | per_section_scores absent for no-section exam | S7 Steps 2–3 |
| FR-BB41 | AC-8 | time_taken_seconds displayed as Xm Ys in result UI | S1 Step 6 |
| FR-BB41 | AC-9 | 404 SESSION_NOT_FOUND for non-existent session ID | S10c |
| FR-BB41 | AC-10 | History endpoint paginated with meta.total | S12 Step 5 |
| FR-BB41 | AC-11 | Admin result endpoint returns 422 SESSION_IN_PROGRESS | S10d |
| FR-BB43 | AC-1 | 403 for wrong user certificate request | S10e |
| FR-BB43 | AC-2 | 422 EXAM_NOT_CERTIFIABLE for cert_enabled=false exam | S10f |
| FR-BB43 | AC-3 | 422 SESSION_NOT_PASSED for failed session | S10g |
| FR-BB43 | AC-4 | 422 SESSION_NOT_SUBMITTED for in_progress session | S10h |
| FR-BB43 | AC-5 | Lazy creation: first request creates; subsequent requests reuse record | S2 Steps 15–17 |
| FR-BB43 | AC-6 | Content-Type: application/pdf; Content-Disposition with correct filename | S2 Steps 2–4 |
| FR-BB43 | AC-7 | Public verify endpoint: valid code → valid:true; unknown code → valid:false | S3cert |
| FR-BB43 | AC-8 | Admin can download any session's certificate with exams:read | S6 Steps 7–8 |
| FR-BB43 | AC-9 | template_snapshot captures branding at issuance (logo, signatory in PDF) | S2 Steps 10–13 |
| FR-BB43 | AC-10 | Idempotent creation (same verification code on repeat requests) | S2 Steps 15–17 |
| FR-BB44 | AC-2 | A4 landscape orientation | S2 Step 14 |
| FR-BB44 | AC-3 | Company logo top-left; fallback to company name text | S2 Step 10 |
| FR-BB44 | AC-4 | "Certificate of Completion" centred ≥28pt; employee name centred ≥22pt | S2 Steps 5–6 |
| FR-BB44 | AC-5 | Exam title, score%, issue date present | S2 Steps 7–9 |
| FR-BB44 | AC-6 | QR code bottom-right ≥30mm×30mm | S2 Step 11 |
| FR-BB44 | AC-7 | Verification code printed in footer | S2 Step 12 |
| FR-BB44 | AC-8 | Signatory name and title bottom-left | S2 Step 13 |
| FR-BB45 | AC-1 | Score circular progress dial with tenant primary_color | S1 Steps 2–3 |
| FR-BB45 | AC-2 | Green "Passed" or red "Failed" banner | S1 Steps 4–5 |
| FR-BB45 | AC-3 | Time taken in "Xm Ys" format; hidden if null | S1 Step 6 |
| FR-BB45 | AC-4 | Section score cards when per_section_scores non-empty (absence tested; presence requires sectioned exam) | S1 Step 7 (partial) |
| FR-BB45 | AC-5 | Breakdown table: stem, employee answer, correct answer (green), points, explanation | S1 Steps 8–11 |
| FR-BB45 | AC-6 | Download Certificate button only when passed=true AND cert_enabled=true | S1 Step 12; S5 Step 9 |
| FR-BB45 | AC-7 | Retake Exam button only when attempt < max_attempts AND not expired | S1 Step 13 |
| FR-BB45 | AC-8 | score_pct null → "Results Pending Manual Review" notice; cert/retake hidden | S9 Steps 4–8 |
| FR-BB45 | AC-9 | Loading/error states with skeleton/alert | S11 Steps 7–8 |
| FR-BB45 | AC-10 | Zero hardcoded user-visible strings | S1 (all labels via i18n); S4 Step 4 |
| FR-BB46 | AC-1 | "My Results" tab visible in employee portal navigation | S4 Steps 1–2 |
| FR-BB46 | AC-2 | Table columns: exam name, date, score%, pass/fail badge, time, certificate link | S4 Steps 4–8 |
| FR-BB46 | AC-3 | Certificate link only when passed=true AND cert_enabled=true | S4 Step 9 |
| FR-BB46 | AC-4 | Date and Score sortable; sort state reflected in URL ?sort=&dir= | S4 Steps 15–17 |
| FR-BB46 | AC-5 | Pagination: 20 rows/page; Previous/Next buttons; "Showing X–Y of Z results" | S4 Steps 18–20 |
| FR-BB46 | AC-6 | Empty state rendered when no sessions exist | S8 |
| FR-BB46 | AC-7 | Clicking exam name navigates to /portal/sessions/:id/result | S4 Steps 12–13 |
| FR-BB46 | AC-8 | Certificate download link triggers file download without navigating away | S4 Steps 10–11 |
| FR-BB46 | AC-9 | React Query refetches on re-navigation | S4 Step 21 |
| FR-BB46 | AC-10 | Zero hardcoded user-visible strings | S4 Step 4 (column headers use i18n keys) |
| FR-BB46 | AC-11 | API error → inline error message shown; table not rendered | S11 Steps 2–4 |

## Out of Scope

- Phase 7 AI-assisted features
- Manual grading workflow (FR-BB42)
- Admin analytics and department reports (FR-BB51)
- Email notification triggers on result or certificate events (FR-BB61)
- Push notifications
- FR-BB44 AC-1 (function signature unit test), AC-9 (invalid base64 error handling), AC-10 (unit test coverage) — backend unit test scope, not UAT-applicable
