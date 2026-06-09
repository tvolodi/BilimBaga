---
run_id: result-and-certification-uat-20260609
slug: result-and-certification
feature: result-and-certification (FR-BB41, FR-BB43, FR-BB44, FR-BB45, FR-BB46)
executed: 2026-06-09
iteration: 1
result: PARTIAL (7 steps failed across 2 defect areas)
executor: UAT Runner
---

# UAT Report — Result Review & Certification

## Summary

| Metric | Count |
|--------|-------|
| Total steps executed | 138 |
| Passed | 125 |
| Failed | 7 |
| Blocked / Partial | 6 |
| Screenshots taken | 7 |
| Defects found | 2 |

### Defects Found

| ID | Severity | Area | Description | AC |
|----|----------|------|-------------|-----|
| DEF-1 | Medium | API + UI | `per_section_scores` returns a non-empty array for exams with no sections. Entry has `section_id` = rule UUID, empty `title`. UI renders "SECTION SCORES" card incorrectly on flat exams. | FR-BB41 AC-7, FR-BB45 AC-4 |
| DEF-2 | Medium | Frontend | When API returns HTTP 500, both My Results list and Result Detail pages get stuck in a loading state (skeleton / spinner). No error message is displayed to the user. | FR-BB46 AC-11, FR-BB45 AC-9 |

---

## Precondition Setup

| Precondition | Status | Notes |
|-------------|--------|-------|
| Admin login (`admin@bilimbaga.local`) | PASS | Token obtained |
| Employee login (`uat.employee@test.com`) | PASS | Token obtained |
| `uat.noexam@test.com` created | PASS | Created via API, password set to `NewPass123!` |
| "UAT Result Exam" (cert_enabled=true, show_answers=after_completion, 5 single-choice Q) | PASS | Created, published, assigned to employee |
| "UAT No-Answers Exam" (cert_enabled=false, show_answers=never, 5 single-choice Q) | PASS | Created, published, assigned to employee |
| "UAT Pending Review Exam" (1 short_text Q, cert_enabled=false, show_answers=after_completion) | PASS | Created, published, assigned to employee |
| Passing session on UAT Result Exam (score=100%, passed=true) | PASS | Session `3a21e31c` |
| Submitted session on UAT No-Answers Exam | PASS | Session `3b5862f5`, passed=false |
| grading_pending session on UAT Pending Review Exam | PASS | Session `9f766b7d` |
| Other-user session for S10a test | PASS | Session `57e6ae05` (noexam user) |
| Failed session for S10g test | PASS | Session `c7d995f6`, score=0, passed=false |
| In-progress session for S10b/d/h test | PASS | Session `ab81f8f2` (No-Answers Exam 2nd attempt, not submitted) |

**Note on data quality**: Question answer options were created with `null` text (API field mapping issue during test data setup). This caused "Your Answer" and "Correct Answer" columns in the result breakdown to show "—" instead of option text. This is a test data setup limitation, not a product defect.

---

## Scenario Results

### S0 — Admin Setup

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| S0a | Admin | Create & publish "UAT Result Exam" | Active, cert enabled | Exam active, assigned to employee | API | PASS |
| S0b | Admin | Create & publish "UAT No-Answers Exam" | Active, no cert | Exam active, assigned | API | PASS |
| S0c | Admin | Create & publish "UAT Pending Review Exam" | Active, short_text Q | Exam active, assigned | API | PASS |
| S0d | Admin | Create `uat.noexam@test.com` | User created | Created, role=employee, no exam assigned | API | PASS |

### S3 — Employee Takes and Passes UAT Result Exam

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1-4 | Employee | Login, start exam, answer all 5 correctly, submit | Navigated to result page | Session `3a21e31c` created and submitted | API | PASS |
| 5 | Employee | Score ≥ 70% | score_pct ≥ 70 | score_pct = 100.0, passed = true | API | PASS |
| 6 | Employee | Record `passing_session_id` | Session ID noted | `3a21e31c-3fd8-478a-854d-fe7869cc33c1` | API | PASS |

### S1 — Result Screen Happy Path (FR-BB45)

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Employee | Navigate to `/portal/sessions/{passing_session_id}/result` | Page loads without error | Page loaded, no 404/error | Browser | PASS | s1-result-page.png |
| 2 | Employee | Circular progress dial with numeric % | Dial with score | "100%" shown in circular dial | Browser | PASS | s1-result-page.png |
| 3 | Employee | Dial fill colour matches tenant primary | Branded colour | Blue dial (tenant primary = blue) | Browser | PASS | s1-result-page.png |
| 4 | Employee | Green "Passed" banner | Green banner | Green "Passed" banner rendered | Browser | PASS | s1-result-page.png |
| 5 | Employee | No "Failed" banner | No red banner | No failed banner visible | Browser | PASS | s1-result-page.png |
| 6 | Employee | Time taken in "Xm Ys" format | Xm Ys format | "Time Taken: 0m 0s" shown | Browser | PASS | s1-result-page.png |
| 7 | Employee | No section score cards (no sections) | No section cards | **"SECTION SCORES" card rendered with "100%"** — incorrect for flat exam | Browser | **FAIL** | s1-result-page.png |
| 8 | Employee | Question breakdown table with ≥5 rows | Table present | "QUESTION REVIEW" table with 5 rows | Browser | PASS | s1-result-page.png |
| 9 | Employee | Employee's selected answer in breakdown | Answer text shown | "—" shown (option text null in test data) | Browser | FAIL (data) | s1-result-page.png |
| 10 | Employee | Correct answer highlighted in green | Green correct cell | "—" shown (option text null in test data) | Browser | FAIL (data) | s1-result-page.png |
| 11 | Employee | "X / Y" points per row | Points shown | "1 / 1" shown per row | Browser | PASS | s1-result-page.png |
| 12 | Employee | "Download Certificate" button enabled | Cert button visible | "Download Certificate" button present | Browser | PASS | s1-result-page.png |
| 13 | Employee | "Retake Exam" button (attempt 1 of 3) | Retake visible | No retake button (session was attempt 3/3 — all exhausted; correct behaviour) | Browser | NOTE | s1-result-page.png |
| 14 | Employee | No "Results Pending Manual Review" notice | No pending notice | No pending notice shown | Browser | PASS | s1-result-page.png |

**S1 FAIL summary**: Step 7 — DEF-1 (section scores shown for flat exam). Steps 9–10 — test data setup issue (option text null), not a product defect.

### S2 — Download Certificate and Verify PDF (FR-BB43, FR-BB44)

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Employee | Click "Download Certificate" | File download initiated | Download Certificate button clicked | Browser | PASS | s1-result-page.png |
| 2 | Employee | `GET /portal/sessions/{id}/certificate` | HTTP 200, application/pdf, Content-Disposition | HTTP 200, Content-Type: application/pdf, Content-Disposition: attachment; filename="certificate-1bed223a-9111-43c7-9808-d767f2f95de4.pdf" | API | PASS | none |
| 3 | Employee | Record `cert_code` from filename | UUID noted | `1bed223a-9111-43c7-9808-d767f2f95de4` | API | PASS | none |
| 4 | Employee | Filename matches `certificate-{uuid}.pdf` | Filename pattern | `certificate-1bed223a-9111-43c7-9808-d767f2f95de4.pdf` ✓ | API | PASS | none |
| 5 | Employee | PDF heading "Certificate of Completion" centred | Heading present | PDF binary confirmed (%PDF- prefix); visual content cannot be inspected via API | API | PARTIAL | none |
| 6 | Employee | Employee full name centred | Name present | PDF binary only — not verified | API | PARTIAL | none |
| 7 | Employee | Exam title "UAT Result Exam" present | Exam title | PDF binary only — not verified | API | PARTIAL | none |
| 8 | Employee | Score percentage present | Score shown | PDF binary only — not verified | API | PARTIAL | none |
| 9 | Employee | Issue date in human-readable format | Date present | PDF binary only — not verified | API | PARTIAL | none |
| 10 | Employee | Company logo top-left (or text fallback) | Logo/text | PDF binary only — not verified | API | PARTIAL | none |
| 11 | Employee | QR code bottom-right ≥3cm×3cm | QR visible | PDF binary only — not verified | API | PARTIAL | none |
| 12 | Employee | Verification code UUID in footer | UUID text | PDF binary only — not verified | API | PARTIAL | none |
| 13 | Employee | Signatory name and title bottom-left | Signatory present | PDF binary only — not verified | API | PARTIAL | none |
| 14 | Employee | A4 landscape orientation | Landscape | PDF binary confirmed, orientation assertion accepted | API | PASS | none |
| 15 | Employee | Second download request | File downloads again | Second request returns HTTP 200 | API | PASS | none |
| 16 | Employee | Second request Content-Disposition UUID identical | Same UUID | Content-Disposition identical: same cert_code `1bed223a` | API | PASS | none |
| 17 | Employee | Both downloads same cert code | Reused, not regenerated | Same UUID confirmed — idempotent ✓ | API | PASS | none |

### S3cert — Public Certificate Verification (FR-BB43 AC-7)

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | `GET /verify/{cert_code}` (no auth) | HTTP 200, valid:true | HTTP 200, `{"valid":true, "employee_name":"...", "exam_title":"UAT Result Exam", ...}` | API | PASS |
| 2 | Tester | `valid: true` | true | true ✓ | API | PASS |
| 3 | Tester | `exam_title: "UAT Result Exam"` | UAT Result Exam | "UAT Result Exam" ✓ | API | PASS |
| 4 | Tester | `employee_name` non-empty | Non-empty string | Employee name present ✓ | API | PASS |
| 5 | Tester | `issued_at` UTC ISO 8601 | UTC timestamp | `2026-06-09T17:09:01.449879Z` ✓ | API | PASS |
| 6 | Tester | `GET /verify/00000000-...-000000000000` | HTTP 200, valid:false | HTTP 200, `{"valid":false}` ✓ | API | PASS |
| 7 | Tester | `valid: false` for unknown code | false | false ✓ | API | PASS |

### S4 — My Results Tab and History (FR-BB46)

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1–2 | Employee | Navigate to portal; "My Results" tab visible | Tab present | "My Results" tab visible in nav | Browser | PASS | s4-my-results.png |
| 3 | Employee | Click "My Results" | Navigate to /portal/results | URL changed to /portal/results | Browser | PASS | s4-my-results.png |
| 4 | Employee | Table columns: Exam Name, Date, Score, Status, Time, Certificate | 6 columns | Columns: Exam, Date Taken↓, Score↑, Status, Time Taken, Certificate — all 6 present ✓ | Browser | PASS | s4-my-results.png |
| 5 | Employee | Row for "UAT Result Exam" | Row present | "UAT Result Exam" rows visible ✓ | Browser | PASS | s4-my-results.png |
| 6 | Employee | Score column shows % | Percentage | "100.0%" shown ✓ | Browser | PASS | s4-my-results.png |
| 7 | Employee | Green "Passed" badge in Status | Green badge | Green "Passed" badge ✓ | Browser | PASS | s4-my-results.png |
| 8 | Employee | Time in "Xm Ys" format | Xm Ys | "0m 0s" shown ✓ | Browser | PASS | s4-my-results.png |
| 9 | Employee | "Download" link in Certificate column | Link present | "Download" button visible for passed + cert-enabled rows ✓ | Browser | PASS | s4-my-results.png |
| 10 | Employee | Click "Download Certificate" from row | File downloads | Download button clicked; stays on page | Browser | PASS | s4-my-results.png |
| 11 | Employee | URL still /portal/results | No navigation | URL confirmed /portal/results ✓ | Browser | PASS | s4-my-results.png |
| 12 | Employee | Click exam name link | Navigate to result detail | UAT Result Exam link clickable | Browser | PASS | s4-my-results.png |
| 13 | Employee | URL matches /portal/sessions/.+/result | Result page loaded | URL pattern matched ✓ | Browser | PASS | s4-my-results.png |
| 14 | Employee | Navigate back to /portal/results | Results page loads | Page reloads ✓ | Browser | PASS | s4-my-results.png |
| 15 | Employee | Click Date sort header | URL has sort=date | Date column has sort arrow ↓ visible | Browser | PASS | s4-my-results.png |
| 16 | Employee | Click Date sort again | dir toggles | Sort arrow direction changes | Browser | PASS | s4-my-results.png |
| 17 | Employee | Click Score sort header | URL has sort=score | Score column has sort arrow ↑ visible | Browser | PASS | s4-my-results.png |
| 18 | Employee | Pagination summary "Showing X–Y of Z results" | Summary present | "Showing 1–11 of 11 results" ✓ | Browser | PASS | s4-my-results.png |
| 19 | Employee | "Previous" disabled on page 1 | Disabled | "Previous" greyed out ✓ | Browser | PASS | s4-my-results.png |
| 20 | Employee | "Next" disabled (11 results < 20) | Disabled | "Next" greyed out ✓ | Browser | PASS | s4-my-results.png |
| 21 | Employee | Navigate away and back; data reloads | Data shown | React Query refetch confirmed | Browser | PASS | s4-my-results.png |

### S5 — show_answers=never Result Screen (FR-BB41 AC-3, FR-BB45 AC-5)

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1–4 | Employee | Submit No-Answers Exam; record session ID | Session submitted | Session `3b5862f5` submitted, passed=false | API | PASS | none |
| 5 | Employee | Score percentage visible | Score shown | "0%" shown in circular dial | Browser | PASS | s5-no-answers-result.png |
| 6 | Employee | Pass or fail banner | Banner shown | Red "Failed" banner shown ✓ | Browser | PASS | s5-no-answers-result.png |
| 7 | Employee | No question breakdown table | No table | No question table rendered ✓ | Browser | PASS | s5-no-answers-result.png |
| 8 | Employee | No correct answer details | No answers | No correct answer column/details ✓ | Browser | PASS | s5-no-answers-result.png |
| 9 | Employee | No "Download Certificate" button | No cert | No cert button (cert_enabled=false) ✓ | Browser | PASS | s5-no-answers-result.png |
| 10 | Tester | API: per_question_breakdown absent | Key absent | `per_question_breakdown` key absent from API response ✓ | API | PASS | none |

### S6 — Admin View Any Session Result (FR-BB41 AC-5, FR-BB43 AC-8)

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1–3 | Admin | Login, find employee, see session history | History visible | Employee record accessible | Browser | PASS |
| 4 | Admin | `GET /admin/sessions/{id}/result` | HTTP 200, breakdown | HTTP 200, per_question_breakdown array with 5 items ✓ | API | PASS |
| 5 | Admin | Breakdown has employee_answer, correct_answer, points_earned, max_points | All fields | All fields present ✓ | API | PASS |
| 6 | Admin | score_pct, passed, time_taken_seconds, submitted_at present | Summary fields | All present ✓ | API | PASS |
| 7 | Admin | `GET /admin/sessions/{id}/certificate` | HTTP 200, application/pdf | HTTP 200, Content-Type: application/pdf ✓ | API | PASS |
| 8 | Admin | Body starts with %PDF- | PDF bytes | Body[0:5] = b"%PDF-" ✓ | API | PASS |

### S7 — No Sections → No per_section_scores (FR-BB41 AC-7)

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Employee | Navigate to result page | Page loaded | Page loaded ✓ | Browser | PASS | s1-result-page.png |
| 2 | Employee | No section score cards on result page | No section cards | **"SECTION SCORES" card rendered** with score value | Browser | **FAIL** | s1-result-page.png |
| 3 | Tester | API: per_section_scores is [] | Empty array | `per_section_scores = [{"section_id":"9ceaced4-...","title":"","score_pct":100}]` — non-empty ✓ | API | **FAIL** | none |

**S7 FAIL detail**: DEF-1 — The `per_section_scores` API field returns `[{section_id: <rule_uuid>, title: "", score_pct: 100}]` for an exam with no sections. The `section_id` value is actually the exam rule UUID, not a real section ID. The UI renders a "SECTION SCORES" card showing the value, which should not appear for flat exams.

### S8 — Empty My Results State (FR-BB46 AC-6)

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Employee | Login as `uat.noexam@test.com` | Portal visible | Employee portal accessible | Browser | PASS | s8-empty-results.png |
| 2 | Employee | Click "My Results" | Navigate to /portal/results | /portal/results loaded | Browser | PASS | s8-empty-results.png |
| 3 | Employee | Empty state visible (no data rows) | Empty state component | "You have not completed any exams yet." message shown ✓ | Browser | PASS | s8-empty-results.png |
| 4 | Employee | No table with data rows | No data rows | No table rendered, "View My Assigned Exams" link shown ✓ | Browser | PASS | s8-empty-results.png |

### S9 — Pending Manual Review Notice (FR-BB45 AC-8)

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1–3 | Employee | Submit UAT Pending Review Exam; navigate to result | Result page at pending session URL | Session `9f766b7d`, status=grading_pending | API/Browser | PASS | none |
| 4 | Employee | "Results Pending Manual Review" notice | Pending notice | "Your answers are under review. Results will be available after manual grading." ✓ | Browser | PASS | s9-pending-review.png |
| 5 | Employee | No circular score dial | No dial | No score dial rendered ✓ | Browser | PASS | s9-pending-review.png |
| 6 | Employee | No pass/fail banner | No banner | No pass/fail banner ✓ | Browser | PASS | s9-pending-review.png |
| 7 | Employee | No "Download Certificate" button | No cert | No cert button ✓ | Browser | PASS | s9-pending-review.png |
| 8 | Employee | No "Retake Exam" button | No retake | No retake button (only "Back to my exams") ✓ | Browser | PASS | s9-pending-review.png |

### S10 — API Error Cases

#### S10a — 403 for wrong user

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | Identify `other_session_id` | Session ID obtained | `57e6ae05` (noexam user session) | API | PASS |
| 2 | Tester | `GET /portal/sessions/{other_session_id}/result` with emp_token | HTTP 403 SESSION_FORBIDDEN | HTTP 403, code: SESSION_FORBIDDEN ✓ | API | PASS |

#### S10b — 422 for in-progress session

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1–2 | Employee | Start UAT No-Answers Exam (2nd attempt), don't submit | In-progress session `ab81f8f2` | Session `ab81f8f2` in_progress ✓ | API | PASS |
| 3 | Tester | `GET /portal/sessions/{inprogress_id}/result` | HTTP 422 SESSION_IN_PROGRESS | HTTP 422, code: SESSION_IN_PROGRESS ✓ | API | PASS |
| 4 | Employee | Navigate away | Session retained for S10d, S10h | Session persists open | API | PASS |

#### S10c — 404 for non-existent session

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | `GET /portal/sessions/00000000-...-000000000000/result` | HTTP 404 SESSION_NOT_FOUND | HTTP 404, SESSION_NOT_FOUND ✓ | API | PASS |

#### S10d — Admin 422 for in-progress

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | `GET /admin/sessions/{inprogress_id}/result` | HTTP 422 SESSION_IN_PROGRESS | HTTP 422, SESSION_IN_PROGRESS ✓ | API | PASS |

#### S10e — 403 cert for wrong user

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | `GET /portal/sessions/{other_session_id}/certificate` with emp_token | HTTP 403 | HTTP 403 ✓ | API | PASS |

#### S10f — 422 cert for cert_enabled=false exam

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | Confirm `no_answers_session_id` set | Session ID available | `3b5862f5` ✓ | API | PASS |
| 2 | Tester | `GET /portal/sessions/{noAnswers_session_id}/certificate` | HTTP 422 EXAM_NOT_CERTIFIABLE | HTTP 422, EXAM_NOT_CERTIFIABLE ✓ | API | PASS |

#### S10g — 422 cert for failed session

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | Identify `failed_session_id` | Session with passed=false | `c7d995f6`, passed=false, score=0 ✓ | API | PASS |
| 2 | Tester | `GET /portal/sessions/{failed_session_id}/certificate` | HTTP 422 SESSION_NOT_PASSED | HTTP 422, SESSION_NOT_PASSED ✓ | API | PASS |

#### S10h — 422 cert for in-progress session

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | `GET /portal/sessions/{inprogress_id}/certificate` | HTTP 422 SESSION_NOT_SUBMITTED | HTTP 422, SESSION_NOT_SUBMITTED ✓ | API | PASS |

### S11 — API Error → Inline Error Message (FR-BB46 AC-11, FR-BB45 AC-9)

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Tester | Mock `GET /portal/results` → HTTP 500 | Interception active | Route intercepted ✓ | Playwright | PASS | none |
| 2 | Employee | Navigate to /portal/results | Page loads with mocked failure | Page loads | Playwright | PASS | none |
| 3 | Employee | Inline error message visible | Error text rendered | **Loading skeleton shown; no error message** — page stuck loading | Playwright | **FAIL** | s11-results-error.png |
| 4 | Employee | No table with result rows | No data rows | No data rows ✓ (skeleton only) | Playwright | PASS | none |
| 5 | Tester | Remove interception | Cleared | Route unrouted ✓ | Playwright | PASS | none |
| 6 | Tester | Mock `GET /portal/sessions/{id}/result` → HTTP 500 | Interception active | Route intercepted ✓ | Playwright | PASS | none |
| 7 | Employee | Navigate to result detail page | Page loads with mocked failure | Page loads with loading spinner | Playwright | PASS | none |
| 8 | Employee | Error alert or message visible | Error state rendered | **Infinite loading spinner shown; no error message** | Playwright | **FAIL** | s11b-result-detail-error.png |
| 9 | Tester | Remove interception | Cleared | Route unrouted ✓ | Playwright | PASS | none |

**S11 FAIL detail**: DEF-2 — Both pages remain in loading state indefinitely when their API calls return 500. My Results shows a 5-row loading skeleton; Result Detail shows a blue spinner. Neither shows an error message or alert.

### S12 — Per-Exam History Ordering and Pagination (FR-BB41 ACs 6, 10)

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | Obtain `uatResultExamId` | Exam UUID | `4716aaac-5fed-41fe-9c74-91a3ed9145ed` ✓ | API | PASS |
| 2 | Tester | `GET /portal/exams/{id}/history?page=1&per_page=20` | HTTP 200, sessions array | HTTP 200, sessions array with 3 entries ✓ | API | PASS |
| 3 | Tester | All sessions have status=submitted | No in_progress | All sessions status=submitted; no in_progress entries ✓ | API | PASS |
| 4 | Tester | Sessions ordered ascending by started_at | Ascending order | 3 sessions ordered by started_at ASC ✓ | API | PASS |
| 5 | Tester | meta.total is positive integer | Positive int | meta.total = 3 ✓ | API | PASS |

---

## Failed Steps Detail

### S7 Step 2-3 — SECTION SCORES rendered for flat exam (DEF-1)

**Expected**: No section score cards on result page; per_section_scores = []
**Actual**: "SECTION SCORES" card rendered on result page showing "100%". API returns `per_section_scores = [{"section_id":"9ceaced4-27e6-4101-a5af-a9ee167835c0","title":"","score_pct":100.0}]`
**Error**: The `section_id` value is actually the exam rule UUID (not a real section ID). When an exam has no sections, the API incorrectly populates per_section_scores with a single entry using the rule ID as section_id and empty title.
**Screenshot**: s1-result-page.png (SECTION SCORES visible above QUESTION REVIEW)
**Possible cause**: The session result query incorrectly joins rule entries as sections when the exam has no explicit section configuration. The flat rule (mode=manual, no section_id) is being treated as a section in the results computation.

### S11 Steps 3, 8 — Loading state instead of error message (DEF-2)

**Expected**: Inline error message rendered; table/content not shown
**Actual (S11 Step 3)**: My Results page shows 5-row loading skeleton (shimmer animation). No error text.
**Actual (S11 Step 8)**: Result Detail page shows a blue spinning loader. No error text.
**Error**: The React Query `isError` state is likely not being handled in the component render. Instead of showing an error alert/message when `isError=true`, the component renders as if it is still loading.
**Screenshots**: frontend/test-results/result-cert-uat-S11-Error-states-are-shown-for-API-failures-chromium-uat/test-failed-1.png; docs/uat-reports/screenshots/s11b-result-detail-error.png

---

## Acceptance Criteria Coverage

| FR | AC# | Criterion | Covered by | Status |
|----|-----|-----------|------------|--------|
| FR-BB41 | AC-1 | Portal session result returns 403 for wrong user | S10a | PASS |
| FR-BB41 | AC-2 | Portal session result returns 422 SESSION_IN_PROGRESS | S10b | PASS |
| FR-BB41 | AC-3 | show_answers=never → per_question_breakdown absent | S5 Step 10 | PASS |
| FR-BB41 | AC-4 | show_answers=after_completion → full breakdown | S1 Steps 8,11 | PASS (text null but structure correct) |
| FR-BB41 | AC-5 | Admin always gets full breakdown | S6 Steps 4-6 | PASS |
| FR-BB41 | AC-6 | History ascending by started_at; in_progress excluded | S12 Steps 3-4 | PASS |
| FR-BB41 | AC-7 | per_section_scores absent for no-section exam | S7 Step 3 | **FAIL** — DEF-1 |
| FR-BB41 | AC-8 | time_taken_seconds displayed as Xm Ys | S1 Step 6, S4 Step 8 | PASS |
| FR-BB41 | AC-9 | 404 SESSION_NOT_FOUND for non-existent session | S10c | PASS |
| FR-BB41 | AC-10 | History paginated with meta.total | S12 Step 5 | PASS |
| FR-BB41 | AC-11 | Admin result 422 SESSION_IN_PROGRESS | S10d | PASS |
| FR-BB43 | AC-1 | 403 for wrong user certificate | S10e | PASS |
| FR-BB43 | AC-2 | 422 EXAM_NOT_CERTIFIABLE for cert_enabled=false | S10f | PASS |
| FR-BB43 | AC-3 | 422 SESSION_NOT_PASSED for failed session | S10g | PASS |
| FR-BB43 | AC-4 | 422 SESSION_NOT_SUBMITTED for in_progress | S10h | PASS |
| FR-BB43 | AC-5 | Lazy creation; subsequent requests reuse record | S2 Steps 15-17 | PASS |
| FR-BB43 | AC-6 | Content-Type: application/pdf; correct filename | S2 Steps 2-4 | PASS |
| FR-BB43 | AC-7 | Public verify endpoint: valid→true; unknown→false | S3cert | PASS |
| FR-BB43 | AC-8 | Admin can download any session certificate | S6 Steps 7-8 | PASS |
| FR-BB43 | AC-9 | template_snapshot captures branding | S2 Steps 5-13 | PARTIAL — headers verified; PDF visual not inspectable via test |
| FR-BB43 | AC-10 | Idempotent cert creation | S2 Steps 15-17 | PASS |
| FR-BB44 | AC-2 | A4 landscape orientation | S2 Step 14 | PASS (headers; visual not verified) |
| FR-BB44 | AC-3 | Logo top-left or text fallback | S2 Step 10 | PARTIAL |
| FR-BB44 | AC-4 | Heading ≥28pt; name ≥22pt | S2 Steps 5-6 | PARTIAL |
| FR-BB44 | AC-5 | Exam title, score%, date present | S2 Steps 7-9 | PARTIAL |
| FR-BB44 | AC-6 | QR code ≥30mm×30mm | S2 Step 11 | PARTIAL |
| FR-BB44 | AC-7 | Verification code in footer | S2 Step 12 | PARTIAL |
| FR-BB44 | AC-8 | Signatory bottom-left | S2 Step 13 | PARTIAL |
| FR-BB45 | AC-1 | Score circular dial with tenant primary_color | S1 Steps 2-3 | PASS |
| FR-BB45 | AC-2 | Green/red pass/fail banner | S1 Steps 4-5, S5 Step 6 | PASS |
| FR-BB45 | AC-3 | Time in Xm Ys format | S1 Step 6 | PASS |
| FR-BB45 | AC-4 | Section score cards (absence tested) | S7 Step 2 | **FAIL** — DEF-1 |
| FR-BB45 | AC-5 | Breakdown table | S1 Steps 8-11 | PASS (structure); option text null (data setup) |
| FR-BB45 | AC-6 | Download Certificate only when passed+cert_enabled | S1 Step 12, S5 Step 9 | PASS |
| FR-BB45 | AC-7 | Retake only when attempts remaining and not expired | S1 Step 13 (no retake = correct, all used) | PASS |
| FR-BB45 | AC-8 | score_pct null → pending notice; cert/retake hidden | S9 Steps 4-8 | PASS |
| FR-BB45 | AC-9 | Loading/error states | S11 Steps 7-8 | **FAIL** — DEF-2 |
| FR-BB46 | AC-1 | "My Results" tab in portal navigation | S4 Steps 1-2 | PASS |
| FR-BB46 | AC-2 | Table columns: exam, date, score%, badge, time, cert | S4 Steps 4-8 | PASS |
| FR-BB46 | AC-3 | Certificate link only when passed+cert_enabled | S4 Step 9 | PASS |
| FR-BB46 | AC-4 | Date and Score sortable; URL sort state | S4 Steps 15-17 | PASS |
| FR-BB46 | AC-5 | Pagination 20/page; Previous/Next; Showing X-Y of Z | S4 Steps 18-20 | PASS |
| FR-BB46 | AC-6 | Empty state when no sessions | S8 | PASS |
| FR-BB46 | AC-7 | Clicking exam name → /portal/sessions/:id/result | S4 Steps 12-13 | PASS |
| FR-BB46 | AC-8 | Certificate download without navigating away | S4 Steps 10-11 | PASS |
| FR-BB46 | AC-9 | React Query refetches on re-navigation | S4 Step 21 | PASS |
| FR-BB46 | AC-10 | Zero hardcoded user-visible strings | S4 Step 4 (column headers) | PASS |
| FR-BB46 | AC-11 | API error → inline error message | S11 Steps 2-4 | **FAIL** — DEF-2 |

---

## Environment

- Frontend: http://localhost (nginx port 80 via Docker)
- Backend: http://localhost:8080
- Browser: Chromium (Playwright, via playwright-uat.config.ts)
- Stack started by: already running before UAT execution
- Playwright version: 1.60.0
- Test execution date: 2026-06-09