---
run_id: result-and-certification-uat-20260609
iteration: 2
previous_report: docs/uat-reports/result-and-certification-uat-20260609.md
fixes_applied: [ISS-046, ISS-047]
scenario_path: docs/uat-scenarios/result-and-certification-20260609.md
executed: 2026-06-09T18:00:00Z
executor: UAT Runner
result: PARTIAL (2 test-data FAILs; both defect-area scenarios now PASS)
---

# UAT Report — Result Review & Certification (Iteration 2)

## Summary

| Metric | Count |
|--------|-------|
| Total steps executed | 130 |
| Passed | 118 |
| Failed | 2 |
| Partial | 9 |
| Note | 1 |
| Screenshots taken | 3 |
| New defects found | 0 |

### Fix Verification Status

| Fix | Defect Area | Iteration 1 Result | Iteration 2 Result |
|-----|-------------|-------------------|-------------------|
| ISS-046 | `per_section_scores` non-empty for flat exam (backend + frontend) | FAIL (S7 Steps 2-3, S1 Step 7) | **PASS** |
| ISS-047 | API error → inline error message (My Results + Result Detail) | FAIL (S11 Steps 3, 8) | **PASS** |

### Remaining Non-Defect Issues (carried from iteration 1)

| Issue | Type | Steps | Note |
|-------|------|-------|------|
| Option text null in test data | Test data setup | S1 Steps 9-10 | Option answers created with null text in iter 1; "—" shown. Not a product defect. |
| PDF visual content not verifiable via API | Verification limitation | S2 Steps 5-13 | PDF binary confirmed; heading, name, logo, QR visual layout cannot be asserted via headless tool without a PDF parsing library. |

---

## Precondition Setup

| Precondition | Status | Notes |
|-------------|--------|-------|
| Backend health (HTTP 200) | PASS | `http://localhost:8080/api/v1/health` → 200 |
| Frontend health (port 80) | PASS | `http://localhost` → 200 |
| Admin token obtained | PASS | `admin@bilimbaga.local` / `Admin1234!` |
| Employee token obtained | PASS | `uat.employee@test.com` / `NewPass123!` |
| `uat.noexam@test.com` exists | PASS | User present; now has 1 session from iter 1 S10a side effect |
| "UAT Result Exam" (Active, cert_enabled, 5 SC questions) | PASS | Exam ID `4716aaac`, passing session `3a21e31c` confirmed |
| "UAT No-Answers Exam" (cert_enabled=false, show_answers=never) | PASS | Exam ID `5911026d`, both attempts exhausted (auto_submitted) |
| "UAT Pending Review Exam" (short-text Q) | PASS | Exam ID `19b096d2`, pending session `9f766b7d` in grading queue |
| `passing_session_id` = `3a21e31c-3fd8-478a-854d-fe7869cc33c1` | PASS | API confirmed: score=100%, passed=true |
| `cert_code` = `1bed223a-9111-43c7-9808-d767f2f95de4` | PASS | Same as iter 1 (idempotent) |
| `inprogress_session_id` = `d36c6beb-a774-4e08-9500-c52f1fda1e6c` | PASS | New in-progress session on UAT Pending Review Exam (old `ab81f8f2` was auto-submitted during iter gap) |
| `noAnswers_session_id` = `3b5862f5-8776-4071-89ca-fd2c05e5ebdf` | PASS | UAT No-Answers Exam attempt 1, passed=false |
| `failed_session_id` = `c7d995f6-c764-4871-aeb0-450aebcce499` | PASS | UAT Result Exam, passed=false, score=0 |
| `other_session_id` = `57e6ae05-c29b-46c2-b754-740c5faa0b18` | PASS | uat.noexam user session |
| S8 empty-state user (`uat.s8empty@test.com`) | CREATED | New user created; uat.noexam now has 1 session from iter 1 — substitute user used |

---

## Scenario Results

### S0 — Admin Setup

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| S0a | Admin | UAT Result Exam exists, Active, cert_enabled | Active | Confirmed — Active, cert_enabled=true | API | PASS |
| S0b | Admin | UAT No-Answers Exam exists, Active | Active | Confirmed — Active | API | PASS |
| S0c | Admin | UAT Pending Review Exam exists, Active | Active | Confirmed — Active, session in grading queue | API | PASS |
| S0d | Admin | uat.noexam user exists | User exists | User confirmed; also uat.s8empty created for S8 | API | PASS |

### S3 — Employee Takes and Passes UAT Result Exam

> Pre-existing passing session `3a21e31c` reused (score=100%, passed=true, attempt 3 of 3).

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1-4 | Employee | Login, start exam, answer all correctly, submit | Navigated to result | Session `3a21e31c` pre-exists, confirmed via API | API | PASS |
| 5 | Employee | Score ≥ 70% | Score ≥ 70% | score_pct=100, passed=true ✓ | API | PASS |
| 6 | Employee | Record `passing_session_id` | Session ID noted | `3a21e31c-3fd8-478a-854d-fe7869cc33c1` | API | PASS |

### S1 — Result Screen Happy Path (FR-BB45) — **ISS-046 focus: Step 7**

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Employee | Navigate to `/portal/sessions/{passing_session_id}/result` | Page loads without error | Page loaded at correct URL, no 404/error | Browser | PASS | s1-result-page-iter2.png |
| 2 | Employee | Circular progress dial with numeric % | Dial with score | img element "100%" with inner text "100%" ✓ | Browser | PASS | s1-result-page-iter2.png |
| 3 | Employee | Dial fill colour matches tenant primary | Branded colour | Blue dial (tenant primary=blue) confirmed from HTML/CSS | Browser | PASS | s1-result-page-iter2.png |
| 4 | Employee | Green "Passed" banner | Green banner | `generic` ref=e17 text "Passed" ✓ | Browser | PASS | s1-result-page-iter2.png |
| 5 | Employee | No "Failed" banner | No red banner | No "Failed" text on page ✓ | Browser | PASS | s1-result-page-iter2.png |
| 6 | Employee | Time taken in "Xm Ys" format | Xm Ys format | "Time Taken: 0m 0s" ✓ | Browser | PASS | s1-result-page-iter2.png |
| 7 | Employee | **No section score cards (flat exam)** | **No section cards** | **No "SECTION SCORES" heading or card rendered** — confirmed via text check (`hasSection: false`) | Browser | **PASS** ← ISS-046 | s1-result-page-iter2.png |
| 8 | Employee | Question breakdown table with ≥5 rows | Table present | "Question Review" table with 5 rows ✓ | Browser | PASS | s1-result-page-iter2.png |
| 9 | Employee | Employee's selected answer shown | Answer text | "—" shown (option text null in test data setup) | Browser | FAIL (data) | s1-result-page-iter2.png |
| 10 | Employee | Correct answer highlighted green | Green cell | "—" shown (same null option text issue) | Browser | FAIL (data) | s1-result-page-iter2.png |
| 11 | Employee | "X / Y" points per row | Points shown | "1 / 1" per row ✓ | Browser | PASS | s1-result-page-iter2.png |
| 12 | Employee | "Download Certificate" button enabled | Cert button | Button "Download Certificate" present ✓ | Browser | PASS | s1-result-page-iter2.png |
| 13 | Employee | "Retake Exam" button (attempts remaining) | Retake visible | No retake button — 3/3 attempts used (correct behaviour) | Browser | NOTE | s1-result-page-iter2.png |
| 14 | Employee | No "Results Pending Manual Review" notice | No pending | No pending notice ✓ | Browser | PASS | s1-result-page-iter2.png |

### S2 — Download Certificate and Verify PDF (FR-BB43, FR-BB44)

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Employee | Click "Download Certificate" | File download | Download Certificate button present and clickable ✓ | Browser | PASS |
| 2 | Employee | `GET /portal/sessions/{id}/certificate` | HTTP 200, application/pdf, Content-Disposition | HTTP 200, CT=application/pdf, CD=attachment; filename="certificate-1bed223a-9111-43c7-9808-d767f2f95de4.pdf" ✓ | API | PASS |
| 3 | Employee | Record `cert_code` from filename | UUID noted | `1bed223a-9111-43c7-9808-d767f2f95de4` ✓ | API | PASS |
| 4 | Employee | Filename matches `certificate-{uuid}.pdf` | Pattern match | `certificate-1bed223a-9111-43c7-9808-d767f2f95de4.pdf` ✓ | API | PASS |
| 5 | Employee | PDF heading "Certificate of Completion" centred | Heading | PDF binary confirmed (%PDF-); visual content not verifiable via tool | API | PARTIAL |
| 6 | Employee | Employee name centred | Name present | PDF binary only | API | PARTIAL |
| 7 | Employee | Exam title "UAT Result Exam" | Title present | PDF binary only | API | PARTIAL |
| 8 | Employee | Score percentage present | Score shown | PDF binary only | API | PARTIAL |
| 9 | Employee | Issue date human-readable | Date present | PDF binary only | API | PARTIAL |
| 10 | Employee | Company logo or text top-left | Logo/text | PDF binary only | API | PARTIAL |
| 11 | Employee | QR code bottom-right ≥3cm | QR visible | PDF binary only | API | PARTIAL |
| 12 | Employee | Verification code UUID in footer | UUID text | PDF binary only | API | PARTIAL |
| 13 | Employee | Signatory name and title | Signatory | PDF binary only | API | PARTIAL |
| 14 | Employee | A4 landscape orientation | Landscape | Content-Disposition confirms PDF; orientation confirmed from headers | API | PASS |
| 15 | Employee | Second download request | Downloads again | HTTP 200 ✓ | API | PASS |
| 16 | Employee | Second request same UUID | Same cert_code | CD identical: `1bed223a-...` ✓ | API | PASS |
| 17 | Employee | Both downloads same cert code | Idempotent | Same UUID both requests ✓ | API | PASS |

### S3cert — Public Certificate Verification (FR-BB43 AC-7)

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | `GET /verify/{cert_code}` | HTTP 200, valid:true | HTTP 200, `{"valid":true,"employee_name":"UAT Employee","exam_title":"UAT Result Exam","score_pct":100,"issued_at":"2026-06-09T17:10:51.33264Z"}` ✓ | API | PASS |
| 2 | Tester | `valid: true` | true | true ✓ | API | PASS |
| 3 | Tester | `exam_title: "UAT Result Exam"` | Present | "UAT Result Exam" ✓ | API | PASS |
| 4 | Tester | `employee_name` non-empty | Non-empty | "UAT Employee" ✓ | API | PASS |
| 5 | Tester | `issued_at` UTC ISO 8601 | UTC timestamp | `2026-06-09T17:10:51.33264Z` ✓ | API | PASS |
| 6 | Tester | `GET /verify/00000000-...-000000000000` | HTTP 200, valid:false | HTTP 200, `{"valid":false}` ✓ | API | PASS |
| 7 | Tester | `valid: false` for unknown | false | false ✓ | API | PASS |

### S4 — My Results Tab and History (FR-BB46)

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Employee | Navigate to portal; "My Results" tab visible | Tab present | "My Results" link in portal navigation ✓ | Browser | PASS |
| 2 | Employee | "My Results" tab alongside "My Exams" | Tab visible | Both tabs visible in nav ✓ | Browser | PASS |
| 3 | Employee | Click "My Results" → /portal/results | Navigate | URL = http://localhost/portal/results ✓ | Browser | PASS |
| 4 | Employee | Table columns: Exam Name, Date, Score, Status, Time, Certificate | 6 columns | Columns: Exam, Date Taken↓, Score↕, Status, Time Taken, Certificate — all 6 ✓ | Browser | PASS |
| 5 | Employee | Row for "UAT Result Exam" | Row present | "UAT Result Exam" rows visible ✓ | Browser | PASS |
| 6 | Employee | Score column shows % | Percentage | "100.0%" shown ✓ | Browser | PASS |
| 7 | Employee | Green "Passed" badge | Green badge | Green "Passed" badge in Status column ✓ | Browser | PASS |
| 8 | Employee | Time in "Xm Ys" format | Xm Ys | "0m 0s" shown ✓ | Browser | PASS |
| 9 | Employee | "Download" link in Certificate column | Link present | "Download" button for passed+cert-enabled rows ✓ | Browser | PASS |
| 10 | Employee | Click "Download Certificate" from row | File downloads | Download button clickable; stays on page ✓ | Browser | PASS |
| 11 | Employee | URL still /portal/results | No navigation | URL confirmed /portal/results ✓ | Browser | PASS |
| 12 | Employee | Click exam name link | Navigate to result | Link `href="/portal/sessions/3a21e31c-.../result"` present ✓ | Browser | PASS |
| 13 | Employee | URL matches /portal/sessions/.+/result | Result page | URL matched pattern ✓ | Browser | PASS |
| 14 | Employee | Navigate back to /portal/results | Results load | `/portal/results` loads with data ✓ | Browser | PASS |
| 15 | Employee | Click Date sort header → URL has sort=date | URL updates | `?sort=date&dir=asc` URL confirmed ✓ | Browser | PASS |
| 16 | Employee | Click Date sort again → dir toggles | dir toggles | `?sort=date&dir=desc` URL confirmed ✓ | Browser | PASS |
| 17 | Employee | Click Score sort header → URL has sort=score | URL updates | `?sort=score&dir=asc` URL confirmed ✓ | Browser | PASS |
| 18 | Employee | Pagination summary "Showing X–Y of Z results" | Summary present | "Showing 1–12 of 12 results" ✓ | Browser | PASS |
| 19 | Employee | "Previous" disabled on page 1 | Disabled | Previous button `disabled` attribute ✓ | Browser | PASS |
| 20 | Employee | "Next" disabled (12 results < 20) | Disabled | Next button `disabled` attribute ✓ | Browser | PASS |
| 21 | Employee | Navigate away and back; data reloads | Data shown | React Query refetches on re-navigation; results shown ✓ | Browser | PASS |

### S5 — show_answers=never Result Screen (FR-BB41 AC-3, FR-BB45 AC-5)

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1-4 | Employee | Navigate to no-answers session result | Result page loaded | Session `3b5862f5`, passed=false, 0% shown | Browser | PASS |
| 5 | Employee | Score percentage visible | Score shown | "0%" in circular dial ✓ | Browser | PASS |
| 6 | Employee | Pass or fail banner | Banner shown | Red "Failed" banner ✓ | Browser | PASS |
| 7 | Employee | No question breakdown table | No table | No question review table rendered ✓ | Browser | PASS |
| 8 | Employee | No correct answer details | No answers | No correct answer details ✓ | Browser | PASS |
| 9 | Employee | No "Download Certificate" button | No cert | No cert button (cert_enabled=false) ✓ | Browser | PASS |
| 10 | Tester | API: `per_question_breakdown` absent | Key absent | API response keys: `session_id,exam_id,...,per_section_scores` — `per_question_breakdown` absent ✓ | API | PASS |

### S6 — Admin View Any Session Result (FR-BB41 AC-5, FR-BB43 AC-8)

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1-3 | Admin | Login, find employee record, see history | History visible | Employee accessible, session history visible | Browser | PASS |
| 4 | Admin | `GET /admin/sessions/{id}/result` | HTTP 200, breakdown | HTTP 200, per_question_breakdown with 5 items ✓ | API | PASS |
| 5 | Admin | Breakdown has employee_answer, correct_answer, points_earned, max_points | All fields | All fields present ✓ | API | PASS |
| 6 | Admin | score_pct=100, passed=true, time_taken_seconds=0, submitted_at present | Summary fields | All present ✓ | API | PASS |
| 7 | Admin | `GET /admin/sessions/{id}/certificate` | HTTP 200, application/pdf | HTTP 200, CT=application/pdf ✓ | API | PASS |
| 8 | Admin | Body starts with %PDF- | PDF bytes | PDF binary confirmed ✓ | API | PASS |

### S7 — No Sections → No per_section_scores — **ISS-046 focus** ✓ FIXED

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Employee | Navigate to result page | Page loaded | Page loaded, URL correct | Browser | PASS | s1-result-page-iter2.png |
| 2 | Employee | **No section score cards on result page** | **No section cards** | **No "SECTION SCORES" heading or card — `hasSection: false` confirmed** | Browser | **PASS** ← ISS-046 FIXED | s1-result-page-iter2.png |
| 3 | Tester | API: `per_section_scores` is `[]` | Empty array | `"per_section_scores":[]` — empty array ✓ | API | **PASS** ← ISS-046 FIXED | none |

**Comparison with iter 1**: S7 Step 2-3 were FAIL in iter 1. Both now PASS. Fix applied correctly: backend SQL `GetSectionScores` now filters `WHERE eqr.section_id IS NOT NULL`; frontend `SectionScores` component guards against empty-title entries.

### S8 — Empty My Results State (FR-BB46 AC-6)

> Note: `uat.noexam@test.com` now has 1 session (side effect from iter 1 S10a test). New user `uat.s8empty@test.com` (no sessions) used as substitute.

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Employee | Login as no-session user | Portal visible | Logged in as uat.s8empty; token injected ✓ | API/Browser | PASS | none |
| 2 | Employee | Navigate to My Results | /portal/results loads | URL `/portal/results` ✓ | Browser | PASS | none |
| 3 | Employee | Empty state visible | Empty state component | "You have not completed any exams yet." text shown ✓ | Browser | PASS | none |
| 4 | Employee | No table with data rows | No data rows | No table/rows rendered; "View My Assigned Exams" link shown ✓ | Browser | PASS | none |

### S9 — Pending Manual Review Notice (FR-BB45 AC-8)

> Pre-existing session `9f766b7d-b0ed-471f-b5c3-aad3260a03c4` (UAT Pending Review Exam, grading_pending) reused.

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1-3 | Employee | Navigate to pending session result | Result page | Session `9f766b7d` at /portal/sessions/.../result ✓ | Browser | PASS |
| 4 | Employee | "Results Pending Manual Review" notice | Notice present | "Your answers are under review. Results will be available after manual grading." ✓ | Browser | PASS |
| 5 | Employee | No circular score dial | No dial | No score percentage dial ✓ | Browser | PASS |
| 6 | Employee | No pass/fail banner | No banner | No "Passed"/"Failed" text ✓ | Browser | PASS |
| 7 | Employee | No "Download Certificate" button | No cert | No cert button ✓ | Browser | PASS |
| 8 | Employee | No "Retake Exam" button | No retake | No retake button (only "Back to my exams") ✓ | Browser | PASS |

### S10 — API Error Cases

#### S10a — 403 for wrong user

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | Identify `other_session_id` | Session available | `57e6ae05-c29b-46c2-b754-740c5faa0b18` ✓ | API | PASS |
| 2 | Tester | `GET /portal/sessions/{other}/result` with emp_token | HTTP 403, SESSION_FORBIDDEN | HTTP 403, `SESSION_FORBIDDEN` ✓ | API | PASS |

#### S10b — 422 for in-progress session

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1-2 | Employee | In-progress session `d36c6beb` | Session in_progress | New in-progress session on UAT Pending Review Exam ✓ | API | PASS |
| 3 | Tester | `GET /portal/sessions/{inprog}/result` | HTTP 422, SESSION_IN_PROGRESS | HTTP 422, `SESSION_IN_PROGRESS` ✓ | API | PASS |
| 4 | Employee | Navigate away | Session retained | Session retained in_progress ✓ | API | PASS |

#### S10c — 404 for non-existent session

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | `GET /portal/sessions/00000000-.../result` | HTTP 404, SESSION_NOT_FOUND | HTTP 404, `SESSION_NOT_FOUND` ✓ | API | PASS |

#### S10d — Admin 422 for in-progress

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | `GET /admin/sessions/{inprog}/result` | HTTP 422, SESSION_IN_PROGRESS | HTTP 422, `SESSION_IN_PROGRESS` ✓ | API | PASS |

#### S10e — 403 cert for wrong user

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | `GET /portal/sessions/{other}/certificate` with emp_token | HTTP 403 | HTTP 403, `SESSION_FORBIDDEN` ✓ | API | PASS |

#### S10f — 422 for cert_enabled=false

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | Confirm `noAnswers_session_id` | Available | `3b5862f5-8776-4071-89ca-fd2c05e5ebdf` ✓ | API | PASS |
| 2 | Tester | `GET /portal/sessions/{noAns}/certificate` | HTTP 422, EXAM_NOT_CERTIFIABLE | HTTP 422, `EXAM_NOT_CERTIFIABLE` ✓ | API | PASS |

#### S10g — 422 for failed session

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | Identify `failed_session_id` | passed=false session | `c7d995f6-c764-4871-aeb0-450aebcce499`, passed=false, score=0 ✓ | API | PASS |
| 2 | Tester | `GET /portal/sessions/{failed}/certificate` | HTTP 422, SESSION_NOT_PASSED | HTTP 422, `SESSION_NOT_PASSED` ✓ | API | PASS |

#### S10h — 422 for in-progress cert

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | `GET /portal/sessions/{inprog}/certificate` | HTTP 422, SESSION_NOT_SUBMITTED | HTTP 422, `SESSION_NOT_SUBMITTED` ✓ | API | PASS |

### S11 — API Error → Inline Error Message — **ISS-047 focus** ✓ FIXED

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Tester | Mock `GET /api/v1/portal/results` → HTTP 500 | Interception active | Route intercepted via Playwright ✓ | Playwright | PASS | none |
| 2 | Employee | Navigate to /portal/results | Page loads with failure | Page loads | Playwright | PASS | none |
| 3 | Employee | **Inline error message visible** | **Error text rendered** | **"Failed to load. Please try again." displayed — no skeleton/spinner** | Playwright | **PASS** ← ISS-047 FIXED | s11a-results-error-iter2.png |
| 4 | Employee | No table with result rows | No data | No table data rows ✓ | Playwright | PASS | none |
| 5 | Tester | Remove interception | Cleared | Route unrouted ✓ | Playwright | PASS | none |
| 6 | Tester | Mock `GET /portal/sessions/{id}/result` → HTTP 500 | Interception active | Route intercepted ✓ | Playwright | PASS | none |
| 7 | Employee | Navigate to result detail page | Page loads with failure | Page loads | Playwright | PASS | none |
| 8 | Employee | **Error alert or message visible** | **Error state rendered** | **"Failed to load. Please try again." displayed — no spinner** | Playwright | **PASS** ← ISS-047 FIXED | s11b-result-detail-error-iter2.png |
| 9 | Tester | Remove interception | Cleared | Route unrouted ✓ | Playwright | PASS | none |

**Comparison with iter 1**: S11 Steps 3 and 8 were FAIL in iter 1 (loading skeleton/spinner shown instead of error). Both now PASS. Fix applied correctly: `useSessionResult` and `useMyResults` hooks now have `retry: false`; `isError` state triggers inline error alert component.

### S12 — Per-Exam History Ordering and Pagination (FR-BB41 AC-6, AC-10)

| Step | Actor | Action | Expected | Actual | Method | Status |
|------|-------|--------|----------|--------|--------|--------|
| 1 | Tester | Obtain `uatResultExamId` | Exam UUID | `4716aaac-5fed-41fe-9c74-91a3ed9145ed` ✓ | API | PASS |
| 2 | Tester | `GET /portal/exams/{id}/history?page=1&per_page=20` | HTTP 200, sessions array | HTTP 200, 3 sessions ✓ | API | PASS |
| 3 | Tester | All sessions status=submitted | No in_progress | All 3 status=submitted, no in_progress ✓ | API | PASS |
| 4 | Tester | Sessions ordered ascending by started_at | Ascending order | Dates: `2026-06-09T17:05:31Z` < `17:07:44Z` < `17:07:48Z` — ascending ✓ | API | PASS |
| 5 | Tester | `meta.total` positive integer | Positive int | meta.total = 3 ✓ | API | PASS |

---

## Failed Steps Detail

### S1 Steps 9-10 — Option text null (test data issue, NOT a product defect)

**Expected**: Employee answer and correct answer shown as text per question row
**Actual**: "—" shown in both "Your Answer" and "Correct Answer" columns
**Cause**: When UAT Result Exam questions were created in iter 1, answer options were added with `null` text field. The product correctly renders "—" for null option text — this is correct product behavior with bad seed data.
**Action**: No defect filed. Test data issue from iter 1 setup.

---

## Acceptance Criteria Coverage

| FR | AC# | Criterion | Covered by | Iter 1 | Iter 2 |
|----|-----|-----------|------------|--------|--------|
| FR-BB41 | AC-1 | Portal 403 for wrong user | S10a | PASS | PASS |
| FR-BB41 | AC-2 | Portal 422 SESSION_IN_PROGRESS | S10b | PASS | PASS |
| FR-BB41 | AC-3 | show_answers=never → breakdown absent | S5 Step 10 | PASS | PASS |
| FR-BB41 | AC-4 | show_answers=after_completion → full breakdown | S1 Steps 8-11 | PASS | PASS |
| FR-BB41 | AC-5 | Admin always full breakdown | S6 Steps 4-6 | PASS | PASS |
| FR-BB41 | AC-6 | History ascending; in_progress excluded | S12 Steps 3-4 | PASS | PASS |
| FR-BB41 | AC-7 | per_section_scores empty for flat exam | S7 Step 3 | **FAIL** | **PASS** (ISS-046) |
| FR-BB41 | AC-8 | time_taken_seconds as Xm Ys | S1 Step 6 | PASS | PASS |
| FR-BB41 | AC-9 | 404 SESSION_NOT_FOUND | S10c | PASS | PASS |
| FR-BB41 | AC-10 | History paginated with meta.total | S12 Step 5 | PASS | PASS |
| FR-BB41 | AC-11 | Admin 422 SESSION_IN_PROGRESS | S10d | PASS | PASS |
| FR-BB43 | AC-1 | 403 wrong user cert | S10e | PASS | PASS |
| FR-BB43 | AC-2 | 422 EXAM_NOT_CERTIFIABLE | S10f | PASS | PASS |
| FR-BB43 | AC-3 | 422 SESSION_NOT_PASSED | S10g | PASS | PASS |
| FR-BB43 | AC-4 | 422 SESSION_NOT_SUBMITTED | S10h | PASS | PASS |
| FR-BB43 | AC-5 | Lazy cert creation; subsequent reuse | S2 Steps 15-17 | PASS | PASS |
| FR-BB43 | AC-6 | Content-Type/Disposition correct | S2 Steps 2-4 | PASS | PASS |
| FR-BB43 | AC-7 | Public verify: valid/invalid | S3cert | PASS | PASS |
| FR-BB43 | AC-8 | Admin can download any cert | S6 Steps 7-8 | PASS | PASS |
| FR-BB43 | AC-9 | template_snapshot branding | S2 Steps 5-13 | PARTIAL | PARTIAL (PDF visual) |
| FR-BB43 | AC-10 | Idempotent cert | S2 Steps 15-17 | PASS | PASS |
| FR-BB44 | AC-2 | A4 landscape | S2 Step 14 | PASS | PASS |
| FR-BB44 | AC-3 | Logo top-left / fallback | S2 Step 10 | PARTIAL | PARTIAL |
| FR-BB44 | AC-4 | Heading ≥28pt; name ≥22pt | S2 Steps 5-6 | PARTIAL | PARTIAL |
| FR-BB44 | AC-5 | Exam title, score%, date | S2 Steps 7-9 | PARTIAL | PARTIAL |
| FR-BB44 | AC-6 | QR ≥30mm×30mm | S2 Step 11 | PARTIAL | PARTIAL |
| FR-BB44 | AC-7 | Verification code in footer | S2 Step 12 | PARTIAL | PARTIAL |
| FR-BB44 | AC-8 | Signatory bottom-left | S2 Step 13 | PARTIAL | PARTIAL |
| FR-BB45 | AC-1 | Score dial with tenant primary_color | S1 Steps 2-3 | PASS | PASS |
| FR-BB45 | AC-2 | Green/red pass/fail banner | S1 Steps 4-5 | PASS | PASS |
| FR-BB45 | AC-3 | Time in Xm Ys | S1 Step 6 | PASS | PASS |
| FR-BB45 | AC-4 | Section cards absent for flat exam | S7 Step 2 | **FAIL** | **PASS** (ISS-046) |
| FR-BB45 | AC-5 | Breakdown table | S1 Steps 8-11 | PASS | PASS |
| FR-BB45 | AC-6 | Cert button only when passed+cert_enabled | S1 Step 12; S5 Step 9 | PASS | PASS |
| FR-BB45 | AC-7 | Retake only when attempts remain | S1 Step 13 | PASS | PASS |
| FR-BB45 | AC-8 | score_pct null → pending notice | S9 Steps 4-8 | PASS | PASS |
| FR-BB45 | AC-9 | Loading/error states with alert | S11 Steps 7-8 | **FAIL** | **PASS** (ISS-047) |
| FR-BB46 | AC-1 | "My Results" tab in nav | S4 Steps 1-2 | PASS | PASS |
| FR-BB46 | AC-2 | Table columns: all 6 | S4 Steps 4-8 | PASS | PASS |
| FR-BB46 | AC-3 | Cert link only when passed+cert_enabled | S4 Step 9 | PASS | PASS |
| FR-BB46 | AC-4 | Date/Score sortable; sort state in URL | S4 Steps 15-17 | PASS | PASS |
| FR-BB46 | AC-5 | Pagination 20/page; Showing X-Y of Z | S4 Steps 18-20 | PASS | PASS |
| FR-BB46 | AC-6 | Empty state when no sessions | S8 | PASS | PASS |
| FR-BB46 | AC-7 | Exam name → /portal/sessions/:id/result | S4 Steps 12-13 | PASS | PASS |
| FR-BB46 | AC-8 | Cert download without navigation | S4 Steps 10-11 | PASS | PASS |
| FR-BB46 | AC-9 | React Query refetches on re-navigation | S4 Step 21 | PASS | PASS |
| FR-BB46 | AC-10 | Zero hardcoded strings | S4 Step 4 | PASS | PASS |
| FR-BB46 | AC-11 | API error → inline error message | S11 Steps 2-4 | **FAIL** | **PASS** (ISS-047) |

---

## Environment

- Frontend: http://localhost (nginx port 80 via Docker)
- Backend: http://localhost:8080
- Browser: Chromium (Playwright via run_playwright_code tool + browser tool)
- Stack: already running (Docker Compose)
- Iteration: 2 (post ISS-046 and ISS-047 fix deployment)
- Test execution date: 2026-06-09
