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

## Rank rule (Supervisor decision)

Rule D-1 (FR-BB117, amended 2026-10-09): strict rank hierarchy `super_admin` > `department_admin` > `examiner` > `employee`, implemented as the explicit table `builtinRank` in `backend/internal/users/service.go` (employee 1, examiner 2, department_admin 3, super_admin 4) and decided in one function, `canReachRole`.

- Built-in caller vs built-in target (or role being assigned): allowed only if the target's rank is strictly lower than the caller's. department_admin may therefore manage examiners and employees, never a peer department_admin or a super_admin; examiner only employees; employee nobody.
- Built-in caller vs custom-role target: the target's permissions must be a subset of the caller's.
- Custom-role caller: ranks below department_admin; any department_admin or super_admin target is refused; examiner, employee and custom targets only when permission-subset. A custom caller holding no permissions is refused.
- super_admin: unrestricted. Department scope is unchanged for everyone else.
- Role assignment on create/update/import uses the same function; custom roles carrying roles:read, roles:manage or tenant:manage can never be assigned by a non-super_admin. An unchanged role on update is not re-checked (the target check covers it).
- Self-service unchanged: acting on one's own record is allowed when the stored role equals the token role; own role change stays blocked.
- Fail closed: empty/unknown caller or target role names, and a missing permission lookup (permsFor/canPerm nil), give 403. Authorisation runs before password generation, DB writes and email, so no temporary password is produced for a forbidden target.

Behaviour change versus the previous head: department_admin vs peer department_admin is now 403 (was allowed); department_admin can no longer assign the department_admin role; examiner vs examiner/department_admin is 403; subset checks are no longer skipped silently when the lookup is missing.

## Regression Test
`backend/internal/users/privilege_escalation_test.go`: forbidden/allowed matrix across reset, update, deactivate, unlock; no password write or temp password; legacy role; other department; self-service; handler 403 without `temporary_password`.

## Resolution Results
- Tests: `go test -count=1 -p 1 ./internal/users/...` passed
- Migration applied: no
- Build clean: `go vet ./internal/users/...` clean; full-module build/vet/test not run (memory hold)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
