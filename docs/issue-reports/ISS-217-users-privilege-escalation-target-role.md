---
id: ISS-217
title: users:manage holder can reset/edit/deactivate/unlock a more privileged user (account takeover)
status: resolved
severity: critical
layer: backend
module: users
tags: [ResetPassword, UpdateUser, DeactivateUser, UnlockUser, checkTargetActionable, FR-BB117, D-1, privilege-escalation]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: backend/internal/users/privilege_escalation_test.go
---

## Symptom
GitHub issue #217 (parent #214, conformance gap G2). A department_admin, or any custom role with `users:manage`, could `POST /users/{id}/reset-password` for a super_admin / department_admin / examiner in its department and receive `temporary_password` in the response, i.e. take over the account. Update, deactivate and unlock had the same hole.

## Root Cause
`service.UpdateUser`, `DeactivateUser`, `ResetPassword`, `UnlockUser` checked only `inCallerScope(department)`; the target's role was never examined. `checkRoleAssignment` (PR #185) covered only the role being assigned.

## Fix Applied
`checkTargetActionable(target, callerRole)` in `backend/internal/users/service.go`, called in all four methods right after the department-scope check and before any password generation or repository write. Shared rule `checkRoleReach(roleName, callerRole)` (also used by `checkRoleAssignment`): super_admin callers unrestricted; super_admin targets forbidden; target role equal to the caller's role allowed; unknown/legacy (empty) target role forbidden (fail-closed); if a permissions lookup is wired, the target role's permissions must be a subset of the caller's, except built-in caller vs built-in target which keeps the historical behaviour (the seeded department_admin set is not a strict superset of examiner/employee, so a strict rule would break FR-BB18 AC-6). Forbidden returns the existing `ErrForbidden` (403 FORBIDDEN), no temporary password. No SQL, no migration. CSV import only creates users (role assignment already checked); there is no bulk update endpoint.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/users/service.go | checkTargetActionable, checkRoleReach; wired into 4 mutations |
| backend/internal/users/privilege_escalation_test.go | service and handler tests |
| docs/uat-scenarios/role-management-20261009.md | S7 steps 2-3 updated, curl recipe |

## Regression Test
`backend/internal/users/privilege_escalation_test.go`: forbidden/allowed matrix across reset, update, deactivate, unlock; no password write or temp password; legacy role; other department; self-service; handler 403 without `temporary_password`.

## Resolution Results
- Tests: `go test -count=1 -p 1 ./internal/users/...` passed
- Migration applied: no
- Build clean: `go vet ./internal/users/...` clean; full-module build/vet/test not run (memory hold)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
