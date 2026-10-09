---
id: ISS-226
title: User create/edit role dropdown and CSV hint offer roles the caller may not assign
status: resolved
severity: medium
layer: frontend
module: users
tags: [FORBIDDEN, assignableRoles, role dropdown, FR-BB117 D-1]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-217]
regression_test: frontend/src/pages/users/UserRoleAssignment.test.tsx
---

## Symptom
A department_admin saw every role (incl. super_admin, department_admin) in the user create/edit drawers; choosing one produced a raw backend 403 message (or nothing, for reset-password).

## Root Cause
Drawers rendered all of GET /users/roles unfiltered; no 403 mapping.

## Fix Applied
Rule as in backend `builtinRank`/`canReachRole`/`checkRoleAssignment` on main (02301aed, PR #222), mirrored in `src/lib/assignableRoles.ts`:
- super_admin: any role. Everyone else: never super_admin.
- Built-in caller: built-in role only if rank strictly lower (employee 1 < examiner 2 < department_admin 3 < super_admin 4); custom role only if its permissions are a non-empty subset of the caller's and contain no roles:read/roles:manage/tenant:manage.
- Custom caller: built-in roles of rank >= department_admin refused; every other role needs a permission subset.
- Unknown/empty permissions fail closed.
Limitation: GET /users/roles returns only {id,name}, so permissions of custom/built-in roles are unknown to non-super_admin callers; custom roles (and, for custom callers, examiner/employee) are therefore hidden unless the API starts returning `permissions` (optional field already honoured). Backend stays authoritative. No difference from FR-BB117 D-1 docs found.
Edit drawer: when the user's current role is not assignable the select is disabled and shows the current role (backend skips the check for an unchanged role). Import modal shows the assignable roles. FORBIDDEN on create/update/deactivate/reset maps to `users.messages.forbidden` (en/ru/kk); reset-password errors were previously swallowed and are now shown.

## Files Changed
| File | Change |
|------|--------|
| frontend/src/lib/assignableRoles.ts (+test) | pure helper, error-key mapper |
| frontend/src/hooks/useAssignableRoles.ts | hook |
| frontend/src/pages/users/{UserCreateDrawer,UserEditDrawer,ImportModal,DeactivateConfirmDialog,UsersListPage}.tsx | use helper, 403 message |
| frontend/src/locales/{en,ru,kk}.json | forbidden, role_locked, assignable_roles_hint, no_assignable_roles |
| frontend/src/pages/users/UserRoleAssignment.test.tsx | new; UserCreateDrawer.test.tsx adjusted to super_admin token |

## Regression Test
UserRoleAssignment.test.tsx (8), assignableRoles.test.ts (7).

## Resolution Results
- Tests: targeted vitest 7 + 8 + 3 + 4 + 3 passed, 0 failed; check:i18n OK
- Migration applied: no
- Build clean: not verified (tsc/full vitest skipped, host memory alert)
