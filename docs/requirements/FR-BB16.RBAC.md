# FR-BB16 — RBAC

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB16 |
| Phase | 1 — Foundation |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB15 |

## Description
Defines the role-based access control model for BilimBaga. Four roles (`super_admin`, `department_admin`, `examiner`, `employee`) are seeded at database level, each mapped to a fixed set of resource/action permissions. A `RequirePermission(resource, action)` Chi middleware factory is used as a route-level guard, keeping permission logic out of handler code. The full permission matrix is seeded via migration so the security model is version-controlled.

## Acceptance Criteria
- [ ] AC-1: Migration creates `roles`, `permissions`, and `role_permissions` tables with the correct columns, constraints, and indexes.
- [ ] AC-2: Seed data inserts all four roles and the complete permission matrix defined in this requirement; the seed is idempotent (re-running `make migrate` on a populated DB does not duplicate rows).
- [ ] AC-3: `RequirePermission("questions", "write")` returns `403 Forbidden` for a `department_admin` JWT, because that role does not hold `questions:write`.
- [ ] AC-4: `RequirePermission("questions", "write")` succeeds (calls `next.ServeHTTP`) for an `examiner` or `super_admin` JWT.
- [ ] AC-5: `super_admin` holds every permission defined in the `permissions` table.
- [ ] AC-6: `employee` holds only `portal:read` and `portal:submit`.
- [ ] AC-7: `RequirePermission` is composed after `auth.Authenticate` in the middleware chain; calling it on an unauthenticated request short-circuits with `401` before the permission check.
- [ ] AC-8: Adding a new permission row to the `permissions` table and mapping it to a role in `role_permissions` takes effect after an API restart without any code changes.
- [ ] AC-9: The `RequirePermission` middleware loads the permission set for the requesting user's role once per request from the in-memory cache built at startup (no per-request DB query for permission checks).

## Technical Specification

### Database Schema

```sql
-- Migration: 004_rbac.up.sql

CREATE TABLE roles (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE
);

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

-- ── Seed roles ───────────────────────────────────────────────────────────────
INSERT INTO roles (id, name) VALUES
    ('00000000-0000-0000-0000-000000000001', 'super_admin'),
    ('00000000-0000-0000-0000-000000000002', 'department_admin'),
    ('00000000-0000-0000-0000-000000000003', 'examiner'),
    ('00000000-0000-0000-0000-000000000004', 'employee')
ON CONFLICT (name) DO NOTHING;

-- ── Seed permissions ─────────────────────────────────────────────────────────
INSERT INTO permissions (id, resource, action) VALUES
    ('10000000-0000-0000-0000-000000000001', 'users',     'read'),
    ('10000000-0000-0000-0000-000000000002', 'users',     'manage'),
    ('10000000-0000-0000-0000-000000000003', 'questions', 'read'),
    ('10000000-0000-0000-0000-000000000004', 'questions', 'write'),
    ('10000000-0000-0000-0000-000000000005', 'exams',     'read'),
    ('10000000-0000-0000-0000-000000000006', 'exams',     'write'),
    ('10000000-0000-0000-0000-000000000007', 'exams',     'assign'),
    ('10000000-0000-0000-0000-000000000008', 'reports',   'read'),
    ('10000000-0000-0000-0000-000000000009', 'portal',    'read'),
    ('10000000-0000-0000-0000-000000000010', 'portal',    'submit'),
    ('10000000-0000-0000-0000-000000000011', 'tenant',    'manage'),
    ('10000000-0000-0000-0000-000000000012', 'audit',     'read')
ON CONFLICT (resource, action) DO NOTHING;

-- ── super_admin: all permissions ─────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000001', id FROM permissions
ON CONFLICT DO NOTHING;

-- ── department_admin ─────────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000002', id
FROM permissions
WHERE (resource, action) IN (
    ('users',     'read'),
    ('users',     'manage'),
    ('questions', 'read'),
    ('exams',     'read'),
    ('exams',     'assign'),
    ('reports',   'read')
)
ON CONFLICT DO NOTHING;

-- ── examiner ─────────────────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000003', id
FROM permissions
WHERE (resource, action) IN (
    ('questions', 'write'),
    ('questions', 'read'),
    ('exams',     'write'),
    ('exams',     'read'),
    ('exams',     'assign'),
    ('reports',   'read')
)
ON CONFLICT DO NOTHING;

-- ── employee ─────────────────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000004', id
FROM permissions
WHERE (resource, action) IN (
    ('portal', 'read'),
    ('portal', 'submit')
)
ON CONFLICT DO NOTHING;
```

```sql
-- Migration: 004_rbac.down.sql
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
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
    "your-module/internal/auth"
    "your-module/internal/api"
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

## Notes
- Fixed UUIDs for seeded roles and permissions make cross-environment consistency easy to verify and simplify down migrations.
- The permission cache is loaded once at startup by `cache.Load(db)` and held for the process lifetime. A future admin endpoint (`POST /api/v1/admin/rbac/reload`) can trigger a hot reload without restart.
- `department_admin` scope enforcement (own department only) is implemented at the handler/service level using `auth.DepartmentIDFromCtx`, not at the RBAC layer.
- Do not use role name strings directly in handler code — always go through `RequirePermission` or the `rbac.Cache` to keep permission logic centralised.
