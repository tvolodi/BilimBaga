# FR-BB18 — User Management API

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB18 |
| Phase | 1 — Foundation |
| Priority | 2 |
| Status | Implemented |
| Depends On | FR-BB14, FR-BB16, FR-BB17, FR-BB19 |

## Description
Exposes the full lifecycle of user accounts to admin roles: paginated listing with filters, individual create/read/update, deactivation (soft delete), password reset, and bulk CSV import with a two-phase preview-then-commit flow. The endpoint `GET /api/v1/users/me` is available to every authenticated user so the frontend can populate the current user's profile without a separate admin request.

## Acceptance Criteria
- [x] AC-1: `GET /api/v1/users` returns a paginated list with `page`, `per_page` (default 20, max 100), and `total` metadata; accessible to `super_admin` and `department_admin`.
- [x] AC-2: The list endpoint supports query filters: `department_id`, `role_id`, and `status` (`active` | `inactive`); multiple filters are ANDed.
- [x] AC-3: `department_admin` can only list users whose `department_id` matches their own; super admin sees all users.
- [x] AC-4: `POST /api/v1/users` creates a user with `force_password_change = true`; the response includes the generated temporary password in plain text exactly once (never stored or returned again). A `department_admin` may only create users whose `department_id` matches their own; supplying any other `department_id` returns `403`.
- [x] AC-5: `GET /api/v1/users/:id` is accessible to `super_admin`, `department_admin`, or the user themselves (`id` matches JWT `sub`); any other user receives `403`. A `department_admin` may access any user whose `department_id` matches their own OR whose `id` matches the calling user's own JWT `sub`.
- [x] AC-6: `PUT /api/v1/users/:id` allows updating `full_name`, `department_id`, and `role_id`; `department_admin` may change `role_id` only to `department_admin` or `employee`; only `super_admin` may assign or remove the `super_admin` role. A `department_admin` may only perform this operation on users whose `department_id` matches their own; any other user ID returns `403`.
- [x] AC-7: `POST /api/v1/users/:id/deactivate` sets `status = 'inactive'`; the user can no longer log in; all existing records (exam results, certificates) are preserved. A `department_admin` may only perform this operation on users whose `department_id` matches their own; any other user ID returns `403`.
- [x] AC-8: `POST /api/v1/users/:id/reset-password` generates a secure random temporary password, hashes it with bcrypt cost 12, stores the hash, sets `force_password_change = true`, and returns the plaintext temporary password in the response exactly once. A `department_admin` may only perform this operation on users whose `department_id` matches their own; any other user ID returns `403`.
- [x] AC-9: `POST /api/v1/users/import` with a valid CSV returns a preview object listing rows that would be created and rows with validation errors; no database changes occur during preview. A `department_admin`'s import preview is restricted to rows whose resolved `department_name` maps to their own department; rows resolving to a different department are treated as validation errors.
- [x] AC-10: `POST /api/v1/users/import?commit=true` with a valid CSV commits all valid rows; rows with validation errors are skipped and listed in the response. A `department_admin`'s committed import only creates users within their own department; rows whose `department_name` resolves to a different department are skipped and listed as errors.
- [x] AC-11: All mutating operations are written to `audit_log`.
- [x] AC-12: `GET /api/v1/users/me` returns the calling user's own profile; accessible to all authenticated roles.

## Technical Specification

### Database Schema

No new tables (uses `users` from FR-BB14). Ensure an index on `department_id` and `role_id` exists (added in FR-BB14 migration).

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/users` | `super_admin`, `department_admin` | Paginated user list |
| POST | `/api/v1/users` | `super_admin`, `department_admin` | Create user (dept-scoped for department_admin) |
| GET | `/api/v1/users/me` | any auth | Own profile |
| GET | `/api/v1/users/:id` | `super_admin`, `department_admin`, or self | User detail |
| PUT | `/api/v1/users/:id` | `super_admin`, `department_admin` | Update user |
| POST | `/api/v1/users/:id/deactivate` | `super_admin`, `department_admin` | Deactivate user |
| POST | `/api/v1/users/:id/reset-password` | `super_admin`, `department_admin` | Reset password |
| POST | `/api/v1/users/import` | `super_admin`, `department_admin` | CSV bulk import (dept-scoped for department_admin) |

The `examiner` role is excluded from all user management endpoints.

#### Request / Response shapes

```json
// GET /api/v1/users?page=1&per_page=20&department_id=aaa&status=active — 200 OK
{
  "data": {
    "items": [
      {
        "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
        "email": "alice@example.com",
        "full_name": "Alice Nurova",
        "department_id": "aaa00000-0000-0000-0000-000000000001",
        "department_name": "Engineering",
        "role_id": "00000000-0000-0000-0000-000000000002",
        "role_name": "department_admin",
        "status": "active",
        "force_password_change": false,
        "created_at": "2026-01-01T00:00:00Z"
      }
    ],
    "meta": {
      "page": 1,
      "per_page": 20,
      "total": 1
    }
  },
  "error": null
}
```

```json
// GET /api/v1/users — 403 Forbidden (caller is examiner or employee)
{
  "data": null,
  "error": { "code": "FORBIDDEN", "message": "insufficient permissions" }
}
```

```json
// POST /api/v1/users — request body
{
  "email": "bob@example.com",
  "full_name": "Bob Serov",
  "department_id": "aaa00000-0000-0000-0000-000000000001",
  "role_id": "00000000-0000-0000-0000-000000000003"
}

// 201 Created
{
  "data": {
    "id": "c9bf9e57-1685-4c89-bafb-ff5af830be8a",
    "email": "bob@example.com",
    "full_name": "Bob Serov",
    "department_id": "aaa00000-0000-0000-0000-000000000001",
    "role_id": "00000000-0000-0000-0000-000000000003",
    "role_name": "employee",
    "department_name": "Engineering",
    "status": "active",
    "force_password_change": true,
    "temporary_password": "Xk9!mQ2pLr",
    "created_at": "2026-05-14T10:00:00Z"
  },
  "error": null
}
```

```json
// POST /api/v1/users — 409 Conflict (duplicate email)
{
  "data": null,
  "error": { "code": "DUPLICATE_EMAIL", "message": "a user with this email already exists" }
}
```

```json
// POST /api/v1/users — 422 Unprocessable Entity (validation error)
{
  "data": null,
  "error": { "code": "VALIDATION_ERROR", "message": "email is required" }
}
```

```json
// GET /api/v1/users/:id — 200 OK
{
  "data": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "email": "alice@example.com",
    "full_name": "Alice Nurova",
    "department_id": "aaa00000-0000-0000-0000-000000000001",
    "department_name": "Engineering",
    "role_id": "00000000-0000-0000-0000-000000000002",
    "role_name": "department_admin",
    "status": "active",
    "force_password_change": false,
    "created_at": "2026-01-01T00:00:00Z"
  },
  "error": null
}
```

```json
// GET /api/v1/users/:id — 403 Forbidden
{
  "data": null,
  "error": { "code": "FORBIDDEN", "message": "insufficient permissions" }
}
```

```json
// GET /api/v1/users/:id — 404 Not Found
{
  "data": null,
  "error": { "code": "NOT_FOUND", "message": "user not found" }
}
```

```json
// GET /api/v1/users/me — 200 OK
{
  "data": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "email": "alice@example.com",
    "full_name": "Alice Nurova",
    "department_id": "aaa00000-0000-0000-0000-000000000001",
    "department_name": "Engineering",
    "role_id": "00000000-0000-0000-0000-000000000002",
    "role_name": "department_admin",
    "status": "active",
    "force_password_change": false
  },
  "error": null
}
```

```json
// GET /api/v1/users/me — 401 Unauthorized
{
  "data": null,
  "error": { "code": "UNAUTHORIZED", "message": "authentication required" }
}
```

```json
// PUT /api/v1/users/:id — request body
{
  "full_name": "Bob Seitkali",
  "department_id": "aaa00000-0000-0000-0000-000000000002",
  "role_id": "00000000-0000-0000-0000-000000000003"
}

// 200 OK
{
  "data": {
    "id": "c9bf9e57-1685-4c89-bafb-ff5af830be8a",
    "email": "bob@example.com",
    "full_name": "Bob Seitkali",
    "department_id": "aaa00000-0000-0000-0000-000000000002",
    "department_name": "HR",
    "role_id": "00000000-0000-0000-0000-000000000003",
    "role_name": "employee",
    "status": "active",
    "force_password_change": false,
    "created_at": "2026-05-14T10:00:00Z"
  },
  "error": null
}
```

```json
// 422 — validation error
{
  "data": null,
  "error": { "code": "VALIDATION_ERROR", "message": "full_name must not be blank" }
}
```

```json
// PUT /api/v1/users/:id — 403 Forbidden
{
  "data": null,
  "error": { "code": "FORBIDDEN", "message": "insufficient permissions" }
}
```

```json
// PUT /api/v1/users/:id — 404 Not Found
{
  "data": null,
  "error": { "code": "NOT_FOUND", "message": "user not found" }
}
```

```json
// POST /api/v1/users/:id/deactivate — no request body
// 200 OK
{
  "data": { "message": "user deactivated" },
  "error": null
}

// 403 Forbidden
{
  "data": null,
  "error": { "code": "FORBIDDEN", "message": "access denied" }
}

// 404 — user not found
{
  "data": null,
  "error": { "code": "NOT_FOUND", "message": "user not found" }
}
```

```json
// POST /api/v1/users/:id/reset-password — no request body
// 200 OK
{
  "data": {
    "temporary_password": "Yw3!zN8kPq"
  },
  "error": null
}

// 403 Forbidden
{
  "data": null,
  "error": { "code": "FORBIDDEN", "message": "access denied" }
}
```

```json
// POST /api/v1/users/:id/reset-password — 404 Not Found
{
  "data": null,
  "error": { "code": "NOT_FOUND", "message": "user not found" }
}
```

```json
// POST /api/v1/users/import — multipart/form-data, field: "file" (CSV)
// CSV format: email,full_name,department_name,role_name
// Preview response (no ?commit=true) — 200 OK
{
  "data": {
    "preview": [
      { "row": 2, "email": "carol@example.com", "full_name": "Carol Kim", "status": "valid" },
      { "row": 3, "email": "invalid-email",      "full_name": "Dave X",   "status": "error", "error": "invalid email format" }
    ],
    "valid_count": 1,
    "error_count": 1,
    "committed": false
  },
  "error": null
}

// POST /api/v1/users/import?commit=true — 200 OK
{
  "data": {
    "created": 1,
    "skipped": [
      { "row": 3, "email": "invalid-email", "error": "invalid email format" }
    ],
    "committed": true
  },
  "error": null
}

// 403 Forbidden
{
  "data": null,
  "error": { "code": "FORBIDDEN", "message": "access denied" }
}

// 422 — CSV exceeds 500-row limit
{
  "data": null,
  "error": { "code": "CSV_TOO_LARGE", "message": "CSV file exceeds the maximum of 500 rows" }
}
```

### Frontend Components
- `UsersListPage` (i18n: `users:list`) — table with columns (name, email, department, role, status), filters sidebar, pagination, bulk import button.
- `UserCreateDrawer` (i18n: `users:create`) / `UserEditDrawer` (i18n: `users:edit`) — slide-over form for create/edit.
- `ImportModal` (i18n: `users:import`) — two-step flow: upload CSV → preview table with error highlights → confirm commit.
- `PasswordResetModal` (i18n: `users:password-reset`) — shows temporary password in a copyable field, closes on dismiss.
- `DeactivateConfirmDialog` (i18n: `users:deactivate`) — confirmation dialog for AC-7; confirms intent before calling `POST /api/v1/users/:id/deactivate`.

## Notes
- Temporary passwords are generated using `crypto/rand` — never `math/rand`. Length: 10 characters, mixed case + digits + one special character.
- The `temporary_password` field is only present in the `POST /users` and `POST /users/:id/reset-password` responses; never returned by any GET endpoint.
- CSV import accepts a maximum of 500 rows per request to prevent timeout issues.
- `department_name` and `role_name` in CSV import are resolved to IDs server-side; unknown values produce a row-level validation error.
- Deactivated users' refresh tokens are revoked immediately as part of the deactivate operation.
