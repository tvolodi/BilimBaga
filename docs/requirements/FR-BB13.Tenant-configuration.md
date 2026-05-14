# FR-BB13 — Tenant Configuration

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB13 |
| Phase | 1 — Foundation |
| Priority | 1 |
| Status | Revised |
| Depends On | FR-BB12 |

## Description
Provides a runtime-configurable branding and localisation store for the platform. A single `tenant_config` table holds key/value pairs (JSONB values) covering application name, logo, colours, and locale settings. The Go service loads and caches this configuration at startup so that every request has access to branding data without a round-trip to the database. Super admins may update configuration at runtime via a protected API endpoint; public endpoints expose the subset of data needed by the frontend before authentication.

## Scope

| Layer | Items |
|-------|-------|
| Database | `tenant_config` table; `migrations/002_tenant_config.up.sql`, `migrations/002_tenant_config.down.sql` |
| API endpoints | `GET /api/v1/tenant/config`, `PUT /api/v1/tenant/config`, `GET /api/v1/tenant/logo` |
| Backend packages | `internal/tenant/handler.go`, `internal/tenant/service.go`, `internal/tenant/repository.go` |
| Frontend | `src/api/useTenantConfig.ts`, `src/components/TenantLogo.tsx` |
| i18n | `src/locales/kk.json`, `src/locales/ru.json`, `src/locales/en.json` — key `common.logo_placeholder` |

## Acceptance Criteria
- [ ] AC-1: Migration creates the `tenant_config` table with `key TEXT PRIMARY KEY`, `value JSONB NOT NULL`, and `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()` columns.
- [ ] AC-2: Seed data populates default rows for all six keys: `app_name`, `logo`, `primary_color`, `accent_color`, `default_locale`, `available_locales`.
- [ ] AC-3: On API startup, all rows are read from `tenant_config` and stored in an in-memory cache; subsequent reads from the cache do not hit the database.
- [ ] AC-4: `GET /api/v1/tenant/config` returns `app_name`, `primary_color`, `accent_color`, `default_locale`, and `available_locales` without requiring authentication.
- [ ] AC-5: `GET /api/v1/tenant/logo` returns the raw logo bytes with the correct `Content-Type` header (e.g. `image/png`, `image/svg+xml`) derived from the stored base64 metadata.
- [ ] AC-6: `PUT /api/v1/tenant/config` accepts a partial update object; only provided keys are modified; missing keys retain their existing values.
- [ ] AC-7: `PUT /api/v1/tenant/config` is accessible only to users with the `super_admin` role; any other role receives `403 Forbidden`.
- [ ] AC-8: After a successful `PUT /api/v1/tenant/config`, the in-memory cache is invalidated and refreshed so subsequent `GET` requests return the new values without a service restart.

> **NOTE**: Audit logging for tenant config updates will be wired in FR-BB19.

## Technical Specification

### Database Schema

```sql
-- Migration: 002_tenant_config.up.sql

CREATE TABLE tenant_config (
    key        TEXT                     PRIMARY KEY,
    value      JSONB                    NOT NULL,
    updated_at TIMESTAMPTZ              NOT NULL DEFAULT now()
);

-- Seed default configuration
INSERT INTO tenant_config (key, value) VALUES
    ('app_name',          '"BilimBaga"'),
    ('logo',              'null'),
    ('primary_color',     '"#0ea5e9"'),
    ('accent_color',      '"#f59e0b"'),
    ('default_locale',    '"kk"'),
    ('available_locales', '["kk","ru","en"]')
ON CONFLICT (key) DO NOTHING;
```

```sql
-- Migration: 002_tenant_config.down.sql
DROP TABLE IF EXISTS tenant_config;
```

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/tenant/config` | None | Returns public branding config |
| PUT | `/api/v1/tenant/config` | super_admin | Updates one or more config keys |
| GET | `/api/v1/tenant/logo` | None | Serves logo image bytes |

#### Request / Response shapes

```json
// GET /api/v1/tenant/config — 200 OK
{
  "data": {
    "app_name": "BilimBaga",
    "primary_color": "#0ea5e9",
    "accent_color": "#f59e0b",
    "default_locale": "kk",
    "available_locales": ["kk", "ru", "en"]
  },
  "error": null
}
```

```json
// PUT /api/v1/tenant/config — request body
{
  "app_name": "Acme Corp Training",
  "primary_color": "#1d4ed8",
  "accent_color": "#f97316"
}

// PUT /api/v1/tenant/config — 200 OK response
{
  "data": { "updated": ["app_name", "primary_color", "accent_color"] },
  "error": null
}
```

```json
// PUT /api/v1/tenant/config — 403 Forbidden
{
  "data": null,
  "error": { "code": "FORBIDDEN", "message": "insufficient permissions" }
}
```

```json
// GET /api/v1/tenant/logo — binary response
// Content-Type: image/png  (or image/svg+xml, image/jpeg as detected)
// Body: raw image bytes
// On no logo configured — 204 No Content
```

### Go Architecture

**Package**: `internal/tenant`

**Files**:
- `repository.go` — direct database access only
- `service.go` — business logic and cache
- `handler.go` — HTTP layer, thin

**Constructor chain** (wired in `internal/router/router.go`):
```
NewRepository(*sqlx.DB) → NewService(Repository) → NewHandler(Service)
```

#### Repository interface

```go
type Repository interface {
    GetAll(ctx context.Context) (map[string]json.RawMessage, error)
    Upsert(ctx context.Context, key string, value json.RawMessage) error
}
```

`GetAll` reads all rows from `tenant_config` and returns them as a keyed map.  
`Upsert` performs an `INSERT … ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`.

#### Service responsibilities

- Holds a `sync.RWMutex`-protected in-memory cache (`map[string]json.RawMessage`).
- `LoadCache(ctx context.Context) error` — called once at startup; populates cache from repository.
- `InvalidateAndRefresh(ctx context.Context) error` — called after a successful `PUT`; clears cache and reloads from repository under write lock.
- Validates on `PUT`: `available_locales` must be a superset of `default_locale`; returns `400 Bad Request` with `code: "INVALID_LOCALE"` if violated.
- Decodes logo base64 data URI (`data:<mime>;base64,<data>`) to extract MIME type and raw bytes; enforces 1 MB decoded size limit.

#### Handler responsibilities

- Parse and validate request body → delegate to service → serialize and return JSON response.
- No business logic in handler.
- `GET /api/v1/tenant/config` and `GET /api/v1/tenant/logo` are registered without JWT middleware.
- `PUT /api/v1/tenant/config` is wrapped with JWT + RBAC middleware requiring `super_admin` role.

### Frontend Components

- `useTenantConfig()` — `src/api/useTenantConfig.ts`; React Query hook with query key `['tenant', 'config']`; fetches `GET /api/v1/tenant/config`; `staleTime: Infinity` so it is fetched once per session and consumed before rendering `<App />`.
- CSS custom properties (`--color-primary`, `--color-accent`) injected into `:root` from the config response so Tailwind's `primary` and `accent` colour tokens resolve to tenant values.
- `<TenantLogo />` — `src/components/TenantLogo.tsx`; renders `<img src="/api/v1/tenant/logo" />`; when the server returns `204 No Content` renders a text fallback using i18n key `common.logo_placeholder`. The key must be added to `src/locales/kk.json`, `src/locales/ru.json`, and `src/locales/en.json`.

## Out of Scope

- Logo upload via multipart/form-data — Phase 1 stores logo as base64 JSONB only.
- Per-schema multi-tenant configuration — Phase 6 concern.
- Per-user theme overrides — not planned.

## Test Strategy

### Backend (Go)
- **Unit tests** (`internal/tenant/service_test.go`):
  - Cache invalidation: verify `InvalidateAndRefresh` returns updated values without a restart.
  - Locale validation: verify `PUT` with `default_locale` not in `available_locales` returns `400`.
  - Logo size limit: verify payload > 1 MB is rejected with `413`.
- **Integration tests** (`internal/tenant/handler_test.go`) using `net/http/httptest`:
  - `GET /api/v1/tenant/config` returns 200 with all five public keys.
  - `PUT /api/v1/tenant/config` with `super_admin` token updates keys and refreshes cache.
  - `PUT /api/v1/tenant/config` with non-admin token returns 403.
  - `GET /api/v1/tenant/logo` returns 204 when logo is `null`.

### Frontend
- `useTenantConfig()` hook test with MSW mock returning the standard config shape — assert returned data matches.
- `<TenantLogo />` component test: when `/api/v1/tenant/logo` returns 204, assert the i18n fallback text (`common.logo_placeholder`) is rendered.

## Notes
- The `logo` value in the database is stored as a base64-encoded string prefixed with its MIME type (e.g. `data:image/png;base64,...`). The `/logo` endpoint decodes the prefix, sets the appropriate `Content-Type`, and streams the decoded bytes.
- Logo uploads are size-limited to 1 MB to prevent oversized payloads in the JSONB column.
- The in-memory cache must be protected by a `sync.RWMutex` so concurrent reads during a write do not cause a data race.
- `available_locales` must always contain `default_locale`; the API validates this constraint on `PUT`.
- Phase 1 is single-tenant; the `tenant_config` table holds exactly one tenant's settings. Multi-tenant schema-per-tenant support is a Phase 6 concern.
