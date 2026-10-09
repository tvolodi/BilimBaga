# FR-BB117 — Role Management (custom roles and permission matrix)

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB117 |
| Phase | 1 — Foundation (gap closure; extends FR-BB16 "static roles in Phase 1" to admin-managed roles; GitHub issue #135) |
| Priority | 2 |
| Status | Implemented (backend #137, frontend #138; live UAT pending) |
| Depends On | FR-BB16, FR-BB18, FR-BB19, FR-BB111, FR-BB62 |

## Description
Customer complaint: "Roles: can't find where a new role can be created." Investigation found this is a missing feature, not a discoverability problem. FR-BB16 explicitly put "Role creation or modification via API" and a permission-management UI out of scope (`docs/requirements/FR-BB16.RBAC.md:253-256`). The only role endpoint is the read-only lookup `GET /users/roles` (`backend/internal/router/router.go:99-100`), used by the user create/edit drawers. There is no roles route in `frontend/src/App.tsx`, no sidebar item in `frontend/src/components/admin/Sidebar.tsx:39-49`, and no `nav.roles` i18n key. The roadmap (`corporate_exam_platform_roadmap.md:52-56`) only seeds four roles and a fixed matrix.

This requirement adds a Roles admin page (super_admin only) and the supporting API so a super_admin can create, edit and delete custom roles and choose their permissions from the seeded `permissions` catalogue. The four built-in roles (`super_admin`, `department_admin`, `examiner`, `employee`) are marked `is_system` and are protected. Custom roles are assignable to users via the existing user drawers (they already read `GET /users/roles`).

## Acceptance Criteria
Backend
- [ ] AC-1: Migration `032_role_management` adds `roles.description TEXT NOT NULL DEFAULT ''` and `roles.is_system BOOLEAN NOT NULL DEFAULT false`, sets `is_system = true` for the four built-in names, and seeds permissions `roles:read` and `roles:manage`, granted to `super_admin` only. Migration is idempotent and has a working `.down.sql`. Existing migrations are not edited.
- [ ] AC-2: `GET /api/v1/roles` (requires `roles:read`) returns every role as `{id, name, description, is_system, user_count, permissions: ["resource:action", ...]}` ordered system roles first, then by name. `GET /api/v1/roles/{id}` returns one role (404 `NOT_FOUND` if unknown). `GET /api/v1/roles/permissions` returns the full catalogue `[{id, resource, action}]` for the matrix.
- [ ] AC-3: `POST /api/v1/roles` (requires `roles:manage`) with `{name, description?, permissions: [permission_id...]}` creates a custom role (`is_system=false`) and returns 201. `name` must match `^[a-z][a-z0-9_]{2,31}$`; otherwise 400 `VALIDATION_ERROR`. Duplicate name returns 409 `ROLE_NAME_TAKEN`. Unknown permission id returns 400 `VALIDATION_ERROR`. Role and its role_permissions rows are written in one transaction.
- [ ] AC-4: `PUT /api/v1/roles/{id}` (requires `roles:manage`) updates `description` and the permission set of a custom role (replace semantics, one transaction). `name` is immutable for all roles (a differing `name` in the body returns 400 `VALIDATION_ERROR`). For a system role the call returns 409 `ROLE_SYSTEM_IMMUTABLE` and changes nothing.
- [ ] AC-5: `DELETE /api/v1/roles/{id}` (requires `roles:manage`) returns 204 for a custom role with zero assigned users (including deactivated users). A system role returns 409 `ROLE_SYSTEM_IMMUTABLE`. A role with one or more users (any status) returns 409 `ROLE_IN_USE` with the user count in the message. Unknown id returns 404.
- [ ] AC-6: Privilege-escalation guard: permissions `roles:read`, `roles:manage` and `tenant:manage` cannot be granted to a custom role; the request returns 400 `VALIDATION_ERROR` naming the offending permission.
- [ ] AC-7: Every successful create, update and delete writes an audit entry via `audit.Writer` with action `role.create`, `role.update`, `role.delete`, entity_type `role`, entity_id = role UUID, metadata containing `name` and, for update, `permissions_added` and `permissions_removed` (resource:action strings). Failed requests write no audit entry.
- [ ] AC-8: After a successful create, update or delete the in-memory `rbac.Cache` is rebuilt (`Cache.Load`) before the response is returned. A user holding the role gets new permissions on their next request without re-login, and a revoked permission yields 403 on the next request. If the cache reload fails the handler returns 500 `INTERNAL`, logs the error, and does not report success (the DB change stays committed; a subsequent successful mutation or restart reloads the cache).
- [ ] AC-9: Existing `GET /api/v1/users/roles` keeps its response shape (additive fields only) so the user create/edit drawers show custom roles. `POST/PUT /users` accept a custom role id. `checkRoleAssignment` still blocks non-super_admin from assigning `super_admin`.
- [ ] AC-10: All new endpoints return the envelope `{data, error:null}` / `{data:null, error:{code,message}}`; a caller without the permission (department_admin, examiner, employee) receives 403 and an unauthenticated caller 401.

Frontend
- [ ] AC-11: A "Roles" item (`nav.roles`, icon `ShieldCheck`) appears in the admin sidebar for `super_admin` only, linking to `/admin/roles`. Other roles do not see it and are redirected away from the route (same `RequireRole` pattern as `AUDIT_READ_ROLES`).
- [ ] AC-12: `/admin/roles` lists roles in a table: name, description, type badge (System / Custom), user count, permission count, actions. System rows show no Delete action and a read-only "View" instead of "Edit". A "Create role" button is shown at the top of the page.
- [ ] AC-13: Create/Edit uses a drawer or dialog with name (disabled on edit), description and a permission matrix grouped by resource with a checkbox per action, fed by `GET /roles/permissions`. Non-assignable permissions (AC-6) are not selectable. Server errors (`ROLE_NAME_TAKEN`, `VALIDATION_ERROR`) display inline with translated text. On success the list is refreshed (React Query invalidation of `['roles']` and the existing users-roles key) and a toast is shown.
- [ ] AC-14: Delete uses a confirmation dialog. `ROLE_IN_USE` (409) shows a translated message including the user count and leaves the role in place.
- [ ] AC-15: A role created here appears in the Role select of `UserCreateDrawer` and `UserEditDrawer` without a page reload.
- [ ] AC-16: A user whose role is a custom role (not one of the four built-ins) can sign in, is routed to the admin shell, and sees only sidebar items and routes whose required permission the role holds. To support this, `GET /users/me` returns `permissions: ["resource:action", ...]` for the caller's role, and `RequireRole` / `Sidebar` treat non-built-in roles by permission instead of by role name (mapping defined in `frontend/src/lib/routeRoles.ts`: users->`users:read`, departments->`departments:read`, questions->`questions:read`, categories->`categories:manage`, tags->`tags:read`, exams->`exams:read`, grading->`grading:read`, reports->`reports:read`, audit->`audit:read`, settings->`tenant:manage`). Built-in role behaviour is unchanged.
- [ ] AC-17: All new user-visible strings use i18n keys under `roles.*` and `nav.roles` in `en.json`, `ru.json` and `kk.json` (identical key sets; permission labels under `roles.permissions.<resource>.<action>`). No hardcoded strings. Page is keyboard accessible (labels, focus trap in dialog, matrix checkboxes labelled).

Tests
- [ ] AC-18: `backend/internal/roles/service_test.go` and `handler_test.go` cover: create (valid, bad name, duplicate, unknown permission, forbidden permission), update (custom OK, system 409, name change 400), delete (OK, system 409, in-use 409, 404), audit call per mutation, cache reload invoked per mutation, 403 for non-super_admin. `internal/rbac` gets a test that `Load` reflects a newly created role. Frontend: `RolesPage.test.tsx` (list, system rows protected, create flow, in-use error), `Sidebar.test.tsx` (Roles item super_admin only), `RequireRole` custom-role permission test. All run and pass (`go test ./...`, `npm test`).

## Technical Specification

### Database Schema
```sql
-- 032_role_management.up.sql
ALTER TABLE roles
    ADD COLUMN description TEXT    NOT NULL DEFAULT '',
    ADD COLUMN is_system   BOOLEAN NOT NULL DEFAULT false;

UPDATE roles SET is_system = true
 WHERE name IN ('super_admin','department_admin','examiner','employee');

INSERT INTO permissions (resource, action) VALUES ('roles','read'), ('roles','manage')
ON CONFLICT (resource, action) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
 WHERE r.name = 'super_admin' AND p.resource = 'roles'
ON CONFLICT DO NOTHING;
```
Down: drop the two columns, delete `role_permissions` rows for `roles:*`, delete those permissions; custom roles are deleted only if none are referenced (otherwise the down migration fails loudly). No tenant scoping: `roles` is deployment-global today (no `tenant_id` column), unchanged. `users.role_id` FK (migration 004) has no ON DELETE action, so the DB also rejects deleting an in-use role; the service checks first to return a clean 409.

### API Contract
Prefix `/api/v1`. Router additions next to the user routes in `backend/internal/router/router.go`:

| Method / path | Permission | Success | Errors |
|---|---|---|---|
| GET `/roles` | `roles:read` | 200 list | 401, 403 |
| GET `/roles/permissions` | `roles:read` | 200 catalogue | 401, 403 |
| GET `/roles/{id}` | `roles:read` | 200 role | 401, 403, 404 |
| POST `/roles` | `roles:manage` | 201 role | 400 VALIDATION_ERROR, 403, 409 ROLE_NAME_TAKEN |
| PUT `/roles/{id}` | `roles:manage` | 200 role | 400, 403, 404, 409 ROLE_SYSTEM_IMMUTABLE |
| DELETE `/roles/{id}` | `roles:manage` | 204 | 403, 404, 409 ROLE_SYSTEM_IMMUTABLE / ROLE_IN_USE |

`/roles/permissions` is registered before `/roles/{id}`. Role object: `{id, name, description, is_system, user_count, permissions[], created_at}`. POST body: `{name, description, permissions:[uuid]}`; PUT body: `{description, permissions:[uuid]}`.

### Go Implementation Notes
- New package `backend/internal/roles` (handler / service / repository / types, no circular imports; `users` keeps its own `ListRoles`). Handler calls `audit.Writer.Write` as the `departments` handler does. Service takes a small `CacheReloader` interface (or closure around `rbacCache.Load(db)`) so tests can stub it.
- JWT note: the access token carries the role NAME (`backend/internal/auth/middleware.go:47`) and `RequirePermission` resolves permissions by name per request, so permission edits apply immediately and no token re-issue is needed. This is why role name is immutable (a rename would orphan live tokens) and why deletion of an in-use role is blocked.
- Multi-instance deployments: the cache is per process. Only the instance handling the request is reloaded; other instances pick up changes on restart. Documented limitation (see Notes).
- Role-name branching in code (`users.checkRoleAssignment` at `backend/internal/users/service.go:383`, `callerRole == "department_admin"` scoping at `:65`) keeps working. Implementer must verify the `users` service scoping does not widen access for a custom role holding `users:read` / `users:manage`: a custom role must not see other departments' users (only `super_admin` is org-wide). If current switch statements would fall through to org-wide access, restrict scoping to the caller's own department for every non-super_admin role.
- `GET /users/me` adds `permissions` (add a `Cache.PermissionsFor(role) []string` accessor to `rbac.Cache`).

### Frontend Implementation Notes
- Files: `frontend/src/api/roles.ts` (hooks `useRolesAdmin`, `usePermissionsCatalogue`, `useCreateRole`, `useUpdateRole`, `useDeleteRole`; keys `['roles']`, `['roles','permissions']`; invalidate the existing `useRoles` key from `api/users.ts`), `frontend/src/pages/admin/roles/RolesPage.tsx`, `RoleFormDrawer.tsx`, `PermissionMatrix.tsx`. Route `/admin/roles` in `App.tsx` wrapped in `<RequireRole roles={ROLES_MANAGE_ROLES}>`; add `ROLES_MANAGE_ROLES = ['super_admin']` to `lib/routeRoles.ts`. Sidebar item `{ key:'roles', icon: ShieldCheck, path:'/admin/roles', labelKey:'nav.roles', roles: ROLES_MANAGE_ROLES }` placed after Users.
- i18n namespaces (`frontend/src/locales/{en,ru,kk}.json`): `nav.roles`, `roles.title`, `roles.create`, `roles.columns.*`, `roles.type.system|custom`, `roles.form.*`, `roles.permissions.<resource>.<action>`, `roles.errors.*`, `roles.delete.*`. shadcn/ui: Table, Sheet or Dialog, Checkbox, Badge, AlertDialog.
- The `users/me` response type gains `permissions: string[]`.

## Notes
Out of scope (explicit):
- Creating, editing or deleting permissions themselves (the catalogue stays migration-seeded; FR-BB16 out-of-scope for permission CRUD still holds).
- Editing the permission set of the four system roles through the UI/API.
- Renaming roles, role hierarchy/inheritance, per-tenant roles (no `tenant_id` on `roles`), per-resource-instance ownership rules.
- Cross-instance cache invalidation (pub/sub or polling) and `POST /admin/rbac/reload`. Single-instance reload only.
- Bulk reassignment of users from one role to another before delete (admin reassigns via the Users page).
- Localized role names in the DB (labels come from `description`; system role display names remain existing i18n / `RoleBadge`).
- CSV user import with custom role names (existing behaviour unchanged; only verify it does not break).

Sizing: L (about 2 PRs: backend roles package + migration + `users/me` permissions, then frontend page + permission-aware guards). Smallest viable slice if needed: AC-1..10, 11-15, 17, 18 without AC-16 (custom-role holders would be redirected to `/admin` but see no items); not recommended because assigning such roles would produce broken logins.

UAT hook: `docs/uat-scenarios/role-management-20261009.md`.

## Implementation notes (backend, issue #137)

- Migration number is `034_role_management` (032/033 were taken by the time of implementation).
- Validation failures return HTTP 422 `VALIDATION_ERROR` (repo-wide convention, see docs/requirements/api-conventions), not 400; malformed JSON is 400 `INVALID_BODY`.
- `PUT /roles/{id}` requires the `permissions` field (replace semantics); omitting it is a validation error.
- Default-deny scoping: only `super_admin` is org-wide in the users service; every other role (custom included) is limited to its own department. Exam assignment is org-wide only for `super_admin` and `examiner`.
- A cache-reload failure after a committed mutation returns 500 `INTERNAL` and writes no audit entry (per AC-7/AC-8).
