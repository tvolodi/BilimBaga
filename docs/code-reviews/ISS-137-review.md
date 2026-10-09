# Code Review ISS-137 — FR-BB117 backend role management

Branch: swarm/137-roles-backend, commit 1e19232 vs origin/main.
Tests run: `go test -p 1 ./internal/roles ./internal/users ./internal/exams ./internal/rbac ./internal/router` — all ok.

## Checked and OK
- Non-assignable guard (roles:read, roles:manage, tenant:manage) is applied on both create and update after de-duplication and existence checks (roles/service.go:87).
- Migration 034 up is idempotent (IF NOT EXISTS, ON CONFLICT). Down refuses to run while users hold custom roles, then removes the custom roles, the roles:* permissions and the added columns.
- Repository SQL is parameterized. The `$n::uuid[]` casts with pq.Array are correct. Create and Update use a single transaction. Delete maps FK violation 23503 to ROLE_IN_USE. The `is_system = false` guard is repeated in SQL.
- Route order is correct (`/roles/permissions` before `/roles/{id}`). The routes sit inside the authenticate group, so `RequireUUIDPathParams("id")` covers `{id}`. The permission middleware is applied on every route. Envelope and status codes are consistent.
- Scoping in users: `isOrgWide` is super_admin only, and every service method uses `inCallerScope`. CSV import now rejects super_admin for non-super callers. Moving a user out of the caller's department is blocked. ListUsers returns an empty result when the caller has no department.
- No other role-name switches remain in the codebase (grep of `RoleFromCtx` users: ai, portal, sessions, rbac, users, exams).

## Verdict
VERDICT: FAIL

## Findings
1. [major] backend/internal/exams/service.go:417-424 and :462-469 — Default-deny is incomplete for exam assignments. For any non-org-wide caller (now including every custom role holding exams:assign) only `all` and `department` assignees are restricted. `AssigneeType == "user"` (and any other type) is not checked against the caller's department, so a scoped role can assign or delete assignments for users and assignments org-wide. This was previously a department_admin-only gap and is now widened to custom roles. Fix: for scoped callers, resolve the assignee user's department and require it to equal CallerDeptID. For `user` assignments in DeleteAssignment, compare the assignee's department too. Reject unknown assignee types. Also reject scoped callers with an empty CallerDeptID.
2. [major] backend/internal/users/service.go:415-430 (`checkRoleAssignment`) — Privilege escalation through custom roles. A non-super_admin `users:manage` holder (department_admin or any custom role) may assign any role except `super_admin` through create, update and CSV import. That includes `examiner` and any custom role with broad permissions (for example audit:read or org-wide exams and reports) that the caller does not hold. A custom role can also hold users:manage and be assigned the same way. The spec (AC-9) only requires blocking super_admin, so this is spec-compliant, but it defeats the purpose of the escalation guard. Fix: require that the target role's permission set is a subset of the caller's. This needs a rbac.Cache lookup, which the PermissionChecker plumbing already allows. Alternatively restrict non-super callers to employee only. At minimum, get an explicit sign-off and record it in the spec.
3. [minor] backend/internal/users/service.go:366 — The CSV check compares `row.RoleName == "super_admin"` as a literal and is not case/whitespace normalised, whereas the lookup is by exact name. It is safe only while role names are lowercase-only (the nameRe in roles ensures this for custom roles). Prefer resolving the role via GetRoleIDByName and reusing `checkRoleAssignment(roleID)` in one place, so create, update and import cannot diverge.
4. [minor] backend/internal/roles/handler.go:101,119,134 + service.go:122,153,188 — When the cache reload fails after a committed mutation, a 500 is returned and no audit entry is written, although the DB change is committed. The spec accepts this (AC-7/AC-8, implementation note). Consider writing the audit entry before returning the 500 (with metadata `cache_reload_failed`) so a real state change is never unaudited. Also consider a retry or a log at error level with the role id. The failure is logged by `fail`'s default branch, but ErrCacheReload carries only a `%v` string, so the cause is lost for `errors.Is`.
5. [minor] backend/internal/roles/service.go:128-160 — Update reads `existing` (for the permission diff) outside the transaction. A concurrent update can make `permissions_added`/`permissions_removed` in the audit entry inaccurate. Also a cache reload is a full `Load` per mutation with no ordering guarantee between concurrent mutations. Acceptable at this scale. Optionally compute the diff inside the repository transaction (SELECT ... FOR UPDATE).
6. [minor] spec vs implementation — AC-3/AC-4/AC-6 say 400 VALIDATION_ERROR, but roles/handler.go:40 returns 422 (consistent with the project checklist, which lists 422 for validation). Update the spec text or the handler. The invalid-JSON path correctly uses 400 INVALID_BODY. Note the spec numbers the migration `032` in AC-1 but the implementation is `034` (fine, the spec is stale).
7. [minor] backend/internal/roles/handler.go:92,110 — No body size limit (`http.MaxBytesReader`) and no `DisallowUnknownFields`, and a permissions array of unbounded length is accepted. The endpoint is super_admin-only, so the risk is low.
8. [minor] backend/internal/users/service.go:129 — GetUser still special-cases the literal `department_admin` as "can read within department" even when the permission checker is present. This is harmless (it only grants dept-scoped read) but is a leftover role-name branch. Use `canPerm(..., "users", "read")` alone once canPerm is always wired.

## AC coverage (backend)
AC-1 covered (migration is 034), AC-2 through AC-10 covered, AC-16 (`/users/me` permissions) covered, AC-18 backend tests present and passing. Frontend ACs (11-15, 17) are out of scope for this commit.

Summary: the role CRUD, migration and users scoping are solid. The exam-assignment scoping gap (1) and the unrestricted custom-role assignment by non-super callers (2) should be fixed before merge.

## Author response (post-review fixes)
- Finding 1: fixed. `user` assignees are checked against the caller's department (new `Repository.UserDepartmentID`), for create and delete; test added.
- Finding 2: fixed for custom-role callers: target role permissions must be a subset of the caller's (`users.WithPermissionsLookup`); built-in callers unchanged to avoid regressing existing flows; test added.
- Finding 3: fixed. CSV import now resolves the role id and reuses `checkRoleAssignment`.
- Finding 4: underlying reload error now preserved (`%w`); no-audit-on-reload-failure kept per AC-7/AC-8.
- Findings 5-8: accepted as minor (5 benign under single super_admin use; 6 documented in the FR doc; 7 super_admin-only; 8 built-in fallback kept).
