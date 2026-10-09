# Conformance Review: FR-BB117 (PR #185, #206), PR #204, PR #211, PR #205

- Date: 2026-10-09
- Reviewer: Business Analyst (static code review; no live stack, no UAT executed, tests not re-run)
- Base: origin/main at 4a4e407 (worktree `ba`, branch swarm/ba-conformance-117)
- Scope: FR-BB117 (18 ACs) vs backend PR #185 (`ed22fd5`, migration 034) and frontend PR #206 (`147b069`); PR #204 (`6921852`, ISS-171); PR #211 (`4a4e407`, ISS-181 / ISS-164); PR #205 (`6e256ac`, ISS-195).
- Severity: p1 only for security, privilege escalation, data exposure or broken core flow; p2 behaviour/spec defect; p3 minor or documentation.

## p1 gaps (privilege escalation / data exposure)

**G2 (p1, PRIVILEGE ESCALATION RISK). A scoped user-manager can take over a more privileged account in its own department.** `ResetPassword`, `UpdateUser` and `DeactivateUser` in `backend/internal/users/service.go:200-262` check only that the target user is in the caller's department (`inCallerScope`). They never compare the target's current role with the caller's. `ResetPassword` returns the new temporary password in the response body (`service.go:292`). A custom role holding `users:manage` (grantable, only `roles:*` and `tenant:manage` are blocked, `roles/service.go:21-25`) can therefore reset the password of any `department_admin`, `examiner` or `super_admin` user who is assigned to the same department, then log in as them. `checkRoleAssignment` (`service.go:432-461`) only guards the role being assigned, not the existing role of the target. The same hole exists for built-in `department_admin` (pre-existing), but FR-BB117 widens the population of callers to every custom role with `users:manage`. Requires the higher-privileged account to have `department_id` set to the caller's department; the seeded admin has none, so exposure depends on data. Suggested fix: reject when the target's role permission set is not a subset of the caller's (reuse the `checkRoleAssignment` subset rule against `existing.RoleName`), and always forbid targeting `super_admin` for non-super_admin callers. Not covered by `users/scoping_test.go`.

**G1 (p1, DATA EXPOSURE, needs a product decision). Department scoping of reports, grading detail and AI is by literal role name.** `backend/internal/deptscope/scope.go:18,32` restricts only `role == "department_admin"`; `FromContext` returns an unrestricted scope for every other role. A custom role that holds `reports:read` (grantable, and used in UAT S5) therefore sees organisation-wide dashboard, per-exam analytics, employee records, results export and the AI narrative, whereas `department_admin` with the same permission is limited to its department subtree. The same applies to `grading:read/write` via `deptscope.RequireSessionInScope` (`router.go:96-101`). FR-BB117 design note (line 82) demands default-deny only for the `users` service, and PR #185 delivered exactly that (`users/service.go:68-80`), so this is a spec blind spot rather than a deviation. Decision needed: either custom roles are department-scoped like `department_admin` (recommended, consistent with default-deny; change `FromContext` to restricted unless role is `super_admin` or `examiner`), or the FR states that a granted `reports:read` / `grading:*` is org-wide. Until decided, treat as p1 because the wrong default exposes cross-department employee results.

## 1. FR-BB117 acceptance criteria

| AC | Result | Evidence | Tests | Sev |
|----|--------|----------|-------|-----|
| 1 Migration | PASS with doc fix. Shipped as `034_role_management` (032/033 taken; 035 now used by ISS-181). Idempotent (`ADD COLUMN IF NOT EXISTS`, `ON CONFLICT DO NOTHING`), sets `is_system`, seeds `roles:read/manage` for super_admin only, working `.down.sql` that fails loudly when custom roles are assigned (`down.sql:3-10`) and deletes `role_permissions` via cascade. FR text corrected. | `migrations/034_role_management.up.sql:6-21`, `.down.sql` | none for SQL (needs-live-db) | - |
| 2 Read endpoints | PASS. Shape `{id,name,description,is_system,user_count,permissions,created_at}`, ordered `is_system DESC, name`, `GET /roles/permissions` registered before `/{id}`, 404 NOT_FOUND. Permissions never null. | `roles/repository.go:30-76`, `router.go:146-158`, `roles/types.go:32-40` | `handler_test.go` List/Get/ListPermissions | - |
| 3 Create | PASS. Name regex, duplicate gives 409 `ROLE_NAME_TAKEN` (pg 23505), unknown/malformed permission id gives validation error, role and permissions in one tx. Deviation: validation status is 422, not 400 (repo convention, documented in FR). Description capped at 500 chars (extra, undocumented in AC). | `roles/service.go:106-126`, `repository.go:94-117`, `handler.go:60-66` | `service_test.go` Create_* | p3 doc |
| 4 Update | PASS. Replace semantics in one tx, SQL repeats `is_system = false`, system role gives 409 `ROLE_SYSTEM_IMMUTABLE`, differing name rejected (checked before the system check, so a system role with a changed name answers 422, not 409). `permissions` required. | `service.go:128-161`, `repository.go:119-142` | `TestUpdate_*` | p3 |
| 5 Delete | PASS. 204; system 409; in-use 409 `ROLE_IN_USE` with count in message; counts all users regardless of status (`SELECT COUNT(*) FROM users WHERE role_id`); FK violation race mapped to ROLE_IN_USE; unknown id 404. | `service.go:163-192`, `repository.go:177-197`, `types.go:24-26` | `TestDelete_*` | - |
| 6 Escalation guard | PASS for the listed set (`roles:read`, `roles:manage`, `tenant:manage`) on create and update, error names the permission. Observation: other powerful permissions stay grantable (`users:manage`, `audit:read`, `exams:assign`); see G2 and the observations below. | `service.go:21-25,87-89` | `TestCreate_ForbiddenPermissions_PrivilegeEscalationGuard`, `TestUpdate_ForbiddenPermissionAndMissingPermissions` | see G2 |
| 7 Audit | PASS. `role.create/update/delete`, entity_type `role`, entity_id = role UUID, metadata `name` (+`permissions` on create, `permissions_added/removed` on update, sorted, empty arrays not null). No audit on failed requests. Gap p3: if the cache reload fails after a committed change the request returns 500 and no audit row is written although the DB changed. | `roles/handler.go:84-117` | `TestHandlerCreate_201AndAudit`, `TestHandlerUpdate_200AndAuditDiff`, `TestHandlerDelete_204AndAudit` | p3 |
| 8 Cache reload | PASS. `reloadCache` after every committed mutation, wrapped as `ErrCacheReload` which falls to default branch: 500 `INTERNAL`, logged, success not reported. Reload closure is `rbacCache.Load(db)` (`cmd/api/main.go:265`). Keyed by role name, so edits apply on the next request without re-login. Single-instance only (documented). | `service.go:96-104`, `handler.go:25-37` | `TestCreate_CacheReloadFailure`, `TestDelete_CacheReloadFailure`, `rbac/permissions_for_test.go:18` | - |
| 9 users API compat | PASS. `GET /users/roles` adds only `description`, `is_system`; create/update accept custom role ids; `checkRoleAssignment` still blocks non-super_admin from `super_admin`, and additionally enforces a permission-subset rule for custom targets/callers and blocks self role change (extra hardening, not in FR). CSV import also checks assignment (fixes a pre-existing hole). | `users/repository.go:239`, `users/service.go:432-461,207-221,376-396` | `users/scoping_test.go` (14 tests) | - |
| 10 Envelope/401/403 | PASS. `{data,error}` envelope via `api.WriteJSON/WriteError`; routes behind `RequirePermission`; only super_admin holds `roles:*` (migration). | `roles/handler.go:25-27`, `router.go:146-158` | `TestRoutesRequireRolesPermissions_403ForNonSuperAdmin` (403; 401 comes from the pre-existing auth middleware, not tested here) | - |
| 11 Sidebar/route | PASS. Item `roles` after Users, `ShieldCheck`, `roles: ROLES_MANAGE_ROLES` (super_admin); route wrapped in `RequireRole roles={ROLES_MANAGE_ROLES} permission={PERM.roles}`; department_admin redirected to `/admin`. | `Sidebar.tsx:46-62`, `App.tsx:203`, `routeRoles.ts:13` | `permissionGuards.test.tsx:141,187` | - |
| 12 List page | PASS. Name, description, type badge, user count, permission count, actions; system rows get View and no Delete; Create button. | `RolesPage.tsx:123-150` | `RolesPage.test.tsx:58,69,80` | - |
| 13 Form/matrix | PASS. Name disabled on edit, matrix grouped by resource from `GET /roles/permissions`, `NON_ASSIGNABLE_PERMISSIONS` not selectable, translated server errors via `roleErrorKey`, invalidation of `['roles']` and `['users','roles']`, toast. | `RoleFormDialog.tsx:102-167`, `PermissionMatrix.tsx:50`, `api/roles.ts:44-122` | `RolesPage.test.tsx:105-181`, `PermissionMatrix.test.tsx` | - |
| 14 Delete dialog | PASS. Confirmation, `ROLE_IN_USE` translated with user count parsed from the message (`/(\d+)/`), role stays listed. | `RolesPage.tsx:52-56`, `api/roles.ts:116-119` | `RolesPage.test.tsx:184-214` | p3 (count parsed from English message text; fragile if the message changes) |
| 15 Users drawer refresh | PASS. `['users','roles']` is the key of `useRoles` (`api/users.ts:223`) and is invalidated. | `api/roles.ts:47` | `roles.test.ts` | - |
| 16 Custom-role login/guards | PASS. `GET /users/me` returns `permissions` from `Cache.PermissionsFor(u.RoleName)` (role name from DB, name immutable); `isCustomRole` = not in built-in list; `RequireRole`, `Sidebar` and `AdminHome` gate by permission; stale `/users/me` for another role ignored; unknown permissions deny; no redirect loop. Mapping matches the FR list (`users:read`, `departments:read`, `questions:read`, `categories:manage`, `tags:read`, `exams:read`, `grading:read`, `reports:read`, `audit:read`, `tenant:manage`). Settings can never be shown to a custom role (`tenant:manage` not grantable), by design. JWT carries role name; hazard handled. | `users/handler.go:97-107`, `rbac/cache.go:78-92`, `useMyPermissions.ts`, `RequireRole.tsx`, `AdminHome.tsx`, `routeRoles.ts:29-77` | `permissionGuards.test.tsx` (24 cases), `users/scoping_test.go:144,173` | - |
| 17 i18n/a11y | i18n PASS: en, ru, kk each have 876 keys, identical key sets, and an identical `roles.*` + `nav.roles` set of 79 keys including `roles.permissions.<resource>.<action>` for all 21 seeded permissions. Accessibility NOT-VERIFIABLE statically (focus trap, return focus, labelled checkboxes): code shows `aria-labelledby`, `aria-invalid`, `aria-describedby` and a dialog focus-return fix with `dialog.test.tsx`; needs UAT S6.2. | `locales/*.json`, `RoleFormDialog.tsx:102-117`, `ui/dialog.tsx` | `dialog.test.tsx` | - |
| 18 Tests | PASS (presence). Backend `roles/service_test.go` (17), `handler_test.go` (12), `rbac/permissions_for_test.go`, `users/scoping_test.go`; frontend `RolesPage.test.tsx`, `PermissionMatrix.test.tsx`, `roles.test.ts`, `permissionGuards.test.tsx` (covers Sidebar and RequireRole instead of separate `Sidebar.test.tsx`), plus `apiFetch` 204 fix. Pass state not re-run here. Real-DB behaviour of `roles/repository.go` and migration 034 has no integration test (NOT-VERIFIABLE). | commits `ed22fd5`, `147b069` | - | p3 |

### Design hazards from the FR

| Hazard | Result |
|--------|--------|
| Role-name scoping in users service: custom role with `users:read` must not fall to org-wide | PASS. `isOrgWide` is `super_admin` only; list, get, create, update, deactivate, reset, unlock, import all use `inCallerScope`; a caller without department sees nothing (`users/service.go:62-80,103-112`). Exam assignment: org-wide only for `super_admin` and `examiner`, plus user-assignee department check (`exams/service.go:530-550`). Remaining name-based scoping outside users/exams: **G1**. Target-role blindness in mutations: **G2**. |
| JWT carries role name, name immutable | PASS (update rejects a differing name; delete blocked while any user holds the role). |
| Escalation guard, system roles immutable | PASS (service plus SQL `is_system = false` guards). |
| 409 ROLE_IN_USE incl. deactivated users | PASS (count has no status filter). |
| RBAC cache reload failure gives 500 | PASS (p3: no audit row on that path). |
| Migration numbering | PASS after doc fix (034; 035 is the email index). |
| i18n key parity | PASS (see AC-17). |

### Observations (not gaps against the written ACs)
- `audit:read`, `users:manage`, `exams:assign`, `portal:*` remain assignable. A custom role with `audit:read` reads the org-wide audit log (FR-BB114 treats `audit:read` as super_admin-only in the SPA, but the API permission is a plain grant). Recommend adding an explicit sentence to FR-BB117 or extending the non-assignable list (product decision).
- `RoleFormDialog` and `RolesPage` rely on the first integer in a server message; consider returning a structured `count` in the error body.
- Role description max length (500) is enforced but not in the AC.

## 2. PR #204 (ISS-171, self change-password revokes older access tokens)

| # | Requirement | Result | Evidence | Sev |
|---|-------------|--------|----------|-----|
| 1 | FR-BB14 AC-8 / api-conventions section 4: change-password stores a bcrypt cost 12 hash, clears `force_password_change` | PASS | `auth/service.go:233-297`, `repository.go:156-183` | - |
| 2 | Earlier access tokens get 401 `TOKEN_REVOKED` (`password_changed_at` stamped from the app clock, truncated to the second) | PASS by code; middleware check pre-existing (`tokenPredatesPasswordChange`). Second-granularity caveat: an old token issued in the same second also survives | `service.go:272-274`, `self_change_revoke_test.go` | p3 |
| 3 | All refresh tokens revoked in the same tx as the password update | PASS | `repository.go:166-182` | - |
| 4 | Caller stays signed in: fresh access token plus new refresh cookie in the response | PASS. Response shape `{message, access_token, token_type, expires_in}` (FR example updated). Frontend stores the returned token (`api/auth.ts`, `ChangePasswordPage.test.tsx`) | `service.go:276-297`, `handler.go:189-204` | - |
| 5 | Per-user lookup cache invalidated | PASS (`h.passwordChanged(userID)` retained, `handler.go:197`) | - | - |
| 6 | Failure after commit: `issueRefreshCookie`/token issue error returns 500 after the password was already changed and sessions revoked, so the caller is signed out and must log in with the new password | Accepted edge, not specified | `service.go:284-290` | p3 |
| 7 | FR-BB14 AC-12 says weak password gives 422 `VALIDATION_ERROR`; code returns 400 `WEAK_PASSWORD` | GAP (pre-existing, unchanged by PR) | `service.go:250-256` | p3 doc |
| 8 | Docs | UPDATED: FR-BB14 implementation note (was "change-password does not stamp"), FR-BB14 200 example, FR-BB115 revocation note, api-conventions section 4 and `TOKEN_REVOKED` row (the PR's own api-conventions edit was already present) | - | - |

No p1 gaps.

## 3. PR #211 (ISS-181, guarded UNIQUE INDEX on lower(email))

| # | Requirement / rule | Result | Evidence | Sev |
|---|--------------------|--------|----------|-----|
| 1 | Email normalised (trim + lowercase) at create, import, login, forgot-password | PASS (ISS-164, `api.NormalizeEmail`); not previously in api-conventions, added to the `DUPLICATE_EMAIL` row | `users/service.go:163,320`, `auth` | - |
| 2 | Case-insensitive uniqueness enforced by the DB | PASS when no legacy twins exist: `idx_users_email_lower_unique ON users (lower(email))`; migration never fails (count guard plus `EXCEPTION WHEN unique_violation`), never merges or deletes | `migrations/035_...up.sql:11-31` | - |
| 3 | Legacy twins reported | PASS: one WARN with capped groups and one audit_log row `users.duplicate_emails_detected` (system actor, tenant `public`) at each startup while twins exist; check never aborts startup; schema-qualified index lookup (review finding fixed) | `users/duplicate_emails.go`, `cmd/api/main.go` | p3 |
| 4 | Create path maps both the pre-check and a 23505 to 409 `DUPLICATE_EMAIL` | PASS | `users/repository.go:120-148`, `handler.go:160-161` | - |
| 5 | Privacy: full email addresses are written to the WARN log (up to 50 groups) and 3 sample groups to audit metadata | Accepted trade-off from review; not stated in an FR | `duplicate_emails.go:75-92` | p3 |
| 6 | Repeats on every restart and replica while twins exist; no alert/UI | Documented in issue report; ops follow-up | - | p3 |
| 7 | Down migration `DROP INDEX IF EXISTS` | PASS | `035...down.sql` | - |
| 8 | Real-Postgres test | NOT-VERIFIABLE here (gated on `TEST_DATABASE_URL`, label needs-live-db) | `db/unique_email_index_test.go` | - |
| 9 | Docs | UPDATED api-conventions (`DUPLICATE_EMAIL` row). No FR owns the email rule; FR-BB18 (user management) could gain an AC in a later documentation pass | - | p3 doc |

No p1 gaps.

## 4. PR #205 (ISS-195, auto_submitted counted as completed)

| # | Requirement | Result | Evidence | Sev |
|---|-------------|--------|----------|-----|
| 1 | FR-BB51 AC-2 `completed_count` | PASS in code (`submitted`, `auto_submitted`, `grading_pending`); FR text said `submitted` or `grading_pending`, so the FR was stale. Updated | `reports/repository.go:193,909` | - |
| 2 | FR-BB51 AC-4 recent activity | PASS in code (adds `auto_submitted`); FR updated | `repository.go:345` | - |
| 3 | FR-BB51 AC-5 `avg_score_by_track` over the last 90 days | PASS (`status IN ('submitted','auto_submitted')`); FR SQL updated | `repository.go:396` | - |
| 4 | FR-BB51 AC-2 `passed_count` | CHANGED: now requires `status IN ('submitted','auto_submitted')`; FR text updated | `repository.go:195,910` | p3 doc |
| 5 | FR-BB52 AC-5 attempts/participants include `auto_submitted` | PASS (unchanged) | `repository.go:518` | - |
| 6 | FR-BB52 AC-3/AC-4 and score distribution, per-question `correct_rate` | CHANGED by PR: score statistics exclude `grading_pending` (partial scores); `pass_rate` denominator is graded sessions, not `total_attempts`. FR AC-3, AC-4, AC-5 and Notes updated. The long SQL examples in FR-BB52 still show the old filter (not rewritten) | `repository.go:476,508-518,550` | p3 doc |
| 7 | Inconsistency: `avg_time_seconds` and `answer_distribution` for the same exam still include `grading_pending` sessions (`repository.go:589,595`) while `correct_rate` excludes them | Minor inconsistency, arguably correct (answers exist even if not scored) | - | p3 |
| 8 | User progress (FR-BB53): `passed` is `BOOL_OR(...) FILTER` so an exam with only `grading_pending` attempts yields NULL | PASS: `ExamProgress.Passed` is `*bool` (`model.go:189`) | `repository.go:737` | - |
| 9 | Tests | `repository_autosubmitted_test.go` asserts the SQL text and mock results; real-DB aggregate behaviour NOT-VERIFIABLE (needs-live-db) | - | - |
| 10 | `pass_rate` when no graded sessions: SQL yields NULL via `NULLIF`; JSON handling (null vs 0) not confirmed statically | NOT-VERIFIABLE | `repository.go:511-516`, `reports/service.go` | p3 |

No p1 gaps.

## 5. Files changed by this review (docs only)

- `docs/requirements/FR-BB117.Role-management.md` (status, migration 034, shipped deviations, open gaps)
- `docs/requirements/README.md` (FR-BB117 status)
- `docs/requirements/FR-BB14.Authentication.md` (revocation scope, 200 body example)
- `docs/requirements/FR-BB115.Account-recovery.md` (revocation note)
- `docs/requirements/api-conventions.md` (`TOKEN_REVOKED` row, `DUPLICATE_EMAIL` row with email normalisation and index rule)
- `docs/requirements/FR-BB51.Dashboard-metrics-API.md`, `FR-BB52.Per-exam-analytics-API.md` (auto_submitted and grading_pending semantics)
- `docs/uat-scenarios/role-management-20261009.md` (v2: migration 034, PR numbers, 422, custom-role landing, new S7 for scoping and escalation)
- this report

## 6. Not verifiable without a live stack

Migration 034 and 035 execution and down paths; real-Postgres behaviour of `roles` repository (cascade, FK race); keyboard/focus accessibility (AC-17); `pass_rate` null serialisation; same-second token revocation boundary; end-to-end custom-role login and cache propagation (UAT S5); G1/G2 reproduction (UAT S7).
