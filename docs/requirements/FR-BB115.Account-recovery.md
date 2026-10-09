# FR-BB115 — Account Recovery: Forgot Password and Lockout Unlock

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB115 |
| Phase | 1 — Foundation (gap closure; completes roadmap 1.4 and 6.1) |
| Priority | 1 |
| Status | Implemented |
| Depends On | FR-BB14, FR-BB18, FR-BB61, FR-BB64, FR-BB110, FR-BB62, FR-BB48 (`PUBLIC_APP_URL`) |

## Description
Roadmap 1.4 requires that a locked account can be "unlocked via admin or email link", and roadmap 6.1 lists a "password reset link" email template. Neither exists today: FR-BB14 and FR-BB110 explicitly defer forgot-password and unlock, FR-BB61 deliberately deviates to a temporary-password email, the router has no forgot/reset/unlock route, and the login screen has no "Forgot password" entry. Consequence: after 5 failed attempts (30-minute lock, `auth/service.go`) or a forgotten password, a user cannot recover without a super admin acting out-of-band, and an admin has no UI/API to clear a lock early. This requirement adds (1) a self-service, token-based forgot/reset-password flow delivered by email, and (2) an admin unlock action, both audited and non-enumerating.

## Acceptance Criteria
- [ ] AC-1: `POST /api/v1/auth/forgot-password` (public, `AuthLimiter`) accepts `{ "email": "..." }` and ALWAYS returns `200 { "data": { "message": "..." }, "error": null }` with identical body and similar latency whether or not the email belongs to an active user (no account enumeration). Malformed or missing email returns 422 `VALIDATION_ERROR` (aligned from 400 by PR #197).
- [ ] AC-2: For an existing active user, a single-use reset token (>= 32 bytes from `crypto/rand`, URL-safe) is generated; only its SHA-256 hash is stored in a new table `password_reset_tokens (id UUID PK, user_id UUID FK, token_hash TEXT UNIQUE, expires_at TIMESTAMPTZ, used_at TIMESTAMPTZ NULL, created_at TIMESTAMPTZ)` via a new numbered migration (with `.down.sql`); expiry is 60 minutes; requesting a new token invalidates earlier unused tokens for that user. Inactive users receive no email and no token.
- [ ] AC-3: The user receives an email (plain text + HTML, rendered in the user's `preferred_locale`, locales `en`/`ru`/`kk`) containing the link `{PUBLIC_APP_URL}/reset-password?token=<token>`; the token never appears in logs, audit metadata, or API responses. Send failures are logged without the token and do not change the API response.
- [ ] AC-4: `POST /api/v1/auth/reset-password` (public, `AuthLimiter`) accepts `{ "token", "new_password" }`; on a valid, unexpired, unused token it validates the password against the same policy as `change-password`, stores a bcrypt hash (cost from `Config.BcryptCost`, default 12), marks the token used, sets `force_password_change = false`, resets `failed_attempts = 0` and `locked_until = NULL`, and revokes all existing refresh tokens of that user; returns 200. An invalid, expired or used token returns 400 `INVALID_TOKEN` with one generic message for all three cases; a policy-violating password returns 400 `VALIDATION_ERROR` and does NOT consume the token.
- [ ] AC-5: `POST /api/v1/users/{id}/unlock` (permission `users:manage`, the same key used by `reset-password`; `super_admin` may unlock any user, a `department_admin` only users in their own department, otherwise 403; no `users:update` key exists) sets `locked_until = NULL` and `failed_attempts = 0` for the target user, returns the updated user, and returns 404 for an unknown id. A subsequent login with the correct password succeeds without waiting for lock expiry.
- [ ] AC-6: Every step writes an `audit_log` entry: `auth.password_reset_requested` (actor = user if resolvable, no token), `auth.password_reset_completed`, `auth.password_reset_failed` (invalid token; no token value), and `users.unlock` (actor = admin, target = user id). Requests for unknown emails write no entry that reveals existence beyond a generic rate-limit key.
- [ ] AC-7: Frontend: the login page shows a localized "Forgot password?" link to `/forgot-password` (email form, neutral confirmation message after submit, regardless of outcome) and a public `/reset-password` page (reads `token` from the query string; new password + confirm with the same client-side rules as the change-password screen; success redirects to `/login` with a success notice; invalid/expired token shows an error with a link back to `/forgot-password`). Both routes are outside `RequireAuth`, send no `Authorization` header, and contain no admin/portal navigation. All strings exist in `en`, `ru`, `kk` under `auth.recovery.*` (no hardcoded strings).
- [ ] AC-8: Frontend admin: the user list/detail in the admin area shows an "Unlock account" action only for users whose `locked_until` is in the future; it calls the unlock endpoint through React Query, shows a localized success toast, and refreshes the row. The `GET /api/v1/users` response exposes `is_locked` (boolean, computed server-side) for this purpose.
- [ ] AC-9: Security: reset tokens are compared by hash with constant-time semantics (lookup by hash); `forgot-password` is additionally throttled to 3 requests per user per hour (atomic: row lock on the user, count then insert in one transaction; unknown emails have no row to throttle and are limited only by `AuthLimiter`) (silently, still returning the generic 200); a reset response never differs between "user not found" and "token reused" beyond the generic `INVALID_TOKEN`; expired tokens are purged by the existing background job runner or on each request (no unbounded table growth).
- [ ] AC-10: Tests: backend `service_test.go` + `handler_test.go` cover generic-200 for unknown email, token hash storage, 60-minute expiry, single use, invalidation of prior tokens, policy failure not consuming the token, lock clearing on reset, admin unlock (success/404/forbidden), and audit entries; frontend Vitest + RTL cover both pages (success, invalid token, policy error) and the admin unlock action; a live E2E spec covers lock -> admin unlock -> login and forgot -> reset (token read from Mailhog at `HOST_MAILHOG` in the Docker stack) -> login.

## Technical Specification

### Database Schema
New migration (next free number after the highest in `backend/migrations/`; never edit existing ones):
```sql
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens (user_id);
```

### API Contract
| Method | Path | Auth | Request | Success | Errors |
|--------|------|------|---------|---------|--------|
| POST | `/api/v1/auth/forgot-password` | public | `{ "email" }` | 200 generic message | 422 `VALIDATION_ERROR`, 429 `RATE_LIMITED` |
| POST | `/api/v1/auth/reset-password` | public | `{ "token", "new_password" }` | 200 `{ "data": { "message": "..." }, "error": null }` | 400 `INVALID_TOKEN`, 422 `VALIDATION_ERROR`, 429 |
| POST | `/api/v1/users/{id}/unlock` | `users:manage` (dept-scoped) | none | 200 user object | 403, 404 |

All responses use the standard `{ data, error }` envelope; IDs UUID v4; timestamps UTC ISO 8601.

### Go Implementation Notes
- Package `auth`: `ForgotPassword`, `ResetPassword` in the service layer; token repository methods in the repository layer; handlers thin. Reuse the password-policy validator and bcrypt helper from `change-password`.
- Package `users`: `Unlock` service method + handler; add `is_locked` to the list/detail projection.
- Package `email`: add template `password_reset_link` (text + HTML, three locales) and `TriggerPasswordResetLink(userID, token)`; the link base comes from `Config.PublicAppURL` (FR-BB48); no `os.Getenv` in handlers. The existing temporary-password email for admin reset (FR-BB61) is unchanged.
- Router: register the two public auth routes under the `AuthLimiter` group and the unlock route beside `reset-password`.
- RBAC: reuse `users:manage` (decision 2026-10-09, #33); department-scope check as for `reset-password`; no new permission key or migration for RBAC.

### Frontend Implementation Notes
- Files: `src/pages/ForgotPasswordPage.tsx`, `src/pages/ResetPasswordPage.tsx`, `src/api/recovery.ts` (React Query mutations using plain unauthenticated fetch), unlock mutation in the users API module; shadcn/ui `Card`, `Input`, `Button`, `Alert`.
- Routes `/forgot-password` and `/reset-password` registered outside auth wrappers in `App.tsx`; mobile-first (>= 375 px).

## Out of Scope
- Email-link self-service unlock distinct from password reset (a successful reset clears the lock; admin unlock covers the rest).
- MFA, SSO, password history or expiry policies.
- SMS-based recovery.

## Notes
- Closes the deliberate deviation recorded in FR-BB61 ("token-based password reset links deferred") and the deferrals in FR-BB14 and FR-BB110.
- Requires a working SMTP configuration (FR-BB61); in the Docker dev stack Mailhog captures messages for E2E verification.

- Implementation note (FR-BB115): the admin unlock route uses the existing RBAC key `users:manage` (same as `reset-password`); no `users:update`/`users:unlock` key exists, so no RBAC migration was needed. Department admins are additionally scoped to their own department (same as reset-password). Policy failure on reset returns `VALIDATION_ERROR` (per AC-4), not the change-password `WEAK_PASSWORD` code.

- Implementation note (ISS-105 / PR #144, hardening beyond AC-1..AC-10):
  - Response-time equalization: `forgot-password` pads every response to a 400 ms floor and runs a decoy path (nil-UUID lookup, token generation and hash) for unknown/inactive/throttled emails; mail is sent asynchronously and the audit write happens after the response. The floor is a minimum only; under DB load above 400 ms known and unknown may again differ.
  - `Referrer-Policy: no-referrer` and `Cache-Control: no-store` are sent on the exact path `/reset-password` by the frontend nginx (`deploy/nginx.conf`); the page also sets `<meta name="referrer" content="no-referrer">` while mounted as a client-side fallback.
  - Access-token revocation: `CompleteReset` stamps `users.password_changed_at` (migration 032); the auth middleware rejects access tokens whose `iat` is earlier with 401 `TOKEN_REVOKED`. The per-user lookup is cached ~10 s per process, so an old access token may survive up to ~10 s; comparison is at second resolution; a DB error in the lookup returns 500 `INTERNAL_ERROR` (fail closed). Migration 032 must ship with the code. Since PR #204 (ISS-171) the self `change-password` endpoint stamps the same column and revokes refresh tokens too (see FR-BB14 note), so all three password-change paths share one revocation rule.
  - Expired reset tokens are purged on each forgot request once expired more than 24 h (so the 1 h throttle window always sees them).
