# ISS-229 - GET /users/roles exposes per-caller `assignable`

Root cause: `GET /users/roles` returned only id/name/description/is_system, so the frontend
helper (PR #227) could not evaluate the D-1 rank rule for non-super_admin callers.

Design (safer option from the issue): the backend computes `assignable` per role for the
current caller. Other roles' permission lists are NOT exposed.

- `backend/internal/users/service.go`: extracted `roleAssignableBy(roleName, callerRole)`
  (canReachRole + sensitive-permission ban). `checkRoleAssignment` and the new
  `ListRoles(ctx, callerRole)` both call it - single source of truth. super_admin: all true.
- `types.go`: `RoleRow.Assignable bool` (`db:"-"`, json `assignable`); id/name kept (additive).
- `handler.go`: passes `auth.RoleFromCtx` to the service. Uses existing rbac lookups already
  injected into the service (permsFor/canPerm); no SQL, no migration.
- Frontend: `RoleRow.assignable?`, `canAssignRole` prefers the server flag, falls back to
  local computation when absent.

Tests: `backend/internal/users/roles_assignable_test.go` (super_admin, department_admin,
custom caller, examiner/employee, unknown caller, unknown permissions fail closed, flag ==
checkRoleAssignment outcome for 9 callers x 11 roles, handler JSON backward compat and no
`permissions` key); `frontend/src/lib/assignableRoles.test.ts` (server flag cases + fallback).

Results: `go test -p 1 ./internal/users/` ok; `go vet ./internal/users/` ok; vitest
assignableRoles.test.ts 11/11; `npm run check:i18n` ok. Full suites, tsc, go build ./... left
to CI (host memory constraint).
