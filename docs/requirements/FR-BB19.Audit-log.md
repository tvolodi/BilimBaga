# FR-BB19 — Audit Log

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB19 |
| Phase | 1 — Foundation |
| Priority | 2 |
| Status | Draft |
| Depends On | FR-BB16 |

## Description
Provides an immutable, append-only record of all significant platform events for compliance, security forensics, and operational monitoring. A lightweight Go helper function makes it trivial for any handler to emit an audit event. Super admins can query the log via a paginated API and export the full log as CSV. No UPDATE or DELETE is ever issued against the `audit_log` table.

## Acceptance Criteria
- [ ] AC-1: Migration creates the `audit_log` table with `id UUID PK`, `actor_id UUID NULLABLE`, `action TEXT NOT NULL`, `entity_type TEXT`, `entity_id UUID`, `ip TEXT`, `metadata JSONB`, and `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`.
- [ ] AC-2: `audit.Write(ctx, action, entityType, entityID, metadata)` is a fire-and-forget call; it must not block the request path — it runs asynchronously or in the same transaction but does not return an error to the caller.
- [ ] AC-3: The `actor_id` is automatically resolved from the request context (`auth.UserIDFromCtx`) inside `audit.Write`; callers do not pass the actor explicitly.
- [ ] AC-4: The `ip` is resolved inside `audit.Write` from `X-Forwarded-For` (first IP) or `RemoteAddr` if the header is absent.
- [ ] AC-5: `GET /api/v1/audit` returns a paginated list (default page_size 50, max 200); accessible to `super_admin` only.
- [ ] AC-6: The audit list endpoint supports filters: `actor_id` (UUID), `action` (exact match), `entity_type`, `from` (ISO 8601 datetime), `to` (ISO 8601 datetime).
- [ ] AC-7: `GET /api/v1/audit/export` returns the filtered result set as a CSV file with `Content-Disposition: attachment; filename="audit_export.csv"` and MIME type `text/csv`; accessible to `super_admin` only.
- [ ] AC-8: The `audit_log` table has no UPDATE or DELETE migrations, triggers, or application-level code paths that would modify or remove existing rows.
- [ ] AC-9: At minimum, the following actions must be present in the audit log when triggered: `auth.login.success`, `auth.login.failure`, `auth.logout`, `auth.password_change`, `user.create`, `user.update`, `user.deactivate`, `user.password_reset`, `department.create`, `department.update`, `department.delete`, `tenant_config.update`.

## Technical Specification

### Database Schema

```sql
-- Migration: 006_audit_log.up.sql

CREATE TABLE audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id    UUID,                              -- NULL for unauthenticated events (e.g. failed login)
    action      TEXT        NOT NULL,              -- e.g. "auth.login.success"
    entity_type TEXT,                              -- e.g. "user", "department"
    entity_id   UUID,                              -- ID of the affected entity
    ip          TEXT,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_log_actor_id    ON audit_log(actor_id);
CREATE INDEX idx_audit_log_action      ON audit_log(action);
CREATE INDEX idx_audit_log_entity_type ON audit_log(entity_type);
CREATE INDEX idx_audit_log_created_at  ON audit_log(created_at DESC);
```

```sql
-- Migration: 006_audit_log.down.sql
DROP TABLE IF EXISTS audit_log;
```

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/audit` | super_admin | Paginated audit log |
| GET | `/api/v1/audit/export` | super_admin | CSV export |

#### Request / Response shapes

```json
// GET /api/v1/audit?page=1&page_size=50&action=auth.login.failure&from=2026-05-01T00:00:00Z — 200 OK
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
    "pagination": {
      "page": 1,
      "page_size": 50,
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

### Configuration / Infrastructure

#### Audit helper (`internal/audit/audit.go`)

```go
package audit

import (
    "context"
    "encoding/json"
    "net/http"
    "strings"

    "github.com/jmoiron/sqlx"
    "your-module/internal/auth"
)

type Writer struct {
    db *sqlx.DB
}

func NewWriter(db *sqlx.DB) *Writer {
    return &Writer{db: db}
}

// Write records an audit event. It is safe to call from any handler.
// It does NOT return an error; failures are logged internally only.
func (w *Writer) Write(ctx context.Context, r *http.Request, action, entityType string, entityID *string, metadata any) {
    actorID := auth.UserIDFromCtx(ctx) // empty string if unauthenticated
    ip      := resolveIP(r)
    meta, _ := json.Marshal(metadata)

    var actorNull *string
    if actorID != "" {
        actorNull = &actorID
    }

    _, err := w.db.ExecContext(ctx,
        `INSERT INTO audit_log (actor_id, action, entity_type, entity_id, ip, metadata)
         VALUES ($1, $2, $3, $4, $5, $6)`,
        actorNull, action, entityType, entityID, ip, meta,
    )
    if err != nil {
        // Log error internally; never surface to caller
        _ = err
    }
}

func resolveIP(r *http.Request) string {
    if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
        return strings.SplitN(fwd, ",", 2)[0]
    }
    return r.RemoteAddr
}
```

#### Usage example in a handler

```go
// In users handler — after successfully creating a user:
audit.Write(ctx, r, "user.create", "user", &newUser.ID, map[string]any{
    "email": newUser.Email,
    "role":  newUser.RoleName,
})
```

## Notes
- Storing email in `metadata` for failed login events (where `actor_id` is NULL) is intentional for forensics purposes. The email field is PII — ensure the audit log table is covered by the data retention and access control policy.
- The `actor_id` is nullable specifically to support unauthenticated events (pre-login failures). Do not default it to a sentinel UUID.
- The CSV export is generated in-memory for Phase 1. If log volume grows large (>100k rows per export), a streaming approach (cursor-based iteration writing directly to the response writer) should be adopted.
- Never expose raw SQL conditions to the filter query parameters — use allowlists of column names and parameterised queries only.
- `metadata` is JSONB, not TEXT, to allow indexed queries in future analytics features.
