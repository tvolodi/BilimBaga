---
run_id: uat-employee-exam-taking-20260609
scenario_path: docs/uat-scenarios/employee-exam-taking-20260609.md
executed: 2026-06-09T12:43:00Z
executor: UAT Runner
result: PARTIAL (8 steps failed)
---

# UAT Report — Employee Exam Taking

## Summary
- Total steps: 26
- Passed: 18
- Failed: 8
- Blocked: 0
- Screenshots taken: 9

## Precondition Setup
| Precondition | Status | Notes |
|-------------|--------|-------|
| Platform running at localhost (port 80 via Nginx) | MET | Frontend served at http://localhost:80, not :5173 |
| Backend API at localhost:8080 | MET | HTTP 200 confirmed |
| Employee uat.employee@test.com / NewPass123! exists | MET | Login confirmed, role=employee |
| UAT Security Assessment — Active, configured correctly | MET (after UAT Runner setup) | Exam existed but required config fix: show_answers was "never" → updated to "after_completion"; on_tab_switch was "log" → updated to "warn"; certificate_enabled was false → set to true; no question rules → added random rule (3 questions, UAT Security category). Exam unpublished, updated, republished. |
| Exam assigned to uat.employee@test.com | MET | Verified via GET /api/v1/portal/exams |

---

## Scenario Results

### Scenario 1: Start Exam, Answer All Questions, Submit — Pass

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Employee | Log in as uat.employee@test.com / NewPass123! | Employee Portal visible | Portal loaded at /portal with "My Exams" heading | Playwright | PASS | none |
| 2 | Employee | Assert UAT Security Assessment card status + time limit | Card shows "Not started", "30 min" | Card: "Not started" badge, "30 min · Pass: 70% · Attempt 0 of 2", "Start exam" button | Playwright + Browser | PASS | screenshot-s1-step2.png |
| 3 | Employee | Click "Start exam" | Modal appears with time limit, passing score, tab-switch policy | Modal appeared; shows "You will have 30 minutes." and tab-switch policy warning. **Passing score (70%) NOT shown in modal.** | Playwright + Browser | FAIL | screenshot-s1-step3.png |
| 4 | Employee | Click "Begin exam" in modal | Exam screen loads; timer ~30:00; first question visible | Navigated to /portal/sessions/{id}; timer 29:59; Q5 "What is the principle of least privilege?" visible | Playwright | PASS | screenshot-s1-step4.png |
| 5 | Employee | Assert: exam title in top bar; progress "0/3 answered" | Title and progress counter correct | "UAT Security Assessm..." in banner; "0 / 3 answered" counter visible | Playwright | PASS | none |
| 6 | Employee | Assert: question navigator panel visible | Sidebar or collapsible panel with question numbers | "Questions" button in header opens panel with numbered buttons (1, 2, 3) and legend (Unanswered/Answered/Flagged) | Playwright + Browser | PASS | screenshot-s1-step6.png |
| 7 | Employee | Select first answer on Q1 | Answer highlighted; auto-save indicator shows "Saved" | Radio button selected + highlighted (blue border); progress updated to "1 / 3 answered". **No "Saved" text indicator visible.** | Playwright | FAIL | none |
| 8 | Employee | Navigate to Q2 via navigator; Q1 marked as answered | Q2 displayed; Q1 marked answered in navigator | All questions on single scrollable page; Q1 navigator button shows blue (answered) state | Playwright + Browser | PASS | screenshot-s1-step8.png |
| 9 | Employee | Select answer on Q2 | Answer highlighted; "Saved" indicator | Radio selected; progress "2 / 3 answered". **No "Saved" text indicator visible.** | Playwright | FAIL | none |
| 10 | Employee | Navigate to Q3, select answer; progress "3/3" | Saved; progress 3/3 | Radio selected; progress "3 / 3 answered". **No "Saved" text indicator visible.** | Playwright | FAIL | none |
| 11 | Employee | Click "Finish exam" | Review screen appears | "Review before submitting" screen appeared with all-answered confirmation | Playwright | PASS | screenshot-s1-step11.png |
| 12 | Employee | Assert no unanswered questions | Review shows all answered | "All questions answered. Ready to submit." displayed | Playwright | PASS | none |
| 13 | Employee | Click "Confirm and submit" | Session submitted; redirected to Result screen | Clicked "Submit anyway" → confirmation dialog "Submit exam" → "Submit" → result screen at /portal/sessions/{id}. **Button label is "Submit anyway" not "Confirm and submit".** | Playwright | PASS | none |
| 14 | Employee | Assert score % and Pass/Fail banner | Score and pass/fail visible | "✅ Passed, Score: 100%" displayed on result screen | Playwright + Browser | PASS | screenshot-s1-step14.png |
| 15 | Employee | Assert per-question review table visible (show_answers=after_completion) | Review table with question stems, answers, correct answers | **Basic result screen shown: only ✅ Passed + Score + "Back to my exams". No review table.** Per-question review IS accessible at /portal/sessions/{id}/result but user is NOT automatically redirected there after submission. | Playwright + Browser | FAIL | screenshot-s1-step15-basic.png screenshot-s1-step15-full-result.png |

---

### Scenario 2: Auto-Save Survives Browser Reload

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Employee | Start second attempt of UAT Security Assessment | Exam-taking screen visible | Session 2 started via API (POST /portal/exams/{id}/sessions); navigated to session URL; 3 questions loaded, timer 29:50 | Playwright (API) | PASS | none |
| 2 | Employee | Answer question 1 | Answer saved; indicator shows "Saved" | Radio selected; progress "1 / 3 answered". **No "Saved" text indicator.** | Playwright | PASS* | none |
| 3 | Employee | Navigate away without submitting | Tab closed / navigation away | Navigated to /login (simulating leaving the exam) | Playwright | PASS | none |
| 4 | Employee | Reopen http://localhost:5173 and log in | Employee Portal visible | Logged in; portal loaded at /portal | Playwright | PASS | none |
| 5 | Employee | Assert card shows "In progress" with "Continue" button | "In progress" status + "Continue" button | **Card shows "Passed" status (from attempt 1) + "View result" button. API confirms open_session_id exists but UI ignores it.** | Playwright + Browser | FAIL | screenshot-s2-step5.png |
| 6 | Employee | Click "Continue" | Exam reloads; Q1 answer preserved; timer < 30:00 | **"Continue" button absent (Step 5 failure).** Direct navigation to session URL shows: Q1 answer IS preserved ("Correct answer" checked), timer 28:28 (< 30:00). Auto-save backend behavior works but UI flow is broken. | Playwright (direct nav) | FAIL | none |

*Step 2 marked PASS because underlying functionality (answer saved to server) is confirmed by Step 6 behavior; "Saved" text indicator is a separate issue tracked in Scenario 1.

---

### Scenario 3: Tab Switch Warning

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Employee | Start exam (in-progress session) | Exam-taking screen visible | Session at /portal/sessions/{id} active with all questions | Playwright | PASS | none |
| 2 | Employee | Switch to another browser tab | Warning modal appears on return, informing about tab switch | Simulated visibilitychange event (hidden → visible). **"⚠️ Warning" modal appeared**: "You left the exam window. Further violations may result in automatic submission." | Playwright | PASS | screenshot-s3-step2.png |
| 3 | Employee | Dismiss the warning modal | Exam continues normally | Clicked "Cancel"; modal closed; exam screen with timer still running | Playwright | PASS | none |

---

### Scenario 4: Attempts Exhausted After Using Both Attempts

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Employee | Complete and submit both attempts | Both attempts used | Attempt 2 submitted via full flow (answered Q1-Q3, Finish → Submit anyway → Submit → Passed). Both attempts consumed. | Playwright | PASS | none |
| 2 | Employee | Navigate to Employee Portal | Card shows "Attempts exhausted" (or similar); Start button absent | Card shows **"Passed" status badge** (not "Attempts exhausted") with "Attempt 2 of 2" and **"View result" button. "Start exam" button IS correctly absent.** | Playwright + Browser | FAIL | screenshot-s4-step2.png |

---

## Failed Steps Detail

### Step 3 (Scenario 1) — Confirmation modal missing passing score
**Expected**: Modal shows time limit, passing score, and tab-switch policy  
**Actual**: Modal shows time limit ("You will have 30 minutes."), attempts remaining ("You have 2 attempt(s) remaining."), and tab-switch warning (highlighted in orange: "Once started, do not close or switch tabs..."). **Passing score (70%) is NOT shown.**  
**Possible cause**: Modal component was not designed to surface passing_score_pct from exam config.

### Steps 7, 9, 10 (Scenario 1) — Auto-save "Saved" indicator missing
**Expected**: After selecting an answer, an explicit "Saved" indicator appears within 2 seconds  
**Actual**: Answer is visually selected (radio highlighted); progress counter updates (e.g., "1 / 3 answered"); navigator button turns blue. However, **no explicit "Saved" text or toast notification** is shown.  
**Possible cause**: The implementation relies on the progress counter as the implicit confirmation. An explicit "Saved" indicator was not implemented or is a very brief transient state that wasn't captured.

### Step 15 (Scenario 1) — Per-question review not shown after submission
**Expected**: After submitting, employee is redirected to Result screen showing per-question review table (question stems, employee answers, correct answers) since show_answers = "after_completion"  
**Actual**: After submission the employee sees a basic screen (`/portal/sessions/{id}`) showing only "✅ Passed, Score: 100%, Back to my exams". The detailed result with question review IS accessible at `/portal/sessions/{id}/result` but the post-submit redirect goes to the basic screen, not the full result page.  
**Note**: The full result page at /result does correctly show the Question Review table respecting show_answers="after_completion".  
**Possible cause**: Post-submission redirect sends user to the session summary page, not the detailed result page. The "Back to my exams" button and absence of a "View full result" link on the basic screen leaves the review table undiscoverable.

### Scenario 2 Steps 5 & 6 — Portal card doesn't show "In progress" for active second attempt
**Expected**: After navigating away from an in-progress second attempt and returning to portal, card shows "In progress" status with "Continue" button  
**Actual**: Portal card shows "Passed" (from attempt 1) with "View result" button. API response confirms `open_session_id` is set but `user_status` is "passed". The frontend renders the card based on `user_status` without checking `open_session_id` first.  
**API data**: `{"user_status":"passed","open_session_id":"1e8f6b96-c5f4-4a3e-94e2-89d228bd09ea"}`  
**Possible cause**: Frontend portal card logic: when `user_status` is "passed", it always shows "View result" even if an open session exists for a subsequent attempt. Priority check (`open_session_id != null` → show "In progress"/"Continue") is missing.  
**Additional defect**: "Retake Exam" button on result page navigates to `/portal/exams/{id}` which matches no route (blank page rendered, console warning: "No routes matched location").

### Scenario 4 Step 2 — "Attempts exhausted" message not shown
**Expected**: Card shows "Attempts exhausted" or similar with Start button absent  
**Actual**: Card shows "Passed" badge with "Attempt 2 of 2" and "View result" button. **No "Attempts exhausted" message shown.** The Start button IS correctly absent.  
**Possible cause**: When all attempts are used AND the status is "passed", the card prioritizes showing the "Passed" status over an "Attempts exhausted" warning. For a failed-all-attempts scenario, the behavior may differ (untested here).

---

## Acceptance Criteria Coverage

| AC# | Criterion | Steps | Status |
|-----|-----------|-------|--------|
| 1 | Employee can start assigned exam and see all questions | S1 Steps 1–6 | PASS (exam started, questions visible) |
| 2 | Selecting an answer saves automatically; "Saved" indicator visible within 2 seconds | S1 Steps 7, 9, 10 | FAIL — save works (progress updates) but no explicit "Saved" indicator |
| 3 | Closing and reopening browser restores answers and remaining time | S2 Steps 1–6 | PARTIAL — backend auto-save works (answer + timer preserved), but portal card doesn't show "In progress"/"Continue" |
| 4 | Timer reaches zero → auto-submit (not tested — requires 30-min wait) | N/A | NOT COVERED |
| 5 | Tab switch on auto-submit exam ends session (not applicable — "Warn" mode used) | N/A | NOT COVERED |
| 6 | After submitting objective-only exam, employee immediately sees score and pass/fail | S1 Steps 13–14 | PASS — score and pass/fail visible, but on basic screen not full result page |
| 7 | Short-text exam shows "Grading in progress" | N/A | NOT COVERED |
| 8 | Attempts exhausted: card shows message; Start button absent | S4 Step 2 | PARTIAL — Start button absent ✓, "Attempts exhausted" message absent ✗ |

---

## Environment
- Frontend: http://localhost (port 80, via Nginx container)
- Backend: http://localhost:8080
- Browser: Chromium (Playwright via run_playwright_code tool)
- Stack started by: already running (Docker Compose)
- Precondition setup performed by UAT Runner (exam config updated via Admin API before execution)

## Defect Index
| # | Defect | Severity | Scenario | Step |
|---|--------|----------|----------|------|
| D1 | Confirmation modal missing passing score | Medium | 1 | 3 |
| D2 | Auto-save "Saved" indicator not shown after answer selection | Medium | 1 | 7, 9, 10 |
| D3 | Post-submit redirect to basic screen, not detailed result page | High | 1 | 15 |
| D4 | Portal card shows wrong status when in-progress second attempt exists | High | 2 | 5, 6 |
| D5 | "Retake Exam" button on result page navigates to non-existent route | High | 2 | — |
| D6 | "Attempts exhausted" message not displayed when all attempts consumed | Low | 4 | 2 |
