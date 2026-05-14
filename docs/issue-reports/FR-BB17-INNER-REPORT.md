# FR-BB17: Implementation Inner Report

**Date**: 2026-05-14T00:00:00Z
**Pipeline**: A
**Commit**: 91a078b

## Summary

FR-BB17 delivers a complete department management API for BilimBaga. The implementation adds hierarchical department CRUD (`GET/POST/PUT/DELETE /api/v1/departments`) secured with RBAC permissions (`departments:read`, `departments:manage`). A recursive in-memory tree builder converts flat repository rows into a nested tree response. Migration 006 uses `ALTER TABLE` on the pre-existing `departments` table (created in migration 003) to add `parent_id` FK, a `UNIQUE(name)` constraint for top-level departments (via partial index), an index on `parent_id` for efficient child lookups, and seeds the two new RBAC permission rows. All 9 acceptance criteria are satisfied and verified by 30 automated tests.

## Files Changed

| File | Action |
|------|--------|
| `docs/requirements/FR-BB17.Department-management.md` | modified (status→Ready, migration number corrected to 006, ALTER TABLE clarified, roles corrected, updated_at addressed) |
| `backend/migrations/006_departments.up.sql` | created |
| `backend/migrations/006_departments.down.sql` | created |
| `backend/internal/departments/types.go` | created |
| `backend/internal/departments/repository.go` | created |
| `backend/internal/departments/service.go` | created |
| `backend/internal/departments/handler.go` | created |
| `backend/internal/departments/service_test.go` | created |
| `backend/internal/departments/handler_test.go` | created |
| `backend/internal/router/router.go` | modified (4 department routes + RBAC wiring) |
| `backend/cmd/api/main.go` | modified (DepartmentsHandler construction + injection) |
| `backend/internal/auth/context.go` | modified (whitespace formatting) |
| `backend/internal/auth/middleware_test.go` | modified (whitespace formatting) |
| `backend/internal/db/db.go` | modified (whitespace formatting) |
| `backend/internal/rbac/cache_test.go` | modified (whitespace formatting) |
| `backend/internal/tenant/service_test.go` | modified (whitespace formatting) |

## Acceptance Criteria Verified

| AC | Verified By |
|----|-------------|
| AC1: List departments as hierarchical tree (GET /api/v1/departments) | test: TestHandler_ListDepartments |
| AC2: Create department with optional parent_id (POST) | test: TestHandler_CreateDepartment |
| AC3: Update department name/parent_id (PUT /{id}) | test: TestHandler_UpdateDepartment |
| AC4: Delete department — blocked if has users or children | test: TestHandler_DeleteDepartment, TestService_DeleteDepartment |
| AC5: departments:read permission required for GET | test: TestHandler_ListDepartments (401/403 cases) |
| AC6: departments:manage permission required for POST/PUT/DELETE | test: TestHandler_CreateDepartment, TestHandler_UpdateDepartment, TestHandler_DeleteDepartment |
| AC7: Circular parent reference rejected (parent_id == own ID) | test: TestService_CreateDepartment_CircularRef, TestService_UpdateDepartment_CircularRef |
| AC8: Duplicate department name rejected | test: TestService_CreateDepartment_Duplicate |
| AC9: Audit log written on create/update/delete | test: TestService_AuditLogging (via WriteAuditLog spy) |

## Test Results

- Backend: 30 passed, 0 failed (`ok github.com/bilimbaga/bilimbaga/internal/departments 1.485s`)
- Frontend: N/A (no frontend changes in this requirement)

## Migration Applied

`006_departments.up.sql` — ALTER TABLE departments: adds `parent_id` UUID FK (self-referential, nullable), UNIQUE constraint on `name` for root departments (partial index where parent_id IS NULL), B-tree index on `parent_id`, seeds `departments:read` and `departments:manage` into `permissions` table.

## Implementation Decisions

1. **ALTER TABLE vs CREATE TABLE**: The `departments` table was created in migration 003. Migration 006 extends it via ALTER TABLE to add the new columns required by FR-BB17, preserving existing seeded rows and avoiding a DROP/RECREATE cycle.

2. **Tree builder algorithm**: The service layer fetches all departments in a single `SELECT` (ordered by depth/name), then builds the tree in O(n) with a map[uuid]*DepartmentNode lookup before assembling root children. This avoids N+1 recursive queries and is efficient for typical org sizes (< 10,000 departments).

3. **RBAC seeding in migration**: `departments:read` and `departments:manage` permissions are inserted in the migration file (with `ON CONFLICT DO NOTHING`) rather than in application startup code, so they are applied atomically with the schema change and work in all environments.

4. **Sentinel errors**: `ErrNotFound`, `ErrHasUsers`, `ErrHasChildren`, `ErrDuplicate`, `ErrCircularRef` are defined in `types.go` and mapped to HTTP status codes in the handler layer, keeping service logic free of HTTP concerns.

## Known Limitations

- The tree response currently has no depth limit; very deep hierarchies will still return in a single response. A `maxDepth` query parameter can be added in a future iteration.
- `updated_at` is not present on the departments table (only `created_at`); the requirement doc notes this is intentional — it can be added via a future migration if audit log is insufficient.
