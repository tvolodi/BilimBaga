# FR-BB19 — Audit Log

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB19 |
| Phase | 1 — Foundation |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB16 |

## Scope

| Layer | Affected |
|-------|----------|
| DB migration (`007_audit_log`) | YES |
| Backend `internal/audit` package | YES |
| Router wiring (`router.New()`) | YES |
| Frontend | NO |

## Description
Provides an immutable, append-only record of all significant platform events for compliance, security forensics, and operational monitoring. A lightweight Go helper function makes it trivial for any handler to emit an audit event. Super admins can query the log via a paginated API and export the full log as CSV. No UPDATE or DELETE is ever issued against the `audit_log` table.

## Acceptance Criteria
- [ ] AC-1: Migration creates the `audit_log` table with `id UUID PK`, `tenant_id UUID NOT NULL`, `actor_id UUID NULLABLE`, `action TEXT NOT NULL`, `entity_type TEXT`, `entity_id UUID`, `ip TEXT`, `metadata JSONB`, and `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`.
- [ ] AC-2: `h.audit.Write(ctx, r, action, entityType, entityID, metadata)` is a synchronous call that swallows all internal errors; it completes within the handler's deadline and never returns an error to the caller.
- [ ] AC-3: The `actor_id` is automatically resolved from the request context (`auth.UserIDFromCtx`) inside `Write`; callers do not pass the actor explicitly.
- [ ] AC-4: The `ip` field is populated from `r.RemoteAddr` as set by Chi's `middleware.RealIP` (applied globally in `router.New()`); `Write` does not re-implement proxy header parsing.
- [ ] AC-5: `GET /api/v1/audit` returns a paginated list (default `per_page` 50, max 200); accessible to `super_admin` only.
- [ ] AC-6: The audit list endpoint supports filters: `actor_id` (UUID), `action` (exact match), `entity_type`, `from` (ISO 8601 datetime), `to` (ISO 8601 datetime).
  Query params are validated as UUIDs where applicable (malformed `actor_id` returns 422 `VALIDATION_ERROR`). Unrecognised query parameters (e.g. `user_id`) are ignored and return the unfiltered list; use `actor_id` (GitHub #158 pending decision on a `user_id` alias).
- [ ] AC-7: `GET /api/v1/audit/export` returns the filtered result set as a CSV file with `Content-Disposition: attachment; filename="audit_export.csv"` and MIME type `text/csv`; accessible to `super_admin` only.
- [ ] AC-8: The `audit_log` table has no UPDATE or DELETE migrations, triggers, or application-level code paths that would modify or remove existing rows.
- [ ] AC-9: At minimum, the following actions must be present in the audit log when triggered: `auth.login.success`, `auth.login.failure`, `auth.logout`, `auth.password_change`, `user.create`, `user.update`, `user.deactivate`, `user.password_reset`, `department.create`, `department.update`, `department.delete`, `tenant_config.update`.

## Technical Specification

### Database Schema

```sql
-- Migration: 007_audit_log.up.sql

CREATE TABLE audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL REFERENCES tenants(id),
    actor_id    UUID,                              -- NULL for unauthenticated events (e.g. failed login)
    action      TEXT        NOT NULL,              -- e.g. "auth.login.success"
    entity_type TEXT,                              -- e.g. "user", "department"
    entity_id   UUID,                              -- ID of the affected entity
    ip          TEXT,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_log_tenant_created ON audit_log(tenant_id, created_at DESC);
CREATE INDEX idx_audit_log_actor_id       ON audit_log(actor_id);
CREATE INDEX idx_audit_log_action         ON audit_log(action);
CREATE INDEX idx_audit_log_entity_type    ON audit_log(entity_type);
```

```sql
-- Migration: 007_audit_log.down.sql
DROP TABLE IF EXISTS audit_log;
```

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/audit` | super_admin | Paginated audit log |
| GET | `/api/v1/audit/export` | super_admin | CSV export |

#### Request / Response shapes

```json
// GET /api/v1/audit?page=1&per_page=50&action=auth.login.failure&from=2026-05-01T00:00:00Z — 200 OK
{
  "data": {
    "items": [
      {
        "id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
        "actor_id": null,
        "action": "auth.login.failure",
        "entity_type": "user",
        "entity_id": null,
        "ip": "203.0.113.42",
        "metadata": { "email": "bob@example.com", "reason": "invalid_password" },
        "created_at": "2026-05-14T08:22:11Z"
      }
    ],
    "meta": {
      "page": 1,
      "per_page": 50,
      "total": 1
    }
  },
  "error": null
}
```

```
// GET /api/v1/audit/export?from=2026-05-01T00:00:00Z
// Content-Type: text/csv
// Content-Disposition: attachment; filename="audit_export.csv"
//
// id,actor_id,action,entity_type,entity_id,ip,metadata,created_at
// d290f1ee-...,,"auth.login.failure","user",,"203.0.113.42","{""email"":""bob@example.com""}","2026-05-14T08:22:11Z"
```

> **Note**: `tenant_id` is intentionally excluded from the CSV — the export is already scoped to the authenticated tenant.

### Configuration / Infrastructure

#### Audit helper (`internal/audit/audit.go`)

```go
package audit

import (
    "context"
    "encoding/json"
    "log/slog"
    "net/http"

    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    "your-module/internal/auth"
)

type Writer struct {
    db     *sqlx.DB
    logger *slog.Logger
}

func NewWriter(db *sqlx.DB, logger *slog.Logger) *Writer {
    return &Writer{db: db, logger: logger}
}

// Write records an audit event. It is safe to call from any handler.
// It does NOT return an error; failures are logged via the injected logger.
func (w *Writer) Write(ctx context.Context, r *http.Request, action, entityType string, entityID *uuid.UUID, metadata any) {
    tenantIDStr := auth.TenantIDFromCtx(ctx)
    tenantUUID, err := uuid.Parse(tenantIDStr)
    if err != nil {
        w.logger.Warn("audit: missing or invalid tenant_id, dropping event", "action", action, "err", err)
        return
    }

    actorID := auth.UserIDFromCtx(ctx) // empty string if unauthenticated
    ip      := r.RemoteAddr            // RealIP middleware has already normalised this
    var meta []byte
    if marshalErr := func() error {
        var err error
        meta, err = json.Marshal(metadata)
        return err
    }(); marshalErr != nil {
        w.logger.Warn("audit marshal failed", "err", marshalErr)
        meta = []byte("{}") // insert with empty JSON object instead of NULL
    }

    var actorNull *uuid.UUID
    if actorID != "" {
        if parsed, parseErr := uuid.Parse(actorID); parseErr == nil {
            actorNull = &parsed
        } else {
            w.logger.Warn("audit: invalid actor UUID", "actor_id", actorID, "err", parseErr)
        }
    }

    _, err := w.db.ExecContext(ctx,
        `INSERT INTO audit_log (tenant_id, actor_id, action, entity_type, entity_id, ip, metadata)
         VALUES ($1, $2, $3, $4, $5, $6, $7)`,
        tenantUUID, actorNull, action, entityType, entityID, ip, meta,
    )
    if err != nil {
        w.logger.Error("audit write failed", "err", err)
    }
}
```

#### Model (`internal/audit/types.go`)

```go
package audit

import (
    "encoding/json"
    "time"

    "github.com/google/uuid"
)

type AuditEntry struct {
    ID         uuid.UUID        `db:"id"          json:"id"`
    TenantID   uuid.UUID        `db:"tenant_id"   json:"-"`
    ActorID    *uuid.UUID       `db:"actor_id"    json:"actor_id"`
    Action     string           `db:"action"      json:"action"`
    EntityType *string          `db:"entity_type" json:"entity_type"`
    EntityID   *uuid.UUID       `db:"entity_id"   json:"entity_id"`
    IP         string           `db:"ip"          json:"ip"`
    Metadata   json.RawMessage  `db:"metadata"    json:"metadata"`
    CreatedAt  time.Time        `db:"created_at"  json:"created_at"`
}
```

#### Repository (`internal/audit/repository.go`)

All reads are tenant-scoped. The `ListAudit` method applies `WHERE tenant_id = $1` as the first mandatory filter, then appends dynamic filter clauses:

```go
package audit

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
)

type AuditFilters struct {
    ActorID    *uuid.UUID
    Action     *string
    EntityType *string
    From       *time.Time
    To         *time.Time
}

// ListAudit returns paginated audit entries scoped to tenantID.
// tenantID is ALWAYS the first WHERE clause — cross-tenant reads are impossible.
func ListAudit(
    ctx context.Context,
    db *sqlx.DB,
    tenantID uuid.UUID,
    filters AuditFilters,
    page, perPage int,
) ([]AuditEntry, int, error) {
    args := []any{tenantID} // $1 = tenant_id (mandatory)
    where := []string{"tenant_id = $1"}
    idx := 2

    if filters.ActorID != nil {
        where = append(where, fmt.Sprintf("actor_id = $%d", idx))
        args = append(args, filters.ActorID)
        idx++
    }
    if filters.Action != nil {
        where = append(where, fmt.Sprintf("action = $%d", idx))
        args = append(args, filters.Action)
        idx++
    }
    if filters.EntityType != nil {
        where = append(where, fmt.Sprintf("entity_type = $%d", idx))
        args = append(args, filters.EntityType)
        idx++
    }
    if filters.From != nil {
        where = append(where, fmt.Sprintf("created_at >= $%d", idx))
        args = append(args, filters.From)
        idx++
    }
    if filters.To != nil {
        where = append(where, fmt.Sprintf("created_at <= $%d", idx))
        args = append(args, filters.To)
        idx++
    }

    base := "FROM audit_log WHERE " + strings.Join(where, " AND ")

    var total int
    if err := db.QueryRowContext(ctx, "SELECT COUNT(*) "+base, args...).Scan(&total); err != nil {
        return nil, 0, fmt.Errorf("audit list count: %w", err)
    }

    offset := (page - 1) * perPage
    args = append(args, perPage, offset)
    rows, err := db.QueryxContext(ctx,
        "SELECT * "+base+fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", idx, idx+1),
        args...)
    if err != nil {
        return nil, 0, fmt.Errorf("audit list query: %w", err)
    }
    defer rows.Close()

    var entries []AuditEntry
    for rows.Next() {
        var e AuditEntry
        if err := rows.StructScan(&e); err != nil {
            return nil, 0, fmt.Errorf("audit list scan: %w", err)
        }
        entries = append(entries, e)
    }
    return entries, total, nil
}
```

```go
const maxExportRows = 10_000
```

The `Export` function uses the same `ListAudit` helper with `perPage = maxExportRows` and always passes the tenant ID extracted from the request context.

#### Handler wiring (`internal/audit/handler.go`)

```go
package audit

import (
    "context"

    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
)

// Service is the business-logic interface for the audit domain.
type Service interface {
    List(ctx context.Context, tenantID uuid.UUID, filters AuditFilters, page, perPage int) ([]AuditEntry, int, error)
    Export(ctx context.Context, tenantID uuid.UUID, filters AuditFilters) ([]AuditEntry, error)
}

// service implements Service, calling the repository layer.
type service struct {
    db *sqlx.DB
}

// NewService creates a new Service backed by db.
func NewService(db *sqlx.DB) Service {
    return &service{db: db}
}

func (s *service) List(ctx context.Context, tenantID uuid.UUID, filters AuditFilters, page, perPage int) ([]AuditEntry, int, error) {
    return ListAudit(ctx, s.db, tenantID, filters, page, perPage)
}

func (s *service) Export(ctx context.Context, tenantID uuid.UUID, filters AuditFilters) ([]AuditEntry, error) {
    entries, _, err := ListAudit(ctx, s.db, tenantID, filters, 1, maxExportRows)
    return entries, err
}

// Handler exposes the audit query endpoints.
type Handler struct {
    writer *Writer
    svc    Service
}

func NewHandler(svc Service, writer *Writer) *Handler {
    return &Handler{writer: writer, svc: svc}
}
```

The `audit.Handler` is added as a parameter to `router.New()`:

```go
func New(
    tenantHandler  *tenant.Handler,
    authHandler    *auth.Handler,
    deptHandler    *departments.Handler,
    usersHandler   *users.Handler,
    auditHandler   *audit.Handler,   // <-- added
    jwtSecret      string,
    rbacCache      *rbac.Cache,
) *chi.Mux {
    // ...
    r.With(rbac.RequirePermission(rbacCache, "audit", "read")).Get("/audit", auditHandler.List)
    r.With(rbac.RequirePermission(rbacCache, "audit", "read")).Get("/audit/export", auditHandler.Export)
    // ...
}
```

#### Usage example in a handler

```go
// In users handler — after successfully creating a user:
h.audit.Write(ctx, r, "user.create", "user", &newUser.ID, map[string]any{
    "email": newUser.Email,
    "role":  newUser.RoleName,
})
```

## Out of Scope

- **Audit log editing or deletion**: No UPDATE/DELETE on `audit_log` — not exposed via API or admin UI.
- **GDPR purge / right-to-erasure**: Selective removal of PII from audit rows is not part of this requirement. A separate privacy compliance feature will handle this.
- **Real-time streaming / webhooks**: Audit events are written synchronously to PostgreSQL; no event bus, WebSocket feed, or outbound webhook is included.
- **Cross-tenant audit aggregation**: Super admins from one tenant cannot query another tenant's audit log. A platform-level super-admin view across all tenants is not in scope here.

## Test Strategy

### Unit Tests (`internal/audit/`)

- **`Write()` DB error path**: inject a mock `*sqlx.DB` that returns an error on `ExecContext`; assert that `logger.Error` is called with the error and that the function returns without panicking.
- **`Write()` marshal failure**: pass an unmarshalable value (e.g., a channel) as `metadata`; assert `logger.Warn` is called with `"audit marshal failed"` and the insert proceeds with `{}` as metadata.
- **`Write()` invalid actor UUID**: pass a non-UUID string from context; assert `logger.Warn` is called and `actor_id` in the insert is NULL.

### Integration Tests (`GET /api/v1/audit`)

- **Pagination**: seed 60 rows for tenant A; request `page=2&per_page=50`; assert `meta.total=60`, `len(items)=10`.
- **Tenant isolation**: create super_admin users for tenant A and tenant B, seed rows for both; authenticate as tenant A's super_admin; assert the response contains only tenant A rows (no tenant B rows leak).
- **Filter by action**: seed rows with mixed actions; filter `action=auth.login.failure`; assert only matching rows returned.
- **Filter by date range**: seed rows across two days; filter `from` / `to`; assert boundary rows included/excluded correctly.

### Integration Tests (`GET /api/v1/audit/export`)

- **Content-Type**: assert response header `Content-Type: text/csv`.
- **Content-Disposition**: assert header value is `attachment; filename="audit_export.csv"`.
- **CSV body**: parse returned CSV; assert column headers match `id,actor_id,action,entity_type,entity_id,ip,metadata,created_at`; assert row count equals number of matching rows.
- **Tenant isolation**: same tenant-isolation assertion as the list endpoint.

## Notes
- Storing email in `metadata` for failed login events (where `actor_id` is NULL) is intentional for forensics purposes. The email field is PII — ensure the audit log table is covered by the data retention and access control policy.
- The `actor_id` is nullable specifically to support unauthenticated events (pre-login failures). Do not default it to a sentinel UUID.
- The CSV export is generated in-memory for Phase 1. If log volume grows large (>100k rows per export), a streaming approach (cursor-based iteration writing directly to the response writer) should be adopted.
- Never expose raw SQL conditions to the filter query parameters — use allowlists of column names and parameterised queries only.
- `metadata` is JSONB, not TEXT, to allow indexed queries in future analytics features.
