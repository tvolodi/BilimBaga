---
slug: role-management
title: "Role Management (custom roles and permission matrix) — UAT Scenario"
feature: role-management (FR-BB117; GitHub issue #135)
version: 2
created: 2026-10-09
author: Business Analyst
---

Target: local | qa (default: local; never the production-class demo instance, see DEC-001)

## Code that must be merged before running

- **Issue #135** (FR-BB117), merged as backend PR #185 (`ed22fd5`, migration **034**, not 032) and frontend PR #206 (`147b069`): migration 034, `/api/v1/roles*` endpoints, `GET /users/me` `permissions`, `/admin/roles` page, sidebar item, permission-aware guards, `roles.*` i18n keys.
- Rebuild frontend and API; run `make migrate`.
- Expected pre-fix baseline: no Roles sidebar item; `/admin/roles` falls to not-found/redirect; `GET /api/v1/roles` returns 404. All scenarios FAIL.

## Preconditions and accounts

| Account | Role | Password | Source |
|---------|------|----------|--------|
| `admin@bilimbaga.local` | super_admin | `Admin1234!` (or current) | seeded |
| `uat.deptadmin@test.com` | department_admin | `NewPass123!` | create per `route-guards-20261009.md` S0 |
| `uat.examiner@test.com` | examiner | `NewPass123!` | same (route-guards S0) |
| `uat.employee@test.com` | employee | `NewPass123!` | earlier UAT scenarios |
| `uat.custom@test.com` | custom role `qa_reviewer` | set at S5 | created in S5 |

- Role name used below: `qa_reviewer` (permissions `questions:read`, `exams:read`, `reports:read`). Delete it at the end (S4) so the run is repeatable.
- Fresh browser context per role.

## Scenario S1: Discoverability and access control

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super admin | Log in; view admin sidebar | "Roles" item present (after Users) | |
| 2 | Super admin | Click it | `/admin/roles` opens; table lists the four system roles with a System badge, user counts and permission counts; "Create role" button visible | |
| 3 | Dept admin, examiner | Log in; view sidebar | No Roles item | |
| 4 | Dept admin, examiner | Open `/admin/roles` directly | Redirected away (`/admin`); no data flash | |
| 5 | Employee | Open `/admin/roles` | Redirected to `/portal` | |
| 6 | Dept admin | `GET /api/v1/roles` with token | 403 | |
| 7 | Anonymous | `GET /api/v1/roles` | 401 | |

## Scenario S2: System roles are protected

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super admin | Inspect system rows in the table | No Delete action; "View" (read-only) instead of Edit | |
| 2 | Super admin | Open View on `examiner` | Matrix shown with checkboxes disabled; name disabled | |
| 3 | Tester | `PUT /api/v1/roles/{examiner_id}` with a changed description | 409 `ROLE_SYSTEM_IMMUTABLE`; role unchanged | |
| 4 | Tester | `DELETE /api/v1/roles/{employee_id}` | 409 `ROLE_SYSTEM_IMMUTABLE` | |

## Scenario S3: Create a custom role

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super admin | Click "Create role" | Drawer opens; matrix grouped by resource; `roles:*` and `tenant:manage` not selectable | |
| 2 | Super admin | Enter name `QA Reviewer` | Inline validation error (pattern `^[a-z][a-z0-9_]{2,31}$`), translated | |
| 3 | Super admin | Enter `qa_reviewer`, description, tick `questions:read`, `exams:read`, `reports:read`; save | Toast; role appears in list as Custom, 0 users, 3 permissions | |
| 4 | Super admin | Create `qa_reviewer` again | Inline `ROLE_NAME_TAKEN` message; no duplicate row | |
| 5 | Tester | `POST /api/v1/roles` with `tenant:manage` permission id (repeat with `roles:manage`) | 422 `VALIDATION_ERROR` (shipped status; FR draft said 400) naming the permission | |
| 6 | Super admin | Open `/admin/audit` | `role.create` entry for `qa_reviewer` with actor and metadata | |
| 7 | Super admin | Open Users, "Create user" drawer | Role select includes `qa_reviewer` without a page reload | |

## Scenario S4: Edit, delete and in-use guard

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super admin | Edit `qa_reviewer`: name field disabled; add `tags:read`, save | Count becomes 4; `role.update` audit entry lists `permissions_added: [tags:read]` | |
| 2 | Super admin | Assign `uat.custom@test.com` to `qa_reviewer` (S5 step 1) then try Delete | Confirmation dialog, then error mentioning the user count (`ROLE_IN_USE`, 409); role still listed | |
| 3 | Super admin | Deactivate that user and try Delete again | Still blocked (deactivated users count) | |
| 4 | Super admin | Reassign the user to `employee`, Delete `qa_reviewer` | Confirmation; 204; row disappears; `role.delete` audit entry | |
| 5 | Tester | `DELETE /api/v1/roles/{random-uuid}` | 404 | |

## Scenario S5: Custom role takes effect immediately (cache and guards)

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super admin | Create user `uat.custom@test.com` with role `qa_reviewer`; capture temp password | Created | |
| 2 | Custom user | Log in (complete forced password change if prompted) | Lands in admin shell (`/admin`), not the portal | |
| 3 | Custom user | View sidebar | Only items for held permissions: Dashboard and Reports (both need `reports:read`), Questions, Exams; no Users, Audit, Settings, Roles. Settings can never appear (`tenant:manage` is not grantable) | |
| 4 | Custom user | `GET /users/me` | `permissions` array (sorted `resource:action`) equals the three (or four) assigned | |
| 5 | Custom user | Open `/admin/users` and `/admin/audit` by URL | Redirected to `/admin`, which forwards to the first permitted page (Dashboard here, as `reports:read` is held); direct API calls return 403 | |
| 6 | Super admin | While custom user stays logged in (same token), remove `reports:read` and save | Success | |
| 7 | Custom user | Without re-login, `GET` a reports endpoint (e.g. dashboard metrics) | 403 on the very next request (cache reloaded) | |
| 8 | Super admin | Re-add `reports:read` | Custom user regains access without re-login | |
| 9 | Custom user | Try `GET /api/v1/users` (needs `users:read`) | 403, and no cross-department user data is exposed | |

## Scenario S7: Scoping and escalation limits (added from static conformance review; expected to FAIL until GAP G1/G2 are fixed)

Setup: a second custom role `dept_manager` with `users:read`, `users:manage` (assign `uat.custom2@test.com`, department A). Department A also contains one `department_admin` user and one `super_admin` test user; department B contains an employee.

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Custom2 user | `GET /api/v1/users` | Only department A users; department B employee absent | |
| 2 | Custom2 user | `POST /api/v1/users/{super_admin_in_A}/reset-password` | 403 `FORBIDDEN`, body has no `temporary_password` (D-1, fixed by issue #217; before the fix it returned the temporary password: GAP G2) | |
| 3 | Custom2 user | `PUT /api/v1/users/{department_admin_in_A}` / `POST .../deactivate` / `POST .../unlock` / `POST .../reset-password` | 403 for the same reason (D-1); an employee target whose permissions the caller holds is allowed (reset-password 200) | |
| 4 | Custom user (`qa_reviewer`, `reports:read`) | `GET /api/v1/admin/dashboard` and `/admin/reports/exams/{id}` | Data limited to the caller's department like `department_admin`, or the product explicitly accepts org-wide reports for a granted `reports:read` (decision required; today org-wide: GAP G1) | |
| 5 | Custom2 user | `PUT /api/v1/users/{self}` changing own role | 403 (self role change blocked) | |
| 6 | Custom2 user | Assign role `department_admin` to a department A employee | 403 if `department_admin` holds permissions the caller lacks | |
| 7 | Dept admin A1 | Using `uat.deptadmin` (department A) against a peer `department_admin` A2 in the same department: `POST /api/v1/users/{A2}/reset-password`, `PUT /api/v1/users/{A2}`, `POST /api/v1/users/{A2}/deactivate`, `POST /api/v1/users/{A2}/unlock` | 403 `FORBIDDEN` on each (strict rank hierarchy, FR-BB117 D-1); the response body contains no temporary password; A2 can still log in with the old password | |
| 8 | Dept admin A1 | Same four calls against a `super_admin` user and against an `examiner` and an `employee` in department A | `super_admin` target: 403 on all four. `examiner` and `employee` targets: allowed (200), because they are of strictly lower rank | |
| 9 | Dept admin A1 | Same calls against an `employee` in department B | 403 or 404 (outside own department; no data leaked) | |
| 10 | Custom2 user | Reset password of a peer user holding a custom role whose permissions are NOT a subset of Custom2's | 403; a role with a subset of Custom2's permissions is allowed | |
| 11 | Dept admin A1 | `PUT /api/v1/users/{employee_in_A}` with `role_id` = `department_admin`; then with `role_id` = `super_admin`; then `POST /api/v1/users` creating a user with role `department_admin` | 403 `FORBIDDEN` each (department_admin may assign `examiner` or `employee` only; FR-BB18 AC-4/AC-6, PR #222) | |
| 12 | Dept admin A1 | `PUT /api/v1/users/{employee_in_A}` with `role_id` = `examiner`; then back to `employee` | 200 each | |

### S7 curl recipe (D-1, issue #217)

```bash
API=http://localhost/api/v1
login() { curl -s $API/auth/login -H 'Content-Type: application/json' -d "{\"email\":\"$1\",\"password\":\"$2\"}" | jq -r .data.access_token; }
ADMIN=$(login admin@bilimbaga.local 'Admin1234!')
H() { echo "Authorization: Bearer $1"; }
# 1. custom role: users:read/manage plus portal:* (so the employee target is a permission subset)
curl -s -X POST $API/roles -H "$(H $ADMIN)" -H 'Content-Type: application/json' \
  -d '{"name":"uat_helper","description":"S7","permissions":["users:read","users:manage","portal:read","portal:submit"]}'
ROLE=$(curl -s $API/users/roles -H "$(H $ADMIN)" | jq -r '.data[]|select(.name=="uat_helper").id')
DEPT=<department A id>   # pick from GET $API/departments
# 2. helper user in dept A (note temporary_password; change it at first login), plus targets in dept A:
#    a department_admin user (DA_ID), a super_admin user (SA_ID), an employee user (EMP_ID)
curl -s -X POST $API/users -H "$(H $ADMIN)" -H 'Content-Type: application/json' \
  -d "{\"email\":\"uat.helper@test.com\",\"full_name\":\"UAT Helper\",\"department_id\":\"$DEPT\",\"role_id\":\"$ROLE\"}"
HELPER=$(login uat.helper@test.com '<password after first-login change>')
# 3. expectations
curl -s -o /dev/null -w '%{http_code}\n' -X POST $API/users/$SA_ID/reset-password -H "$(H $HELPER)"      # 403
curl -s -X POST $API/users/$DA_ID/reset-password -H "$(H $HELPER)"                                       # 403 FORBIDDEN, no temporary_password
curl -s -o /dev/null -w '%{http_code}\n' -X POST $API/users/$DA_ID/deactivate -H "$(H $HELPER)"          # 403
curl -s -o /dev/null -w '%{http_code}\n' -X POST $API/users/$EMP_ID/reset-password -H "$(H $HELPER)"     # 200 (employee perms are a subset)
# 4. rank rule (steps 7-8): as a department_admin A1 (DA1 token) against peer DA2, examiner EX_ID, employee EMP_ID
curl -s -X POST $API/users/$DA2_ID/reset-password -H "$(H $DA1)"                                         # 403 FORBIDDEN, no temporary_password
for a in deactivate unlock; do curl -s -o /dev/null -w "$a %{http_code}
" -X POST $API/users/$DA2_ID/$a -H "$(H $DA1)"; done  # 403 403
curl -s -o /dev/null -w '%{http_code}
' -X POST $API/users/$EMP_ID/reset-password -H "$(H $DA1)"        # 200
curl -s -o /dev/null -w '%{http_code}
' -X POST $API/users/$EX_ID/reset-password -H "$(H $DA1)"         # 200
# 5. cleanup: deactivate uat.helper@test.com, DELETE $API/roles/$ROLE
```

## Scenario S6: Localization and accessibility

| Step | Actor | Action | Expected Outcome | Pass/Fail |
|------|-------|--------|-----------------|-----------|
| 1 | Super admin | Switch UI to ru, then kk | Page title, columns, badges, form labels, matrix permission labels, error and delete texts all translated; no raw keys (`roles.` prefix) visible | |
| 2 | Super admin | Keyboard only: Tab to Create role, open drawer, tick matrix boxes with Space, Esc to close | Focus trapped in the drawer, returned on close; every checkbox has an accessible label | |

## Pass / Fail criteria

PASS: S1 to S7 all pass and test data is cleaned (`qa_reviewer` and `uat.custom@test.com` removed or deactivated). DEFECT: any behavioural deviation. REQ GAP: expected behaviour not covered by FR-BB117. ENV ISSUE: stack or seed problems (e.g. migration 034 not applied).

## Acceptance Criteria Coverage

| AC | Scenario |
|----|----------|
| AC-1, AC-2 | S1.6/7, S2 |
| AC-3, AC-6 | S3.2-5 |
| AC-4 | S2.3, S4.1 |
| AC-5 | S2.4, S4.2-5 |
| AC-7 | S3.6, S4.1, S4.4 |
| AC-8 | S5.6-8 |
| AC-9 | S3.7, S5.1 |
| AC-10 | S1.6-7 |
| AC-11 | S1.1-5 |
| AC-12, AC-13, AC-14 | S1.2, S2.1-2, S3, S4 |
| AC-15 | S3.7 |
| AC-16 | S5.2-5, S5.9 |
| AC-17 | S6 |
| Scoping hazards (design notes, no AC) | S7 |
| AC-18 | covered by unit/component test runs, not UAT |

## Out of Scope

Editing system-role permissions, renaming roles, cross-instance cache sync, bulk user reassignment, per-tenant roles.
