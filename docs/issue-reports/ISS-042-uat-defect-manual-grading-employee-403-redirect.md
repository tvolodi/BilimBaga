---
id: ISS-042
title: "UAT Defect: Manual Grading — Employee redirected to blank /admin page instead of 403 message"
status: resolved
severity: low
layer: frontend
module: grading
tags: [uat, manual-grading, employee-redirect, RequireRole]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: []
regression_test: frontend/src/components/RequireRole.test.tsx
---

## Symptom
UAT Scenario: `Manual Grading — Role-Based Access Control`, Step 3
Actor: Employee (role: employee)
Action: Navigate to `/admin/grading` while authenticated as an employee
Expected: Employee is redirected with a visible 403 message (per FR-BB47 AC-1: "employees are redirected with a 403 message")
Actual: Employee is redirected to `/admin` which renders a blank page. No 403 message or access-denied UI is displayed. Security enforcement is correct (employee cannot access the queue) but the user-facing redirect target and error message are absent.
Screenshot: none

## Root Cause
`RequireRole` (frontend/src/components/RequireRole.tsx) redirects any unauthorized user to `/admin` regardless of their role. When an employee navigates to `/admin/grading`, the outer `/admin` route's `RequireRole` guard (roles: `['super_admin', 'department_admin', 'examiner', 'hr_admin']`) redirects to `/admin`. The employee is also unauthorized at `/admin`, so `RequireRole` fires again with the same redirect — React Router detects the same-path replace and stops, leaving the page blank. The `AdminLayout` employee→portal redirect never executes because `RequireRole` blocks it upstream.

## Fix Applied
Changed `RequireRole` to redirect `employee` role to `/portal` instead of `/admin`. For all other role mismatches (e.g., an admin role without the specific permission), the redirect to `/admin` is preserved. This is a single-line change using a conditional redirect target: `role === 'employee' ? '/portal' : '/admin'`.

Updated two test files to match:
- `RequireRole.test.tsx`: added `/portal` route stub; updated employee-token test to assert redirect to "Portal Home" and updated its description.
- `ReportsPage.test.tsx` AC-2: added `/portal` route stub; updated assertion from "Admin Home" to "Portal Home".

## Files Changed
| File | Change |
|------|--------|
| `frontend/src/components/RequireRole.tsx` | Redirect `employee` role to `/portal` instead of `/admin` |
| `frontend/src/components/RequireRole.test.tsx` | Add `/portal` route stub; update employee redirect assertion and test title |
| `frontend/src/pages/admin/ReportsPage.test.tsx` | Add `/portal` route stub; update AC-2 assertion to "Portal Home" |

## Regression Test
`frontend/src/components/RequireRole.test.tsx` — test: "redirects employee to /portal when token role does not match (FR-BB47 AC-1: prevents blank-page loop)"

## Resolution Results
- Tests: 12 passed, 0 failed (RequireRole.test.tsx × 5 + ReportsPage.test.tsx × 7)
- Migration applied: no
- Build clean: yes (✓ built in 10.05s)
