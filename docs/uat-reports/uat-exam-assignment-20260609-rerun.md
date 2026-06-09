---
run-id: uat-exam-assignment-20260609-rerun
scenario-file: docs/uat-scenarios/exam-assignment-20260609.md
feature: exam-assignment (FR-BB33, FR-BB34, FR-BB61)
executed: 2026-06-09
result: ALL PASS
---

# UAT Report: Exam Assignment — Re-run (2026-06-09)

## Summary

| Field | Value |
|-------|-------|
| Run ID | uat-exam-assignment-20260609-rerun |
| Scenario file | docs/uat-scenarios/exam-assignment-20260609.md |
| Base URL | http://localhost (Nginx port 80) |
| Executed | 2026-06-09T12:07:57Z |
| Duration | 21.5 seconds |
| Result | **ALL PASS** |
| Scenarios | 4 / 4 passed |
| Steps | 18 / 18 passed |

---

## Context

This re-run was triggered after ISS-035 was resolved. ISS-035 identified that the Exam
Wizard blocked navigation to Step 3 (Assignments) for active exams, and the ExamsListPage
had no "Assign" shortcut for active exams. The fix added:
- An "Assign" button on the ExamsListPage for every active exam, linking to
  `/admin/exams/{id}/edit?step=3`.
- Wizard navigation to Step 3 for active exams without forcing step-1 validation.

Two additional defects were discovered and fixed during this UAT run:

**ISS-036 (discovered during run): 204 No Content JSON parse error**  
`examsFetch` in `frontend/src/api/exams.ts` always called `res.json()` even on `204 No Content`
responses. The DELETE `/exams/{id}/assign/{assignmentId}` endpoint returns `204` with no body.
This caused "Failed to execute 'json' on 'Response': Unexpected end of JSON input" and the
assignment row remained visible after clicking Remove. Fixed by skipping JSON parsing when
`res.status === 204`.

**Test infrastructure fix: Frontend image was stale**  
The running Docker container served a pre-built frontend image that did not include the
ISS-035 "Assign" button fix. The frontend image was rebuilt and the container restarted
before the final test run.

---

## Scenario Results

### S1: Assign Exam to Individual Employee — Employee Sees It in Portal

**Result: PASS**

| Step | Actor | Action | Expected | Actual |
|------|-------|--------|----------|--------|
| 1 | Super Admin | Log in, navigate to Exams | Exams list visible | PASS |
| 2 | Super Admin | Locate "UAT Security Assessment" row | Status Active visible | PASS |
| 3 | Super Admin | Click "Assign" button | Wizard opens at Step 3 | PASS — "Assign" link present in active exam row (ISS-035 fix confirmed) |
| 4 | Super Admin | Select scope "Individual", pick uat.employee@test.com | Employee selected | PASS |
| 5 | Super Admin | Leave deadline blank, Save | Assignment row appears | PASS |
| 6 | Employee | Log in as uat.employee@test.com | Employee Portal loads | PASS |
| 7 | Employee | Assert exam card visible | Card shows "Not started" | PASS |
| 8 | Employee | Assert card shows 30 min / 2 attempts | Time limit and attempt info visible | PASS |

---

### S2: Assignment with Deadline — Deadline Shown on Card

**Result: PASS**

| Step | Actor | Action | Expected | Actual |
|------|-------|--------|----------|--------|
| 1 | Super Admin | Navigate to Assignments via "Assign" button | Step 3 visible | PASS |
| 2 | Super Admin | Add assignment with deadline = tomorrow 23:59 | Assignment row with deadline | PASS — deadline input filled successfully |
| 3 | Employee | Load Employee Portal | Card shows deadline countdown "Due …" | PASS — `portal.card.deadline` renders "Due {{countdown}}" |

---

### S3: Remove Assignment — Exam Disappears from Portal

**Result: PASS**

| Step | Actor | Action | Expected | Actual |
|------|-------|--------|----------|--------|
| 1 | Super Admin | Navigate to Step 3 | Assignment for uat.employee visible | PASS |
| 2 | Super Admin | Click Remove (trash icon) | Row disappears | PASS — 204 fix confirmed working |
| 3 | Super Admin | Confirm row gone | No assignment row | PASS |
| 4 | Employee | Reload portal | Exam card not visible | PASS |

---

### S4: Assignment Completion Status Visible to Admin

**Result: PASS**

| Step | Actor | Action | Expected | Actual |
|------|-------|--------|----------|--------|
| 1 | Super Admin | Re-assign uat.employee@test.com | Assignment created | PASS |
| 2 | Super Admin | View Step 3 | Employee name visible in assignments list | PASS |
| 3 | Employee | Navigate to portal | Exam card visible | PASS |

---

## Defects Discovered and Resolved During Run

| ID | Title | Severity | Resolution |
|----|-------|----------|------------|
| ISS-036 | `examsFetch` crashes on 204 No Content (DELETE assignment) | High | Fixed in `frontend/src/api/exams.ts` — skip JSON parsing for 204 responses |

---

## Acceptance Criteria Coverage

| AC# | Criterion | Status |
|-----|-----------|--------|
| 2 | Assignment with deadline shows countdown on exam card | VERIFIED — Scenario 2, Step 3 |
| 4 | Completion table shows correct status per assignee | VERIFIED — Scenario 4, Steps 1–3 |
| 5 | Removing assignment removes card from portal | VERIFIED — Scenario 3, Steps 1–4 |
| ISS-035 | "Assign" button visible on ExamsListPage for active exams | VERIFIED — Scenario 1, Step 3 |
| ISS-036 | DELETE assignment (204 No Content) handled correctly | VERIFIED — Scenario 3 |

---

## UAT Decision

**PASS** — All 4 scenarios and 18 steps completed successfully. The exam assignment process
(FR-BB33, FR-BB34, FR-BB61) is verified end-to-end. ISS-035 fix is confirmed working.
The additionally discovered ISS-036 defect was resolved during this run.

Recommend marking the exam-assignment process as **uat-verified**.
