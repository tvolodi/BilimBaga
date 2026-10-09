---
id: ISS-164
title: Mixed-case email - login and forgot-password fail because only user creation lowercases the address
status: resolved
severity: high
layer: backend
module: auth
tags: [email, lowercase, INVALID_CREDENTIALS, GetUserByEmail, forgot-password, NormalizeEmail]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-105]
regression_test: backend/internal/auth/email_case_test.go
---

## Symptom
User created as `uat.MixedCase@Test.com` is stored lowercased. POST /auth/login with the mixed-case form returns 401 INVALID_CREDENTIALS; POST /auth/forgot-password returns the neutral 200 but sends no mail.

## Root Cause
`users.CreateUser`/import lowercase the email, but `auth.Login` and `auth.ForgotPassword` passed the address to `GetUserByEmail` (`WHERE u.email = $1`) exactly as typed (forgot-password only trimmed). Reset-password is token-based and unlock/deactivate are by user id, so they are not email lookups.

## Fix Applied
Added the shared leaf helper `api.NormalizeEmail` (trim + lowercase) in `backend/internal/api/email.go` and applied it at the service boundary in: `auth.Login`, `auth.ForgotPassword`, `users.CreateUser` (normalise before validation), and `users.ImportUsers` (once per row before validate/commit, so previews show the stored form). No SQL change: all writers have lowercased since FR-BB18 and the only seeded user (`admin@bilimbaga.local`) is lowercase, so no mixed-case legacy rows exist. Follow-up (needs a migration, lock held by another dev): a `lower(email)` unique index / CHECK (`email = lower(email)`) would enforce this in the DB.

## Files Changed
| File | Change |
|------|--------|
| backend/internal/api/email.go (+_test) | new NormalizeEmail helper |
| backend/internal/auth/service.go | Login normalises email |
| backend/internal/auth/recovery_service.go | ForgotPassword uses NormalizeEmail |
| backend/internal/users/service.go | Create + Import use NormalizeEmail |
| backend/internal/auth/email_case_test.go | new regression tests |
| backend/internal/users/service_test.go | create/import normalisation tests |

## Regression Test
`backend/internal/auth/email_case_test.go`: login (service + handler) with mixed case/whitespace against an exact-match repo, forgot-password mixed-case issues token+mail, unknown mixed-case stays neutral. Users: create and CSV import store normalised email.

## Resolution Results
- Tests: full `go test -p 1 ./...` passed, 0 failed
- Migration applied: no
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
