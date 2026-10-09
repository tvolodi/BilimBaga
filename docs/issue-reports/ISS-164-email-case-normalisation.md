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

## Cycle 2 (supervisor decision): legacy mixed-case rows

Cycle 1 assumed no mixed-case rows exist. The production-class instance may hold legacy ones, which the lowercased input would never match with `WHERE email = $1`. Changes (no migration, no index; migration lock held):

- `auth.pgRepository.GetUserByEmail` (login + forgot-password): `WHERE lower(u.email) = $1` with the already-normalised input.
- `auth.pgBootstrapStore.GetAdminCredentials`: `WHERE lower(email) = lower($1)`.
- `users.pgRepository.Create`: the table's UNIQUE(email) is case-sensitive, so a legacy `John@X.com` would not block `john@x.com`. Added a pre-check `SELECT EXISTS (SELECT 1 FROM users WHERE lower(email) = lower($1))` returning ErrDuplicateEmail (also covers CSV import, which goes through Create). The UNIQUE constraint remains the backstop for concurrent inserts (small race window, mixed-case twin only).
- Other email lookups (`internal/email`) are by id/status, not by address.

Tests: `auth/email_sql_test.go` (recording fake driver asserts exact SQL and bound param), `users/email_sql_test.go` (probe SQL; no INSERT when twin exists), service tests with stored `John.Doe@Corp.com` for login, forgot-password and create-duplicate. Fakes now mirror the lower() semantics. schemaguard green.

Follow-up (needs migration): `CREATE INDEX ... ON users (lower(email))` (ideally UNIQUE, after de-duplicating legacy rows), otherwise lower() lookups cannot use idx_users_email and seq-scan (fine at current table sizes).

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|-------------- |
