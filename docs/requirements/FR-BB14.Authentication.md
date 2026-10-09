# FR-BB14 — Authentication

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB14 |
| Phase | 1 — Foundation |
| Priority | 1 |
| Status | Ready |
| Depends On | FR-BB12, FR-BB16, FR-BB17, FR-BB19 |

## Scope

| Layer | Artifact |
|-------|----------|
| Migration | `migrations/007_auth.up.sql`, `migrations/007_auth.down.sql` |
| Backend | `internal/auth/` (handler, service, repository) |
| Router | `internal/router/` (auth routes) |

## Description
Implements the full authentication lifecycle: credential validation, JWT access token issuance, httpOnly refresh-cookie rotation, logout, and forced password change. Account lockout after repeated failures protects against brute-force attacks. All authentication events — successful logins, failures, logouts, and password changes — are recorded in the `audit_log` for compliance and forensics.

## Acceptance Criteria
- [ ] AC-1: Migration creates the `users` table with columns: `id UUID PK`, `email TEXT UNIQUE NOT NULL`, `password_hash TEXT NOT NULL`, `full_name TEXT NOT NULL`, `department_id UUID NULLABLE`, `role_id UUID NOT NULL`, `status TEXT NOT NULL DEFAULT 'active'`, `force_password_change BOOLEAN NOT NULL DEFAULT false`, `failed_attempts INT NOT NULL DEFAULT 0`, `locked_until TIMESTAMPTZ NULLABLE`, `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`, `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`.
- [ ] AC-2: Migration creates the `refresh_tokens` table with columns: `id UUID PK`, `user_id UUID NOT NULL REFERENCES users(id)`, `token_hash TEXT NOT NULL`, `expires_at TIMESTAMPTZ NOT NULL`, `revoked_at TIMESTAMPTZ NULLABLE`, `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`.
- [ ] AC-3: `POST /api/v1/auth/login` with valid credentials returns an access token (JWT, 15-minute TTL) in the JSON body and sets a `refresh_token` httpOnly, Secure, SameSite=Strict cookie (7-day TTL).
- [ ] AC-4: `POST /api/v1/auth/login` with invalid credentials increments `failed_attempts`; after 5 consecutive failures the account is locked by setting `locked_until = now() + 30 minutes` and a `ACCOUNT_LOCKED` error is returned.
- [ ] AC-5: `POST /api/v1/auth/login` against a locked account always returns `423 Locked` with `ACCOUNT_LOCKED` error, regardless of whether the supplied password is correct.
- [ ] AC-6: `POST /api/v1/auth/refresh` validates the `refresh_token` cookie against the hashed value in `refresh_tokens`; on success, issues a new access token and rotates the cookie (old record is revoked, new record is inserted).
- [ ] AC-7: `POST /api/v1/auth/logout` revokes the refresh token by setting `revoked_at = now()` and clears the cookie; returns `200 OK` even if no cookie was present.
- [ ] AC-8: `POST /api/v1/auth/change-password` requires the caller to supply their current password, validates it, hashes the new password with bcrypt cost 12, stores it, and sets `force_password_change = false`. Amended 2026-10-09 (BA, PR #180 / ISS-160): while `force_password_change` is true the API itself enforces the change: every authenticated route except `POST /auth/change-password` and `GET /users/me` answers `403 PASSWORD_CHANGE_REQUIRED` (`auth/middleware.go:113-115`); the flag is read with the password epoch in one cached lookup that is invalidated right after a change or reset. Startup bootstrap of the seeded admin password (`BOOTSTRAP_ADMIN_PASSWORD` / `BOOTSTRAP_ADMIN_GENERATE`, PR #162) is described in `api-conventions.md` section 4.
- [ ] AC-9: Passwords are hashed with bcrypt at cost 12; plaintext passwords are never logged, stored, or returned in any API response.
- [ ] AC-10: Every authentication event (login success, login failure, refresh, logout, password change) writes a row to `audit_log` with the actor's `user_id` (or the email if no user is resolved), the IP address from `X-Forwarded-For` / `RemoteAddr`, and relevant metadata.
- [ ] AC-11: `POST /api/v1/auth/refresh` with a previously-revoked refresh token immediately revokes ALL of that user's active refresh tokens and returns `401 Unauthorized` with `INVALID_REFRESH_TOKEN`.
- [ ] AC-12: `POST /api/v1/auth/change-password` with `new_password` shorter than 8 characters returns `422 Unprocessable Entity` with `VALIDATION_ERROR`.

## Technical Specification

### Database Schema

```sql
-- Migration: 007_auth.up.sql

CREATE TABLE users (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email                 TEXT        NOT NULL UNIQUE,
    password_hash         TEXT        NOT NULL,
    full_name             TEXT        NOT NULL,
    department_id         UUID        REFERENCES departments(id) ON DELETE SET NULL,
    role_id               UUID        NOT NULL REFERENCES roles(id),
    status                TEXT        NOT NULL DEFAULT 'active'
                              CHECK (status IN ('active', 'inactive')),
    force_password_change BOOLEAN     NOT NULL DEFAULT false,
    failed_attempts       INT         NOT NULL DEFAULT 0,
    locked_until          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_email         ON users(email);
CREATE INDEX idx_users_department_id ON users(department_id);
CREATE INDEX idx_users_role_id       ON users(role_id);

CREATE TABLE refresh_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT        NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id    ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
```

```sql
-- Migration: 007_auth.down.sql
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
```

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/login` | None | Authenticate with email + password |
| POST | `/api/v1/auth/refresh` | Cookie | Rotate access + refresh tokens |
| POST | `/api/v1/auth/logout` | Cookie | Revoke refresh token, clear cookie |
| POST | `/api/v1/auth/change-password` | Bearer | Change own password |

#### Request / Response shapes

```json
// POST /api/v1/auth/login — request body
{
  "email": "alice@example.com",
  "password": "S3cur3P@ss!"
}

// POST /api/v1/auth/login — 200 OK
// Set-Cookie: refresh_token=<opaque>; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=604800
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": {
      "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "full_name": "Alice Nurova",
      "email": "alice@example.com",
      "role": "department_admin",
      "force_password_change": false
    }
  },
  "error": null
}

// POST /api/v1/auth/login — 401 Unauthorized (bad credentials)
{
  "data": null,
  "error": { "code": "INVALID_CREDENTIALS", "message": "invalid email or password" }
}

// POST /api/v1/auth/login — 423 Locked
{
  "data": null,
  "error": { "code": "ACCOUNT_LOCKED", "message": "account locked, try again after 2026-05-14T10:30:00Z" }
}
```

```json
// POST /api/v1/auth/refresh — no request body (cookie sent automatically)
// 200 OK — new cookie set, new access token returned
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 900
  },
  "error": null
}

// POST /api/v1/auth/refresh — 401 (missing, expired, or revoked cookie)
{
  "data": null,
  "error": { "code": "INVALID_REFRESH_TOKEN", "message": "refresh token is invalid or expired" }
}
```

```json
// POST /api/v1/auth/logout — no request body
// 200 OK — cookie cleared (Max-Age=0)
{
  "data": { "message": "logged out" },
  "error": null
}
```

```json
// POST /api/v1/auth/change-password — request body
{
  "current_password": "OldP@ss!",
  "new_password": "NewS3cur3P@ss!"
}

// 200 OK
{
  "data": { "message": "password changed" },
  "error": null
}

// 400 Bad Request — wrong current password
{
  "data": null,
  "error": { "code": "INVALID_CREDENTIALS", "message": "current password is incorrect" }
}

// 422 Unprocessable Entity — new password too weak
{
  "data": null,
  "error": { "code": "VALIDATION_ERROR", "message": "new password must be at least 8 characters" }
}
```

### JWT Claims Shape

```json
{
  "sub": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "email": "alice@example.com",
  "role": "department_admin",
  "department_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
  "iat": 1715683200,
  "exp": 1715684100
}
```

## Out of Scope

- OAuth / SSO providers (deferred to a future phase)
- Multi-factor authentication (MFA / 2FA)
- Admin unlock UI (deferred to FR-BB18)
- Email-based password reset / forgot-password flow
- Session management beyond JWT (e.g. server-side session store)

## Test Strategy

- **Unit tests** — `internal/auth/` service layer:
  - Login happy path and credential failure
  - Account lockout counter increment and lock threshold
  - Refresh token rotation (new record inserted, old record revoked)
  - Token-reuse revocation (revoked token presented → all tokens for user revoked)
  - Password change (correct current password, wrong current password, too-short new password)
- **Integration tests** — all four endpoints exercised against a test database:
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/refresh`
  - `POST /api/v1/auth/logout`
  - `POST /api/v1/auth/change-password`
- **Table-driven edge-case tests**: locked account, expired refresh token, revoked refresh token, reused refresh token.

## Notes
- The refresh token stored in the cookie is a cryptographically random 32-byte value (base64url-encoded). Only its SHA-256 hash is stored in `refresh_tokens.token_hash` — the raw token is never persisted.
- On refresh token rotation, the old token is revoked (not deleted) to enable detection of token-reuse attacks: if a revoked token is presented, all of that user's refresh tokens are revoked immediately (see AC-11).
- `failed_attempts` is reset to 0 on successful login.
- Admin unlock (FR-BB18) sets `locked_until = NULL` and `failed_attempts = 0`.
- New password minimum requirements: ≥ 8 characters (see AC-12). Stricter policy (uppercase, digit, special char) is configurable via tenant config in a future phase.
- Migration `007_auth.up.sql` runs after FR-BB16 (`roles`) and FR-BB17 (`departments`), so both FK targets are guaranteed to exist.

- Implementation note (PR #144, ISS-105): `change-password` does not stamp `users.password_changed_at`; only admin reset (FR-BB61) and token reset (FR-BB115) do, so only those revoke previously issued access tokens. Intended scope still to be decided.
