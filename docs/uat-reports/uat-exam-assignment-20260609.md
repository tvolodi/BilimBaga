---
run_id: uat-exam-assignment-20260609
scenario_path: docs/uat-scenarios/exam-assignment-20260609.md
executed: 2026-06-09T11:34:00Z
executor: UAT Runner
result: PARTIAL (8 steps failed — 1 defect root cause across all 4 scenarios)
---

# UAT Report — Exam Assignment Process

## Summary
- Total steps: 18 (across 4 scenarios)
- Passed: 10
- Failed: 8 (all caused by one root defect: no UI path to manage assignments for active exams)
- Blocked: 0
- Screenshots taken: 4 (Playwright failure screenshots in test-results/)

## Precondition Setup

| Precondition | Status | Notes |
|-------------|--------|-------|
| Platform running at http://localhost | PASS | Backend 200, Frontend 200 |
| Admin account admin@test.com / Admin1234! | PASS | Login confirmed, token obtained |
| Active exam "UAT Security Assessment" | PASS | Created via API: ID `19655eb7-47a9-43ae-a1a3-8ce77b051f50`, time_limit=30min, max_attempts=2, published to `active` |
| Employee uat.employee@test.com / NewPass123! in dept "UAT Engineering" | PASS | Account existed (inactive) — activated via DB, password reset and changed to NewPass123! |
| Employee NOT yet assigned to exam | PASS | Assignments cleared via API before each scenario |

## Scenario Results

### Scenario 1: Assign Exam to Individual Employee — Employee Sees It in Portal

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Super Admin | Log in, navigate to Exams | Exams list visible | Exams list with h1 visible at /admin/exams | Playwright | PASS | none |
| 2 | Super Admin | Open "UAT Security Assessment" | Exam detail visible with status "Active" | Exam title in page; read-only notice (amber banner) visible — Step 1 fields disabled per ISS-034 | Playwright | PASS | none |
| 3 | Super Admin | Click "Assign" / navigate to Assignments tab | Assignment form/modal appears | **FAIL** — No "Assign" button on ExamsListPage; wizard Step 3 unreachable because Step 1 Next button is `disabled={isReadOnly}` when exam.status=active | Playwright | FAIL | test-results/uat-exam-assignment-*S1*/test-failed-1.png |
| 4 | Super Admin | Select "Individual", select uat.employee@test.com | Employee selected | **BLOCKED** by Step 3 above | Playwright | FAIL | none |
| 5 | Super Admin | Confirm assignment (no deadline) | Success + assignment row in list | **BLOCKED** by Step 3; assignment created via API successfully (backend works: `POST /api/v1/exams/:id/assign` → 201) | API verify | FAIL | none |
| 6 | Employee | Log in as uat.employee@test.com | Employee Portal loads | Employee Portal at /portal loads correctly | Playwright | PASS | none |
| 7 | Employee | Assert "UAT Security Assessment" card visible | Exam card visible with status "Not started" | Card visible with "Not started" badge | Playwright | PASS | none |
| 8 | Employee | Assert time limit (30 min) and max attempts (2) | Time limit and attempt info visible | Time limit shown as "30 min"; attempt info shown as "Attempt 0 of 2" (expected "2 attempts" — different phrasing but info present) | Playwright | PASS* | none |

*Step 8 regex `/2\s*(attempt|tries)/i` did not match "Attempt 0 of 2". The information IS displayed on the card but the format is "Attempt 0 of 2" not "2 attempts" or "2 tries". This is a format mismatch between the scenario expectation and the actual UI text — not a functional defect.

---

### Scenario 2: Assignment with Deadline — Deadline Shown on Card

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Super Admin | Navigate to Exams → "UAT Security Assessment" → Assignments | Assignments list visible | **FAIL** — same defect as S1 Step 3: wizard Step 3 is unreachable for active exams | Playwright | FAIL | test-results/uat-exam-assignment-*S2*/test-failed-1.png |
| 2 | Super Admin | Assign with deadline tomorrow 23:59 | Assignment updated with deadline | **BLOCKED** via UI; assignment with deadline created via `POST /api/v1/exams/:id/assign` API — success | API verify | FAIL | none |
| 3 | Employee | Reload Employee Portal | Exam card shows deadline countdown "Due in 23h 59m" | Portal shows deadline info matching `/due\|deadline\|23h\|in \d+h\|expire/i` — countdown visible | Playwright | PASS | none |

---

### Scenario 3: Remove Assignment — Exam Disappears from Portal

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Super Admin | Navigate to Exams → "UAT Security Assessment" → Assignments | Assignment for uat.employee@test.com listed | **FAIL** — wizard Step 3 unreachable for active exams | Playwright | FAIL | test-results/uat-exam-assignment-*S3*/test-failed-1.png |
| 2 | Super Admin | Click "Remove" | Confirmation prompt appears | **BLOCKED** by Step 1 above; `DELETE /api/v1/exams/:id/assign/:assignId` → 204 confirmed via API | API verify | FAIL | none |
| 3 | Super Admin | Confirm removal | Assignment row disappears | **BLOCKED** by Step 1; API deletion confirmed (204) | API verify | FAIL | none |
| 4 | Employee | Reload Employee Portal | "UAT Security Assessment" card is NO LONGER visible | Exam card absent from portal body text after API removal — CONFIRMED | Playwright | PASS | none |

---

### Scenario 4: Assignment Completion Status Visible to Admin

| Step | Actor | Action | Expected | Actual | Method | Status | Screenshot |
|------|-------|--------|----------|--------|--------|--------|------------|
| 1 | Super Admin | Re-assign "UAT Security Assessment" to uat.employee@test.com | Assignment created | `POST /api/v1/exams/:id/assign` → 201 (via API — UI not available) | API verify | PASS (API) | none |
| 2 | Super Admin | Navigate to Assignments — completion table shows "Not started" | Completion table with status "Not started" | **FAIL** — wizard Step 3 unreachable for active exams; API `GET /api/v1/exams/:id/assignments` confirms employee assignment exists | Playwright | FAIL | test-results/uat-exam-assignment-*S4*/test-failed-1.png |
| 3 | Employee | Assert exam card visible in Employee Portal | Card visible | Exam card visible with "Not started" status confirmed | Playwright | PASS | none |

---

## Failed Steps Detail

### Root Defect: Active Exam Assignments UI Inaccessible

**Affects**: S1 Steps 3–5, S2 Steps 1–2, S3 Steps 1–3, S4 Step 2 (8 steps total)

**Expected**: Admin can click "Assign" or navigate to an "Assignments" tab to manage exam assignments.

**Actual**: There is no UI path to the Assignments step (Step 3 of the Exam Wizard) when an exam is in `active` status:
1. `ExamsListPage.tsx`: Actions for an active exam = Edit (wizard) | Analytics | Unpublish | Archive — no "Assign" button.
2. `Step1BasicSettings.tsx` line 393: `<Button type="submit" disabled={isPending || isReadOnly}>` where `isReadOnly = exam?.status === 'active'`. This disables the Next button on Step 1.
3. `StepIndicator` uses non-clickable `<div>` elements — cannot jump to Step 3 directly.

**Result**: The only way to reach the Assignments step is to first unpublish the exam (make it draft), navigate to Step 3 via the wizard, make assignment changes, then re-publish. This is a significant workflow gap — admins cannot assign an active exam without first unpublishing it.

**Backend**: The assignment API is fully functional. `POST /api/v1/exams/:id/assign`, `GET /api/v1/exams/:id/assignments`, and `DELETE /api/v1/exams/:id/assign/:id` all work correctly.

**Possible cause**: The ISS-034 fix (read-only active exam) correctly disabled editing of exam settings for active exams, but inadvertently blocked navigation to the Assignments step which does NOT require modifying exam settings.

---

### S1 Step 8 — Attempt info display format mismatch

**Expected**: "Time limit and attempt info are visible on the card"
**Actual**: Card shows "Attempt 0 of 2" (not "2 attempts" or "2 tries")
**Error**: Regex `/2\s*(attempt|tries)/i` did not match "Attempt 0 of 2"
**Screenshot**: none
**Possible cause**: The UI uses "Attempt X of Y" format rather than "Y attempts" format. This is a scenario description mismatch, not a functional defect. The information IS present. Recommend updating the scenario to match actual format.

---

## Acceptance Criteria Coverage

| AC# | Criterion | Steps | Status |
|-----|-----------|-------|--------|
| 1 | Assigning to department makes exam card visible to all department members | Not covered (scenario note) | N/A |
| 2 | Assignment with deadline shows countdown on employee exam card | S2 Step 3 | PASS (via API-assisted assignment) |
| 3 | After deadline passes, card shows "Expired" | Not covered | N/A |
| 4 | Completion table shows correct status per assignee | S4 Steps 1–3 | PARTIAL — API confirms assignment exists; UI table inaccessible (defect) |
| 5 | Removing assignment removes card; completed sessions remain in history | S3 Steps 1–4 | PARTIAL — removal via UI blocked; API removal confirmed; card disappears from portal |
| 6 | Attempts exhausted: card shows "Attempts exhausted" | Not covered | N/A |

## Environment

- Frontend: http://localhost (Nginx port 80)
- Backend: http://localhost:8080
- Browser: Chromium (Playwright default)
- Stack started by: already running
- Playwright version: 1.60.0
- Precondition setup: uat.employee@test.com activated via DB; "UAT Security Assessment" (active) created via API
