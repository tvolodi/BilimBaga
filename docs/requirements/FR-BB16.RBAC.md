# FR-BB16 — RBAC

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB16 |
| Phase | 1 — Foundation |
| Priority | 1 |
| Status | Ready |
| Depends On | FR-BB15 |

## Description
Defines the role-based access control model for BilimBaga. Four roles (`super_admin`, `department_admin`, `examiner`, `employee`) are seeded at database level, each mapped to a fixed set of resource/action permissions. A `RequirePermission(resource, action)` Chi middleware factory is used as a route-level guard, keeping permission logic out of handler code. The full permission matrix is seeded via migration so the security model is version-controlled.

Migration 003 already created the `roles` table and seeded `super_admin`, `hr_admin`, `department_admin`, `employee`. Migration 005 (this feature) renames `hr_admin` → `examiner`, creates `permissions` and `role_permissions`, and seeds the full permission matrix.

## Scope

| Layer | Items |
|-------|-------|
| Database | New migration `005_rbac.up.sql` / `005_rbac.down.sql`: rename `hr_admin` → `examiner`; create `permissions`, `role_permissions`; seed permission matrix |
| API — new files | `internal/rbac/cache.go`, `internal/rbac/middleware.go` |
| API — modified files | `internal/router/router.go` (wire `RequirePermission` guards), `cmd/api/main.go` (initialise `rbac.Cache` at startup) |
| Frontend | None |
| i18n keys | None |

## Acceptance Criteria
- [ ] AC-1: Migration 005 creates `permissions` and `role_permissions` tables with the correct columns, constraints, and indexes. It does NOT create the `roles` table (already owned by migration 003).
- [ ] AC-2: Migration 005 renames the `hr_admin` role to `examiner`. The down migration reverses this rename and does NOT drop the `roles` table.
- [ ] AC-3: Seed data inserts all permissions and the complete role-permission matrix defined in this requirement; the seed is idempotent (re-running `make migrate` on a populated DB does not duplicate rows).
- [ ] AC-4: `RequirePermission("questions", "write")` returns `403 Forbidden` for a `department_admin` JWT, because that role does not hold `questions:write`.
- [ ] AC-5: `RequirePermission("questions", "write")` succeeds (calls `next.ServeHTTP`) for an `examiner` or `super_admin` JWT.
- [ ] AC-6: `super_admin` holds every permission defined in the `permissions` table.
- [ ] AC-7: `employee` holds only `portal:read` and `portal:submit`.
- [ ] AC-8: `RequirePermission` is composed after `auth.Authenticate` in the middleware chain; calling it on an unauthenticated request short-circuits with `401` before the permission check.
- [ ] AC-9: Adding a new permission row to the `permissions` table and mapping it to a role in `role_permissions` takes effect after an API restart without any code changes.
- [ ] AC-10: The `RequirePermission` middleware loads the permission set for the requesting user's role once per request from the in-memory cache built at startup (no per-request DB query for permission checks).

## Technical Specification

### Database Schema

The `roles` table already exists with the following schema (created by migration 003):

```sql
-- Existing table — DO NOT recreate
CREATE TABLE roles (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Migration 005 creates only the new tables and seeds:

```sql
-- Migration: 005_rbac.up.sql

-- ── Rename hr_admin to examiner (migration 003 seeded hr_admin) ───────────────
UPDATE roles SET name = 'examiner' WHERE name = 'hr_admin';

-- ── New tables ────────────────────────────────────────────────────────────────
CREATE TABLE permissions (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource TEXT NOT NULL,
    action   TEXT NOT NULL,
    UNIQUE (resource, action)
);

CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id)       ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- ── Seed permissions ─────────────────────────────────────────────────────────
INSERT INTO permissions (resource, action) VALUES
    ('users',     'read'),
    ('users',     'manage'),
    ('questions', 'read'),
    ('questions', 'write'),
    ('exams',     'read'),
    ('exams',     'write'),
    ('exams',     'assign'),
    ('reports',   'read'),
    ('portal',    'read'),
    ('portal',    'submit'),
    ('tenant',    'manage'),
    ('audit',     'read')
ON CONFLICT (resource, action) DO NOTHING;

-- ── super_admin: all permissions ─────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'super_admin'
ON CONFLICT DO NOTHING;

-- ── department_admin ─────────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON (p.resource, p.action) IN (
    ('users',     'read'),
    ('users',     'manage'),
    ('questions', 'read'),
    ('exams',     'read'),
    ('exams',     'assign'),
    ('reports',   'read')
)
WHERE r.name = 'department_admin'
ON CONFLICT DO NOTHING;

-- ── examiner ─────────────────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON (p.resource, p.action) IN (
    ('questions', 'read'),
    ('questions', 'write'),
    ('exams',     'read'),
    ('exams',     'write'),
    ('exams',     'assign'),
    ('reports',   'read')
)
WHERE r.name = 'examiner'
ON CONFLICT DO NOTHING;

-- ── employee ─────────────────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON (p.resource, p.action) IN (
    ('portal', 'read'),
    ('portal', 'submit')
)
WHERE r.name = 'employee'
ON CONFLICT DO NOTHING;
```

```sql
-- Migration: 005_rbac.down.sql
-- NOTE: Do NOT drop `roles` — it is owned by migration 003.

UPDATE roles SET name = 'hr_admin' WHERE name = 'examiner';

DELETE FROM role_permissions;
DELETE FROM permissions;

DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
```

### Permission Matrix (summary)

| Permission | super_admin | department_admin | examiner | employee |
|---|:---:|:---:|:---:|:---:|
| users:read | ✓ | ✓ | — | — |
| users:manage | ✓ | ✓ (own dept) | — | — |
| questions:read | ✓ | ✓ | ✓ | — |
| questions:write | ✓ | — | ✓ | — |
| exams:read | ✓ | ✓ | ✓ | — |
| exams:write | ✓ | — | ✓ | — |
| exams:assign | ✓ | ✓ | ✓ | — |
| reports:read | ✓ | ✓ | ✓ | — |
| portal:read | ✓ | — | — | ✓ |
| portal:submit | ✓ | — | — | ✓ |
| tenant:manage | ✓ | — | — | — |
| audit:read | ✓ | — | — | — |

> Note: `department_admin`'s `users:manage` is scoped to their own department at the handler level — the permission table only stores that the right exists, not its scope.

### Configuration / Infrastructure

#### Permission cache (`internal/rbac/cache.go`)

```go
package rbac

import "sync"

// PermissionSet maps "resource:action" → true for fast O(1) lookup.
type PermissionSet map[string]bool

type Cache struct {
    mu   sync.RWMutex
    data map[string]PermissionSet // key: role name
}

func (c *Cache) Load(db *sqlx.DB) error {
    rows, err := db.Query(`
        SELECT r.name, p.resource, p.action
        FROM role_permissions rp
        JOIN roles r       ON r.id = rp.role_id
        JOIN permissions p ON p.id = rp.permission_id
    `)
    // ... scan, build map, store under c.mu.Lock()
}

func (c *Cache) Has(role, resource, action string) bool {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.data[role][resource+":"+action]
}
```

#### RequirePermission middleware (`internal/rbac/middleware.go`)

```go
package rbac

import (
    "net/http"
    "github.com/bilimbaga/bilimbaga/internal/auth"
    "github.com/bilimbaga/bilimbaga/internal/api"
)

func RequirePermission(cache *Cache, resource, action string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            role := auth.RoleFromCtx(r.Context())
            if role == "" {
                api.WriteError(w, http.StatusUnauthorized, "MISSING_TOKEN", "authentication required")
                return
            }
            if !cache.Has(role, resource, action) {
                api.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

#### Usage in router

```go
// Example: questions routes
r.Group(func(r chi.Router) {
    r.Use(auth.Authenticate(cfg.JWTSecret))

    r.With(rbac.RequirePermission(cache, "questions", "read")).
        Get("/api/v1/questions", questionsHandler.List)

    r.With(rbac.RequirePermission(cache, "questions", "write")).
        Post("/api/v1/questions", questionsHandler.Create)
})
```

## Out of Scope

- Dynamic permission management UI (create/edit/delete permissions via the admin UI) — Phase 1 only seeds a static matrix.
- Per-resource-instance ownership checks (e.g., "examiner can only edit their own questions") — enforced at the handler/service level, not in this middleware.
- Hot-reload admin endpoint (`POST /api/v1/admin/rbac/reload`) — deferred to a later phase.
- Role creation or modification via API — roles are static in Phase 1.

## Test Strategy

- **Unit tests** (`internal/rbac/cache_test.go`): table-driven tests for `Cache.Has` — verify true/false for each role × permission combination from the matrix; verify `Has` returns false for an unknown role or unknown permission.
- **Middleware tests** (`internal/rbac/middleware_test.go`): table-driven tests covering: (a) no JWT in context → 401, (b) role without permission → 403, (c) role with permission → 200/next called. Use `httptest.NewRecorder` and a stub context value.
- **Integration test** (`internal/rbac/seed_test.go`): run the 005 migration on a test DB, call `cache.Load(db)`, and assert the full permission matrix matches expectations. Run migration twice to confirm idempotency (no duplicate rows, no error).

## Notes
- The permission cache is loaded once at startup by `cache.Load(db)` and held for the process lifetime.
- `department_admin` scope enforcement (own department only) is implemented at the handler/service level using `auth.DepartmentIDFromCtx`, not at the RBAC layer.
- Do not use role name strings directly in handler code — always go through `RequirePermission` or the `rbac.Cache` to keep permission logic centralised.
