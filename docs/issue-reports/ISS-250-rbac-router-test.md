---
id: ISS-250
title: Missing router-level real-RBAC test for users authz D-4 (follow-up of ISS-240 / PR #246)
status: resolved
severity: low
layer: backend
module: users
tags: [router, rbac, TOKEN_REVOKED, strict-subset, 404-parity, RequirePermission]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-240, ISS-248, ISS-141]
regression_test: backend/internal/router/users_authz_test.go
---

## Symptom
D-4 rules were only tested at service/handler level; no test drove the real route tree
(auth.Authenticate + account-state cache + rbac.RequirePermission + users.Handler).

## Root Cause
Test gap (conformance report PR222-PR227-PR231, "Missing" item 8). No production bug found.

## Fix Applied
Tests only. Router built with real rbac.Cache, real users.Service/Handler over an in-memory
users.Repository, and a recording database/sql driver serving the account-state query so the
real AccountStateCache runs. Cases: strict-subset peer refused (update/create/manage), examiner
RBAC 403 on users:manage routes, no self role change / self deactivate (super_admin and
department_admin), 404 parity unknown vs out-of-scope across 5 operations, malformed UUID
(path 404, body/query 422), TOKEN_REVOKED for demoted/deactivated/moved claims, and ISS-248
immediate revocation after a role change. Frontend: UsersListPage.roles.test.tsx asserts the
routed page's create/edit drawers offer only allowed roles for department_admin.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/router/users_authz_test.go | new router-level test |
| frontend/src/pages/users/UsersListPage.roles.test.tsx | new page-level test |

## Regression Test
See above.

## Resolution Results
- Tests: backend go test -p 2 ./... all ok; frontend vitest 88 files, 628 tests passed
- Migration applied: no
- Build clean: yes
