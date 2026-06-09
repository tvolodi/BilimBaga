---
id: ISS-043
title: "UAT Defect: Manual Grading — Result page shows 0%/Failed for grading_pending session"
status: resolved
severity: high
layer: frontend
module: grading
tags: [uat, manual-grading, grading_pending, status, score_pct]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: [ISS-044]
regression_test: null
---

## Symptom
UAT Scenario: Scenario 1, Step 9
Actor: Employee
Action: Employee completes and submits an exam that contains short-text questions requiring manual grading; navigates to the result page.
Expected: Page shows an "Awaiting review" message with no numeric score and no Pass/Fail banner, because the session `status` is `grading_pending`.
Actual: Result page renders "0% / Failed" — it auto-scores the session as 0 instead of detecting the `grading_pending` status and showing a pending-review state.
Screenshot: none

## Root Cause
The backend `SessionResultResponse` lacked a `status` field. When a session is `grading_pending`, short-text questions are not yet auto-graded so `score_pct` may be 0 (not null), causing the frontend `isPending` check (`result.score_pct === null`) to evaluate false and render the "0% / Failed" score dial and pass/fail banner instead of the pending-review notice.

## Fix Applied
1. Added `Status string` to `SessionResultResponse` in `backend/internal/sessions/model.go`.
2. Populated `Status: row.Status` in `buildSessionResult` in `backend/internal/sessions/service.go`.
3. Added `status: string` to the frontend `SessionResult` type in `frontend/src/api/sessions.ts`.
4. Updated `isPending` in `frontend/src/pages/ResultPage.tsx` to `result.score_pct === null || result.status === 'grading_pending'`, so the pending-review notice is always shown for `grading_pending` sessions regardless of `score_pct`.

## Files Changed
| File | Change |
|------|--------|
| `backend/internal/sessions/model.go` | Added `Status string` field to `SessionResultResponse` |
| `backend/internal/sessions/service.go` | Propagate `row.Status` into response in `buildSessionResult` |
| `frontend/src/api/sessions.ts` | Added `status: string` to `SessionResult` interface |
| `frontend/src/pages/ResultPage.tsx` | Updated `isPending` to check `status === 'grading_pending'` |

## Regression Test
None added (covered by manual UAT verification and build + type-check).

## Resolution Results
- Tests: 254 passed, 0 failed
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|
