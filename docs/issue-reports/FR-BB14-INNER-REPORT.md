# FR-BB14 — Authentication — Inner Report

**Date**: 2026-05-14
**Pipeline**: A — Feature Development
**Status**: COMPLETE

---

## Summary

Implemented the full authentication lifecycle for BilimBaga: credential validation, JWT access token issuance, httpOnly refresh-cookie rotation, logout, forced password change, account lockout after repeated failures, and audit logging for all events.

---

## Files Created / Modified

### New Files
| File | Description |
|------|-------------|
| `backend/migrations/003_roles_departments.up.sql` | Stub `roles` (seeded with 4 roles) and `departments` tables — FK targets for `users` |
| `backend/migrations/003_roles_departments.down.sql` | Rollback for 003 |
| `backend/migrations/004_auth.up.sql` | `users`, `refresh_tokens`, `audit_log` tables with all indexes |
| `backend/migrations/004_auth.down.sql` | Rollback for 004 |
| `backend/internal/auth/model.go` | Domain types: `User`, `RefreshToken`, `LoginRequest/Response`, `RefreshResponse`, `UserInfo`, `ChangePasswordRequest`, `Claims`, `ServiceError` |
| `backend/internal/auth/repository.go` | `pgRepository` implementing all DB operations: user lookups, failed-attempt tracking, refresh token lifecycle, audit log writes |
| `backend/internal/auth/service.go` | Business logic: `Login`, `Refresh`, `Logout`, `ChangePassword`, `ParseAccessToken` + private helpers |
| `backend/internal/auth/handler.go` | HTTP handlers wired to service; `writeJSON`, `handleServiceError`, `clientIP` helpers |
| `backend/internal/auth/handler_test.go` | 23 tests covering all handlers, service logic, and helpers |

### Modified Files
| File | Change |
|------|--------|
| `backend/internal/router/router.go` | Added `authHandler *auth.Handler` parameter; registered 4 auth routes |
| `backend/cmd/api/main.go` | Wired up `auth.NewRepository`, `auth.NewService`, `auth.NewHandler`; passed to router |
| `backend/go.mod` / `go.sum` | Added `github.com/golang-jwt/jwt/v5 v5.3.1` and upgraded `golang.org/x/crypto` |

---

## Acceptance Criteria Verification

| AC | Description | Verified By |
|----|-------------|-------------|
| AC-1 | `users` table created | `004_auth.up.sql` |
| AC-2 | `refresh_tokens` table created | `004_auth.up.sql` |
| AC-3 | `POST /auth/login` returns JWT + httpOnly cookie | `TestLogin_ValidCredentials_Returns200WithTokenAndCookie` |
| AC-4 | Failed login increments `failed_attempts`; 5 failures → lock 30 min | `TestService_Login_FifthFailureLocks` |
| AC-5 | Locked account returns 423 regardless of password | `TestService_Login_LockedAccount_Returns423BeforePasswordCheck`, `TestLogin_LockedAccount_Returns423` |
| AC-6 | `POST /auth/refresh` validates cookie, rotates pair | `TestRefresh_ValidCookie_Returns200WithNewToken` |
| AC-7 | `POST /auth/logout` returns 200 even without cookie | `TestLogout_WithoutCookie_Returns200`, `TestLogout_WithCookie_Returns200AndClearsCookie` |
| AC-8 | `POST /auth/change-password` validates current, hashes new (bcrypt 12), clears flag | `TestChangePassword_Valid_Returns200`, `TestChangePassword_WrongCurrentPassword_Returns400` |
| AC-9 | bcrypt cost 12 (configurable, default 12); never logged/returned | Service code — `bcrypt.GenerateFromPassword` with `cfg.BcryptCost` |
| AC-10 | All auth events written to `audit_log` | `service.writeAudit` called in all auth paths; `audit_log` table in migration 004 |
| AC-11 | Revoked token triggers revocation of ALL user tokens → 401 | `TestRefresh_RevokedToken_Returns401`; `RevokeAllUserRefreshTokens` call in service |
| AC-12 | `new_password` < 8 chars → 422 VALIDATION_ERROR | `TestChangePassword_ShortNewPassword_Returns422`, `TestService_ChangePassword_ShortPassword_Returns422` |

---

## Test Results

```
ok  github.com/bilimbaga/bilimbaga/internal/auth    0.759s   (23 tests)
ok  github.com/bilimbaga/bilimbaga/internal/config  0.488s
ok  github.com/bilimbaga/bilimbaga/internal/db      0.787s
ok  github.com/bilimbaga/bilimbaga/internal/health  0.704s
ok  github.com/bilimbaga/bilimbaga/internal/tenant  0.742s
```

**No regressions. All 23 new tests pass.**

---

## Design Decisions

1. **Migration numbering**: The requirement specified `007_auth.up.sql` but migrations 003-006 did not exist. Used `003_roles_departments` and `004_auth` as the next sequential numbers after 002.

2. **`roles` and `departments` stubs**: Required as FK targets for `users`. Full implementations come in FR-BB16 and FR-BB17. Roles are pre-seeded with the 4 canonical values.

3. **`audit_log` in migration 004**: The requirement depends on FR-BB19 (Audit Log), but since `audit_log` is needed by FR-BB14 for AC-10, it was created in the same migration as `users` (since `audit_log.user_id` references `users(id)`).

4. **Refresh token security**: Cookie carries a base64url-encoded 32-byte random value. DB stores its SHA-256 hex hash. This prevents DB compromise from yielding valid tokens.

5. **Timing attack mitigation**: When a user is not found, a dummy bcrypt comparison runs to prevent user-enumeration via response timing.

6. **JWT parsing in handler**: `change-password` requires Bearer auth. Since FR-BB15 (JWT middleware) is not yet implemented, the handler calls `svc.ParseAccessToken()` directly. This will be replaced by middleware when FR-BB15 is delivered.
