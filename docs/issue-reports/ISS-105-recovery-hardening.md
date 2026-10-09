---
id: ISS-105
title: Account recovery hardening (timing, throttle race, Referrer-Policy, token revocation)
status: resolved
severity: medium
layer: backend
module: auth
tags: [forgot-password, user-enumeration, throttle, Referrer-Policy, password_changed_at, TOKEN_REVOKED]
created: 2026-10-09
resolved: 2026-10-09
recurrence_count: 1
related_issues: [ISS-033]
regression_test: backend/internal/auth/hardening_test.go
---

## Symptom
Review follow-ups of FR-BB115 (PR #104, docs/code-reviews/ISS-033-review.md):
1. forgot-password latency differed for known vs unknown emails (enumeration).
2. 3/hour throttle was check-then-insert (race).
3. Reset page (token in URL) had no no-referrer policy.
4. Access tokens issued before a reset stayed valid until expiry.

## Root Cause
1. Known path did purge + count + 2-statement tx + audit insert; unknown path did only a lookup and a hash.
2. `CountRecentResetTokens` and `CreateResetToken` were separate calls without a lock.
3. nginx sent `same-origin`; the page set no meta referrer.
4. Authenticate only verified the JWT signature/expiry.

## Fix Applied
1. Every ForgotPassword path now runs the same repository sequence (GetUserByEmail, token generation+hash, PurgeExpiredResetTokens, IssueResetToken). Unknown/inactive accounts use the nil UUID decoy (IssueResetToken locks no row and stops). The handler pads every response to >= 400 ms (injectable sleep for tests) and writes the response before the audit insert. Mail was already sent in a goroutine. Residual: the DB work on the decoy path is still smaller than the real path (one statement vs four); the constant floor hides this as long as the real path stays under 400 ms.
2. `IssueResetToken` replaces Count+Create: one transaction does `SELECT id FROM users WHERE id=$1 AND status='active' FOR UPDATE`, then COUNT, then invalidate + INSERT. The row lock serialises concurrent requests per user; each later statement gets a fresh READ COMMITTED snapshot so it sees the earlier insert.
3. `<meta name="referrer" content="no-referrer">` set by ResetPasswordPage while mounted (same pattern as VerifyCertificatePage's noindex); `location = /reset-password` block in deploy/nginx.conf sends `Referrer-Policy: no-referrer` (security headers repeated there because add_header is not inherited) plus `Cache-Control: no-store`.
4. Migration 032 adds nullable `users.password_changed_at`. It is stamped by CompleteReset (same tx that revokes refresh tokens) and by admin password reset (users.UpdatePassword). `auth.Authenticate(secret, WithPasswordEpoch(lookup))` rejects tokens whose `iat` is strictly earlier than the stamp (401 TOKEN_REVOKED), tokens without `iat`, and unknown users; lookup errors fail closed (500). The lookup is cached per user for 10 s (bounded map), so revocation takes effect within 10 s on a running instance. The user's own change-password does not stamp (the current session is intentionally kept).

## Files Changed
| File | Change |
|------|--------|
| backend/migrations/032_users_password_changed_at.{up,down}.sql | new column |
| backend/internal/auth/recovery_repository.go | IssueResetToken; CompleteReset stamps password_changed_at |
| backend/internal/auth/recovery_service.go | uniform ForgotPassword sequence |
| backend/internal/auth/recovery_handler.go, handler.go | constant-time padding, audit after response |
| backend/internal/auth/password_epoch.go, middleware.go | epoch lookup + cache + Authenticate option |
| backend/internal/router/router.go | wires epoch lookup when a DB is present |
| backend/internal/users/repository.go | admin reset stamps password_changed_at |
| deploy/nginx.conf | /reset-password no-referrer |
| frontend/src/pages/auth/ResetPasswordPage.tsx | referrer meta |

## Regression Test
backend/internal/auth/hardening_test.go (sequence equality, padding, concurrency 40 goroutines <= 3, SQL lock order via tx-aware fake driver, epoch middleware, cache), backend/internal/router/nginx_conf_test.go, frontend/src/pages/auth/recovery.test.tsx.

## Resolution Results
- Tests: Go, frontend suites green (see PR)
- Migration applied: no (needs-live-db; no Postgres here)
- Build clean: yes

## Recurrence Log
| Date | Trigger | Action Taken |
|------|---------|--------------|

Note: user-initiated change-password intentionally does not stamp password_changed_at (current session kept; other sessions of that user stay valid until expiry). Reset and admin reset stamp with the DB clock (now()). IssueResetToken uses an explicit READ COMMITTED transaction.
