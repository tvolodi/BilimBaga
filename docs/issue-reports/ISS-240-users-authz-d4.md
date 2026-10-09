---
id: ISS-240
title: Users authz hardening D-4 - peer custom roles manageable, self role/deactivate, stale JWT claims, 500 on malformed ids, ambiguous import department
status: resolved
severity: high
layer: backend
module: users
tags: [FR-BB117, D-4, permissionSubset, claimsStale, AccountState, GetDepartmentIDByName, ErrAmbiguousName, TOKEN_REVOKED]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-217, ISS-229, ISS-105, ISS-160]
regression_test: backend/internal/users/authz_d4_test.go
---

## Symptom
GitHub #240 (conformance gaps G1-G4, G7, G8 of PR222/227/231): a custom role with exactly the caller's permissions was manageable/assignable; super_admin could change its own role and any caller could deactivate itself; unknown ids answered 404 but out-of-scope ids 403; a demoted/moved/deactivated user kept old role, department and API access for up to 15 minutes (JWT claims trusted, per-request lookup only checked the password epoch); malformed role_id/department_id reached Postgres (500); CSV import resolved department names with LIMIT 1 and no ambiguity check, twice (validate, commit).

## Root Cause
`permissionSubset` was inclusive; self checks exempted super_admin and had no deactivate guard; scope failures returned ErrForbidden; `AccountState` held only password epoch and force flag; repository wrote ids straight into uuid columns; `GetDepartmentIDByName` used `LIMIT 1`.

## Fix Applied
- users/service.go: strict proper-subset rule (caller must hold at least one permission the role lacks); self role change and self deactivate forbidden for everyone; out-of-scope target answers ErrNotFound on get/update/deactivate/reset/unlock; import resolves department and role once per row (`validateImportRow` returns ids, `commitImportRow` reuses them) and reports ambiguity as a row error.
- users/repository.go: `GetDepartmentIDByName` fetches up to 2 rows and returns `ErrAmbiguousName`; malformed UUIDs: `GetByID`/`GetRoleNameByID` return ErrNotFound (404 / 422 role not found), `Create`/`Update` return ErrValidation (422) before SQL.
- auth: `AccountState` now carries Status, RoleName, DepartmentID (query joins roles); `Authenticate` answers 401 TOKEN_REVOKED when status != active or role/department differ from the token. Rides the existing 10 s cache, so effect is within 10 s of the change.
- Unrelated blocker fixed (Unblock-Everything): auth tests hard-coded a 2026-10-09 12:00 UTC clock and failed once the real clock passed 12:15; they now anchor to `time.Now()`.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/users/service.go, repository.go, types.go | rules above |
| backend/internal/auth/password_epoch.go, middleware.go | identity check in account-state lookup |
| backend/internal/users/*_test.go, authz_d4_test.go | updated expectations (403->404 out-of-scope, strict subset), new D-4 tests |
| backend/internal/auth/stale_claims_test.go, force_password_change_test.go, self_change_revoke_test.go | stale-claims tests, clock fix |
| docs/requirements/FR-BB117.Role-management.md | D-4 shipped behaviour note |

## Regression Test
backend/internal/users/authz_d4_test.go (strict subset on check/create/update/import, self role/deactivate incl. super_admin, 404 parity, no-department, inactive targets, malformed ids, ambiguity, handler status mapping); backend/internal/auth/stale_claims_test.go (demotion, move, deactivation, cache TTL).

## Resolution Results
- Tests: `go test -p 2 ./...` all packages ok; `go vet ./...` clean; internal/schemaguard ok
- Migration applied: no
- Build clean: yes
- Not covered by a real-Postgres test: the new account-state JOIN query and the 2-row department query (recorded-driver tests only) - PR labelled needs-live-db.

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
