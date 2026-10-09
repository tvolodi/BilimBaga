---
id: ISS-133
title: Users list - filter by role returns nothing / errors
status: resolved
severity: high
layer: frontend
module: users
tags: [role_id, invalid input syntax for type uuid, UsersListPage, users.List]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: []
regression_test: frontend/src/pages/admin/users/UsersListPage.test.tsx
---

## Symptom
Customer complaint #7 item 1 (GH #133): choosing a role in the Users list "Filter by role" does not filter.

## Root Cause
`UsersListPage.tsx` hardcoded the role `<option>` values to role NAMES (`super_admin`, `examiner`, ...) and sent them as `?role_id=examiner`. The backend (`users.Handler.ListUsers` -> `pgRepository.List`) treats `role_id` as a UUID: `AND u.role_id = $n`. Postgres rejects the non-UUID text ("invalid input syntax for type uuid"), the repository error surfaces as a 500 and the list fails. The roles endpoint (`GET /api/v1/users/roles`, `useRoles`) already exposed real ids and was used by the create/edit drawers, but not by the filter. Hardcoding also hid any custom role.

## Fix Applied
- Frontend: the filter options are built from `useRoles()`; value = role id, label = `users.roles.<name>` i18n with the raw role name as fallback.
- Backend hardening: `ListUsers` handler validates `role_id` as a UUID and returns 422 VALIDATION_ERROR instead of a 500 from the DB. No SQL change.

## Files Changed
| File | Change |
|------|--------|
| frontend/src/pages/admin/users/UsersListPage.tsx | role filter driven by useRoles (ids) |
| backend/internal/users/handler.go | uuid validation of role_id -> 422 |
| frontend/src/pages/admin/users/UsersListPage.test.tsx | regression test |
| backend/internal/users/handler_test.go, service_test.go | handler + service tests |

## Regression Test
UsersListPage.test.tsx "sends the selected role UUID (not the role name) as role_id and clears it"; handler tests for UUID pass-through and 422 on non-UUID; service test for role filter.

## Resolution Results
- Tests: backend `go test -p 2 ./...` all ok; frontend vitest 474 passed, 0 failed
- Migration applied: no
- Build clean: yes (go build/vet, tsc)
- Live-DB verification: not run (SQL unchanged; UAT should confirm against the local stack)

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
