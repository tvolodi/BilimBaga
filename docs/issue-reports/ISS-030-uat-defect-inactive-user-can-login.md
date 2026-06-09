---
id: ISS-030
title: "UAT Defect: User Onboarding — Inactive user can authenticate and access portal"
status: resolved
severity: high
layer: backend
module: auth
tags: [uat, auth, user-management, security, ACCOUNT_INACTIVE]
created: 2026-06-09
resolved: 2026-06-09
recurrence_count: 1
related_issues: []
regression_test: backend/internal/auth/handler_test.go::TestService_Login_InactiveUser_Returns401
---

## Symptom
UAT Scenario: `User Onboarding — UAT Scenario`, Scenario 2, Step 5
Actor: Employee
Action: Attempt to log in with `uat.employee@test.com` / `NewPass123!` after the account was deactivated (status set to `inactive` via Admin UI)
Expected: Login is rejected; an error message is shown (e.g. "Account is inactive")
Actual: Login succeeded. `POST /api/v1/auth/login` returned a valid JWT. Employee was redirected to `/portal` and had full access. DB confirms `users.status = 'inactive'` at the time of login.
Screenshot: test-results/uat-user-onboarding-202606-4c3a7-activate-User-Login-Blocked-chromium-uat/test-failed-1.png

## Root Cause
`auth.service.Login` (`backend/internal/auth/service.go`) fetched the user record and checked only for account lockout before proceeding to password verification and JWT issuance. It never inspected `users.status`. The `User` model already carries a `Status string` field populated from the DB, but the login flow did not gate on it. Any user with a correct password — regardless of whether their account was `inactive` or any other non-`active` status — received a valid JWT and refresh cookie.

## Fix Applied
Added a status guard immediately after the lockout check (AC-5) and before password verification in `auth.service.Login`. If `user.Status != "active"`, the method returns a `ServiceError` with code `ACCOUNT_INACTIVE` and HTTP 401. Placing it after the lockout check preserves the priority order (lockout is a stricter condition) while ensuring inactive accounts are rejected before any bcrypt work is done.

Two pre-existing mock-repository tests (`TestService_Login_LockedAccount_Returns423BeforePasswordCheck` and `TestService_Login_FifthFailureLocks`) had their mock `User` structs missing `Status: "active"`, which caused them to inadvertently hit the new inactive check. Those mocks were updated to set `Status: "active"` to reflect the intended active-user scenarios they test.

## Files Changed
| File | Change |
|------|--------|
| `backend/internal/auth/service.go` | Added `user.Status != "active"` guard in `Login`, returning `ACCOUNT_INACTIVE` / HTTP 401 |
| `backend/internal/auth/handler_test.go` | Added `TestService_Login_InactiveUser_Returns401` regression test; added `Status: "active"` to two existing mock users; added `golang.org/x/crypto/bcrypt` import |

## Regression Test
`backend/internal/auth/handler_test.go` — `TestService_Login_InactiveUser_Returns401`

Uses a mock repository that returns a user with `Status: "inactive"` and a valid bcrypt password hash. Asserts that `svc.Login` returns a `*ServiceError` with `Code == "ACCOUNT_INACTIVE"` and `HTTPStatus == 401`, and that both the `LoginResponse` and `*http.Cookie` are `nil`.

## Resolution Results
- Tests: all packages pass (23 packages, 0 failures)
- Migration applied: no (no schema change needed; `users.status` column already exists)
- Build clean: yes (`go build ./...` exits 0)
