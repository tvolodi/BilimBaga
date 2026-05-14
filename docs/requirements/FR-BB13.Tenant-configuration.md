# FR-BB13 — Tenant Configuration

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB13 |
| Phase | 1 — Foundation |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB12 |

## Description
Provides a runtime-configurable branding and localisation store for the platform. A single `tenant_config` table holds key/value pairs (JSONB values) covering application name, logo, colours, and locale settings. The Go service loads and caches this configuration at startup so that every request has access to branding data without a round-trip to the database. Super admins may update configuration at runtime via a protected API endpoint; public endpoints expose the subset of data needed by the frontend before authentication.

## Acceptance Criteria
- [ ] AC-1: Migration creates the `tenant_config` table with `key TEXT PRIMARY KEY`, `value JSONB NOT NULL`, and `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()` columns.
- [ ] AC-2: Seed data populates default rows for all six keys: `app_name`, `logo`, `primary_color`, `accent_color`, `default_locale`, `available_locales`.
- [ ] AC-3: On API startup, all rows are read from `tenant_config` and stored in an in-memory cache; subsequent reads from the cache do not hit the database.
- [ ] AC-4: `GET /api/v1/tenant/config` returns `app_name`, `primary_color`, `accent_color`, `default_locale`, and `available_locales` without requiring authentication.
- [ ] AC-5: `GET /api/v1/tenant/logo` returns the raw logo bytes with the correct `Content-Type` header (e.g. `image/png`, `image/svg+xml`) derived from the stored base64 metadata.
- [ ] AC-6: `PUT /api/v1/tenant/config` accepts a partial update object; only provided keys are modified; missing keys retain their existing values.
- [ ] AC-7: `PUT /api/v1/tenant/config` is accessible only to users with the `super_admin` role; any other role receives `403 Forbidden`.
- [ ] AC-8: After a successful `PUT /api/v1/tenant/config`, the in-memory cache is invalidated and refreshed so subsequent `GET` requests return the new values without a service restart.
- [ ] AC-9: All config update events are written to `audit_log` with `action = "tenant_config.update"` and a metadata payload listing the changed keys.

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

### Frontend Components
- `useTenantConfig()` — React Query hook that fetches `GET /api/v1/tenant/config` and caches it for the session lifetime; consumed before rendering `<App />`.
- CSS custom properties (`--color-primary`, `--color-accent`) injected into `:root` from the config response so Tailwind's `primary` and `accent` colour tokens resolve to tenant values.
- `<TenantLogo />` — renders `<img src="/api/v1/tenant/logo" />` with a fallback text placeholder if the server returns 204.

## Notes
- The `logo` value in the database is stored as a base64-encoded string prefixed with its MIME type (e.g. `data:image/png;base64,...`). The `/logo` endpoint decodes the prefix, sets the appropriate `Content-Type`, and streams the decoded bytes.
- Logo uploads are size-limited to 1 MB to prevent oversized payloads in the JSONB column.
- The in-memory cache must be protected by a `sync.RWMutex` so concurrent reads during a write do not cause a data race.
- `available_locales` must always contain `default_locale`; the API validates this constraint on `PUT`.
- Phase 1 is single-tenant; the `tenant_config` table holds exactly one tenant's settings. Multi-tenant schema-per-tenant support is a Phase 6 concern.
