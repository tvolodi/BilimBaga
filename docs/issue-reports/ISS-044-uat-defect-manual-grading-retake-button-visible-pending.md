---
id: ISS-044
title: "UAT Defect: Manual Grading — Retake Exam button visible while session is grading_pending"
status: resolved
severity: medium
layer: frontend
module: grading
tags: [uat, manual-grading, grading_pending, retake]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: [ISS-043]
regression_test: null
---

## Symptom
UAT Scenario: Scenario 1, Step 11
Actor: Employee
Action: Employee views the result page for a session whose status is `grading_pending`.
Expected: No "Retake Exam" button is shown — retake must be suppressed while the current session is awaiting manual grading.
Actual: "Retake Exam" button is visible. Frontend only checks `attempts_used < max_attempts` to decide retake eligibility and does not suppress the button when the active session status is `grading_pending`.
Screenshot: none

## Root Cause
The retake button suppression relied on `canRetake={isPending ? false : canRetake}` in `ResultPage.tsx`. Because `isPending` was computed only as `result.score_pct === null` (see ISS-043), a `grading_pending` session with `score_pct = 0` set `isPending = false`, meaning `canRetake` was evaluated and the button became visible.

## Fix Applied
Fix applied as part of ISS-043: updating `isPending` to `result.score_pct === null || result.status === 'grading_pending'` ensures that the existing `canRetake={isPending ? false : canRetake}` guard correctly hides the retake button for all `grading_pending` sessions.

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/pages/ResultPage.tsx` | `isPending` now includes `result.status === 'grading_pending'` (shared fix with ISS-043) |
| `frontend/src/api/sessions.ts` | `status` field added to `SessionResult` type (shared fix with ISS-043) |
| `backend/internal/sessions/model.go` | `Status` field added to `SessionResultResponse` (shared fix with ISS-043) |
| `backend/internal/sessions/service.go` | `Status` propagated in `buildSessionResult` (shared fix with ISS-043) |

## Regression Test
None added (covered by ISS-043 fix and build + type-check).

## Resolution Results
- Tests: 254 passed, 0 failed
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
