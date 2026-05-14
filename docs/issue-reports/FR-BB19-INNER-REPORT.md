# FR-BB19 Audit Log — Inner Report

## Summary

Implemented the audit log feature end-to-end as specified in `docs/requirements/FR-BB19.Audit-log.md`.

## Migration

- **`backend/migrations/007_audit_log.up.sql`** — Creates `audit_log` table with columns:
  `id` (UUID PK), `tenant_id` (TEXT NOT NULL), `actor_id` (UUID nullable), `action` (TEXT), `entity_type` (TEXT nullable), `entity_id` (UUID nullable), `ip` (TEXT), `metadata` (JSONB), `created_at` (TIMESTAMPTZ).  
  Four indexes: `idx_audit_log_tenant_created`, `idx_audit_log_actor_id`, `idx_audit_log_action`, `idx_audit_log_entity_type`.
- **`backend/migrations/007_audit_log.down.sql`** — `DROP TABLE IF EXISTS audit_log;`
- Migration applied to DB at implementation time.

## New Package: `internal/audit`

| File | Purpose |
|------|---------|
| `types.go` | `AuditEntry`, `AuditFilters`, `maxExportRows` const |
| `writer.go` | Nil-safe `Writer` struct; `Write()` extracts tenant/actor from context, INSERTs row |
| `repository.go` | `ListAudit()` — dynamic WHERE + COUNT + paginated SELECT |
| `service.go` | `Service` interface + `service` impl; clamps pagination; delegates to `ListAudit` |
| `handler.go` | `Handler` with `List` (GET /api/v1/audit) and `Export` (GET /api/v1/audit/export as CSV) |
| `writer_test.go` | Tests nil-writer no-op and missing-tenant-ID early-return |
| `handler_test.go` | Tests List 200, List 401, Export 200 (CSV), Export 401 |

## New Package: `internal/ctxkeys`

Created to break the import cycle between `audit` and `auth`. Defines typed context key constants (`CtxUserID`, `CtxRole`, `CtxDepartmentID`, `CtxTenantID`) and getter functions. Both `auth` and `audit` now import `ctxkeys` instead of each other.

## Modified Files

### Backend packages
| File | Change |
|------|--------|
| `internal/auth/context.go` | Delegates to `ctxkeys` (preserves public API) |
| `internal/auth/tenant.go` | Uses `ctxkeys.CtxTenantID` |
| `internal/auth/middleware.go` | Uses `ctxkeys.Ctx*` constants |
| `internal/auth/repository.go` | Removed `AuditEntry` struct and `WriteAuditLog` method |
| `internal/auth/service.go` | Removed all `writeAudit()` calls |
| `internal/auth/handler.go` | Added `*audit.Writer`; emits `auth.login.success`, `auth.login.failure`, `auth.logout`, `auth.password_change` |
| `internal/auth/handler_test.go` | Updated `NewHandler` calls; removed `mockRepository.WriteAuditLog` |
| `internal/users/repository.go` | Removed `WriteAuditLog` |
| `internal/users/service.go` | Removed all `WriteAuditLog` calls |
| `internal/users/handler.go` | Added `*audit.Writer`; emits `user.create`, `user.update`, `user.deactivate`, `user.password_reset` |
| `internal/users/service_test.go` | Removed `mockRepo.auditLog` and `WriteAuditLog` method |
| `internal/departments/repository.go` | Removed `WriteAuditLog` |
| `internal/departments/service.go` | Removed all `WriteAuditLog` calls |
| `internal/departments/handler.go` | Added `*audit.Writer`; emits `department.create`, `department.update`, `department.delete` |
| `internal/departments/handler_test.go` | Updated `NewHandler` calls |
| `internal/departments/service_test.go` | Removed `mockRepo.auditLog` and `WriteAuditLog` method |
| `internal/tenant/handler.go` | Added `*audit.Writer`; emits `tenant_config.update` |
| `internal/tenant/handler_test.go` | Updated `NewHandler` calls |
| `internal/router/router.go` | Added `*audit.Handler` param; registered `GET /audit` and `GET /audit/export` with `audit:read` RBAC |
| `cmd/api/main.go` | Wires `auditWriter`, `auditSvc`, `auditHandler`; passes writer to all domain handlers |

## Test Results

```
ok  github.com/bilimbaga/bilimbaga/internal/audit
ok  github.com/bilimbaga/bilimbaga/internal/auth
ok  github.com/bilimbaga/bilimbaga/internal/config
ok  github.com/bilimbaga/bilimbaga/internal/db
ok  github.com/bilimbaga/bilimbaga/internal/departments
ok  github.com/bilimbaga/bilimbaga/internal/health
ok  github.com/bilimbaga/bilimbaga/internal/rbac
ok  github.com/bilimbaga/bilimbaga/internal/tenant
ok  github.com/bilimbaga/bilimbaga/internal/users
```

## Deviations from Spec

1. **`tenant_id` is TEXT, not UUID FK** — Phase 1 has no `tenants` table; `TenantIDFromCtx()` returns `"public"` (string). The column accepts any string value.
2. **`actor_id` stored as `TEXT` UUID string** — There is no `google/uuid` dependency; all IDs are plain Go strings throughout the codebase.
