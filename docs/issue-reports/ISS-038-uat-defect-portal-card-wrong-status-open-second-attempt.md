---
id: ISS-038
title: "UAT Defect: Employee Exam Taking — Portal card shows 'Passed'/'View result' when an in-progress second attempt exists"
status: resolved
severity: high
layer: both
module: employee-portal
tags: [uat, employee-exam-taking, open_session_id, user_status, ExamCard]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: [ISS-039]
regression_test: frontend/src/pages/EmployeePortal/EmployeePortal.test.tsx
---

## Symptom
UAT Scenario: `Auto-Save Survives Browser Reload`, Step 5
Actor: Employee
Action: Navigate away from an in-progress second attempt (attempt 2 of 2) and return to Employee Portal; exam has a completed first attempt (status = "passed") and an open second-attempt session
Expected: Card shows "In progress" status badge and "Continue" button (per FR-BB313 AC-3, AC-4, AC-7 — when open_session_id is present the card must reflect the active session)
Actual: Card shows "Passed" status badge and "View result" button. API response confirms open_session_id is set (`open_session_id: "1e8f6b96-c5f4-4a3e-94e2-89d228bd09ea"`) but user_status is "passed" (computed from the previous completed attempt). Frontend renders the card based solely on user_status without checking open_session_id first.
API data: `{"user_status":"passed","open_session_id":"1e8f6b96-c5f4-4a3e-94e2-89d228bd09ea"}`
Screenshot: screenshot-s2-step5.png

## Root Cause
`ExamCard.tsx` computed `statusVariant`, `statusLabel`, `ctaLabel`, `handleCta`, `ctaDisabled`, `showCta`, and the button `variant` all from `exam.user_status` directly. When an employee has a completed first attempt (`user_status: 'passed'`) and opens a second attempt, the API correctly returns both `user_status: 'passed'` and `open_session_id: '<uuid>'`. However, the card rendered based solely on `user_status`, showing the "Passed" badge and "View result" button instead of the "In progress" badge and "Continue" button required by FR-BB313 AC-3, AC-4, and AC-7.

## Fix Applied
Added `displayStatus` computed variable in `ExamCard.tsx` that takes priority of `open_session_id` over `user_status`:
```typescript
const displayStatus: UserStatus = exam.open_session_id ? 'in_progress' : exam.user_status
```
All rendering logic (`statusVariant`, `statusLabel`, `handleCta`, `ctaLabel`, `ctaDisabled`, `showCta`, and Button `variant`) was updated to use `displayStatus` instead of `exam.user_status`. The existing `handleCta` `'in_progress'` case already navigates to `/portal/sessions/${exam.open_session_id}`, so it correctly handles the Continue button action.

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/pages/EmployeePortal/ExamCard.tsx` | Added `displayStatus` priority logic; replaced all `exam.user_status` references with `displayStatus` in rendering logic |
| `frontend/src/pages/EmployeePortal/EmployeePortal.test.tsx` | Added regression test: "shows 'In progress' status and 'Continue' button when open_session_id is set even if user_status is 'passed'" |

## Regression Test
File: `frontend/src/pages/EmployeePortal/EmployeePortal.test.tsx`
Test: `shows "In progress" status and "Continue" button when open_session_id is set even if user_status is "passed"`
Verifies that when `open_session_id` is non-null and `user_status` is `'passed'`, the card displays the "In progress" badge and "Continue" button, not "View result".

## Resolution Results
- Tests: 253 passed, 0 failed (45 test files)
- Migration applied: no
- Build clean: yes
