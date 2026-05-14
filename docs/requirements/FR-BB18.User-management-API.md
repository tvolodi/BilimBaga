# FR-BB18 — User Management API

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB18 |
| Phase | 1 — Foundation |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB16 |

## Description
Exposes the full lifecycle of user accounts to admin roles: paginated listing with filters, individual create/read/update, deactivation (soft delete), password reset, and bulk CSV import with a two-phase preview-then-commit flow. The endpoint `GET /api/v1/users/me` is available to every authenticated user so the frontend can populate the current user's profile without a separate admin request.

## Acceptance Criteria
- [ ] AC-1: `GET /api/v1/users` returns a paginated list with `page`, `page_size` (default 20, max 100), and `total` metadata; accessible to `super_admin` and `department_admin`.
- [ ] AC-2: The list endpoint supports query filters: `department_id`, `role_id`, and `status` (`active` | `inactive`); multiple filters are ANDed.
- [ ] AC-3: `department_admin` can only list users whose `department_id` matches their own; super admin sees all users.
- [ ] AC-4: `POST /api/v1/users` creates a user with `force_password_change = true`; the response includes the generated temporary password in plain text exactly once (never stored or returned again).
- [ ] AC-5: `GET /api/v1/users/:id` is accessible to admins (any user) or to the user themselves (`id` matches JWT `sub`); any other user receives `403`.
- [ ] AC-6: `PUT /api/v1/users/:id` allows updating `full_name`, `department_id`, and `role_id`; only `super_admin` may change `role_id` to `super_admin`.
- [ ] AC-7: `POST /api/v1/users/:id/deactivate` sets `status = 'inactive'`; the user can no longer log in; all existing records (exam results, certificates) are preserved.
- [ ] AC-8: `POST /api/v1/users/:id/reset-password` generates a secure random temporary password, hashes it with bcrypt cost 12, stores the hash, sets `force_password_change = true`, and returns the plaintext temporary password in the response exactly once.
- [ ] AC-9: `POST /api/v1/users/import` with a valid CSV returns a preview object listing rows that would be created and rows with validation errors; no database changes occur during preview.
- [ ] AC-10: `POST /api/v1/users/import?commit=true` with a valid CSV commits all valid rows; rows with validation errors are skipped and listed in the response.
- [ ] AC-11: All mutating operations are written to `audit_log`.
- [ ] AC-12: `GET /api/v1/users/me` returns the calling user's own profile; accessible to all authenticated roles.

## Technical Specification

### Database Schema

No new tables (uses `users` from FR-BB14). Ensure an index on `department_id` and `role_id` exists (added in FR-BB14 migration).

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/users` | admin roles | Paginated user list |
| POST | `/api/v1/users` | admin roles | Create user |
| GET | `/api/v1/users/me` | any auth | Own profile |
| GET | `/api/v1/users/:id` | admin or self | User detail |
| PUT | `/api/v1/users/:id` | admin roles | Update user |
| POST | `/api/v1/users/:id/deactivate` | admin roles | Deactivate user |
| POST | `/api/v1/users/:id/reset-password` | admin roles | Reset password |
| POST | `/api/v1/users/import` | admin roles | CSV bulk import |

#### Request / Response shapes

```json
// GET /api/v1/users?page=1&page_size=20&department_id=aaa&status=active — 200 OK
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
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 1
    }
  },
  "error": null
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
    "status": "active",
    "force_password_change": true,
    "temporary_password": "Xk9!mQ2pLr",
    "created_at": "2026-05-14T10:00:00Z"
  },
  "error": null
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
    "role": "department_admin",
    "status": "active",
    "force_password_change": false
  },
  "error": null
}
```

```json
// PUT /api/v1/users/:id — request body
{
  "full_name": "Bob Seitkali",
  "department_id": "aaa00000-0000-0000-0000-000000000002",
  "role_id": "00000000-0000-0000-0000-000000000003"
}

// 200 OK — returns updated user (same shape as GET /users/:id, without temporary_password)
```

```json
// POST /api/v1/users/:id/deactivate — no request body
// 200 OK
{
  "data": { "message": "user deactivated" },
  "error": null
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
```

### Frontend Components
- `UsersListPage` — table with columns (name, email, department, role, status), filters sidebar, pagination, bulk import button.
- `UserCreateDrawer` / `UserEditDrawer` — slide-over form for create/edit.
- `ImportModal` — two-step flow: upload CSV → preview table with error highlights → confirm commit.
- `PasswordResetModal` — shows temporary password in a copyable field, closes on dismiss.

## Notes
- Temporary passwords are generated using `crypto/rand` — never `math/rand`. Length: 10 characters, mixed case + digits + one special character.
- The `temporary_password` field is only present in the `POST /users` and `POST /users/:id/reset-password` responses; never returned by any GET endpoint.
- CSV import accepts a maximum of 500 rows per request to prevent timeout issues.
- `department_name` and `role_name` in CSV import are resolved to IDs server-side; unknown values produce a row-level validation error.
- Deactivated users' refresh tokens are revoked immediately as part of the deactivate operation.
