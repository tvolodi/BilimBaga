# FR-BB66 Inner Report — Observability

## Metadata

| Field | Value |
|-------|-------|
| Date | 2026-05-16 |
| Pipeline | A (Feature Development) |
| Run ID | FR-BB66 |
| Commit | b3410d2 |
| Commit Message | feat(infra): add structured logging, request IDs, health check, and panic recovery (FR-BB66) |

## Summary

Implemented full observability infrastructure for the BilimBaga Go API:

- **Structured JSON logging** via `zerolog`: every HTTP request emits a single JSON log line with method, path, status code, latency, user_id (or "-" for anonymous), IP, and request_id.
- **Per-request correlation IDs**: UUID generated per request by `RequestID` middleware, stored in context, returned as `X-Request-ID` response header, propagated to all log lines within the request lifetime.
- **LOG_LEVEL env var**: controls zerolog global level (`debug`/`info`/`warn`/`error`); defaults to `info`; invalid values fall back to `info` with a startup warning logged.
- **Health check endpoint** `GET /api/v1/health`: no authentication required; returns `{ "data": { "status": "ok", "db_ok": true, "version": "<git-sha>" }, "error": null }` on success (HTTP 200) or `db_ok: false` (HTTP 503) when `db.Ping()` fails.
- **Panic recovery middleware**: recovers any handler panic, logs full stack trace at error level with the request_id, returns a clean `500 INTERNAL_ERROR` response — no internal detail exposed to client.
- **Slow query logger** (`internal/db/instrumented.go`): wraps DB calls and logs queries exceeding 500ms at warn level with sanitized query text (bind parameters replaced with `?`).
- **Build-time version injection**: Makefile `build` target uses `-ldflags "-X main.Version=$(git rev-parse --short HEAD)"` to embed Git SHA; `var Version = "dev"` in `main.go` as default.
- **Docker Compose log rotation**: all services in `docker-compose.yml`, `deploy/docker-compose.prod.yml`, and `deploy/docker-compose.test.yml` configured with `json-file` driver, `max-size: 100m`, `max-file: 3`.
- **`backend/.env.example`**: documents `LOG_LEVEL=info` for local development.

## Files Changed

| File | Type | Description |
|------|------|-------------|
| `backend/internal/middleware/request_id.go` | New | UUID request ID generation middleware and context helpers |
| `backend/internal/middleware/request_id_test.go` | New | Tests for request ID middleware |
| `backend/internal/middleware/logger.go` | New | Structured JSON request logger middleware (zerolog) |
| `backend/internal/middleware/logger_test.go` | New | Tests for request logger middleware |
| `backend/internal/middleware/recovery.go` | New | Panic recovery middleware |
| `backend/internal/middleware/recovery_test.go` | New | Tests for panic recovery middleware |
| `backend/internal/health/handler.go` | Modified | Health check endpoint (was stub, now full db ping + version) |
| `backend/internal/health/handler_test.go` | Modified | Tests for health handler including 503 DB failure case |
| `backend/internal/db/instrumented.go` | New | Slow query logger (>500ms threshold, sanitized query text) |
| `backend/internal/auth/middleware.go` | Modified | Added SetUserIDInHolder call for context propagation |
| `backend/cmd/api/main.go` | Modified | Version var, initLogger, middleware chain wiring, health route |
| `backend/internal/config/config.go` | Modified | Added LogLevel field |
| `backend/internal/router/router.go` | Modified | Registered new middleware stack and health route |
| `backend/go.mod` | Modified | zerolog declared as direct dependency |
| `backend/go.sum` | Modified | Updated checksums |
| `backend/.env.example` | New | Documents LOG_LEVEL=info |
| `Makefile` | Modified | build target with -ldflags for Git SHA injection |
| `deploy/docker-compose.prod.yml` | Modified | Log rotation for all services |
| `deploy/docker-compose.test.yml` | Modified | Log rotation for all services |
| `docker-compose.yml` | Modified | Log rotation for all 5 services |
| `docs/requirements/FR-BB66.Observability.md` | Modified | Status: Validated → Implemented; ACs checked |
| `docs/requirements/README.md` | Modified | FR-BB66 row status: Draft → Implemented |

## Acceptance Criteria Verification

| AC | Description | Status |
|----|-------------|--------|
| AC-1 | Structured JSON log per request with method/path/status/latency/user_id/ip/request_id | PASS |
| AC-2 | UUID request ID generated per request, stored in context, in every log line | PASS |
| AC-3 | LOG_LEVEL env var; default info; invalid falls back to info with warning | PASS |
| AC-4 | GET /api/v1/health no auth; 200 + db_ok/version on success; 503 + db_ok:false on failure | PASS |
| AC-5 | Panic recovery; stack trace logged at error with request_id; client gets clean 500 INTERNAL_ERROR | PASS |
| AC-6 | Docker Compose log rotation max-size:100m max-file:3 all services | PASS |
| AC-7 | -ldflags -X main.Version in Makefile build target; var Version="dev" default | PASS |
| AC-8 | Slow queries >500ms logged at warn with sanitized query text and latency_ms | PASS |

## Test Results

| Package | Result |
|---------|--------|
| internal/middleware | PASS (cached) |
| internal/health | PASS (cached) |
| internal/auth | PASS (cached) |
| internal/config | PASS (cached) |
| internal/db | PASS (cached) |
| internal/sessions | PASS (cached) |
| internal/tenant | PASS (cached) |
| internal/users | PASS (cached) |
| internal/audit | PASS (cached) |
| internal/categories | PASS (cached) |
| internal/certificates | PASS (cached) |
| internal/departments | PASS (cached) |
| internal/email | PASS (cached) |
| internal/exams | PASS (cached) |
| internal/portal | PASS (cached) |
| internal/questions | PASS (cached) |
| internal/rbac | PASS (cached) |
| internal/reports | PASS (cached) |
| internal/tags | PASS (cached) |
| internal/upload | PASS (cached) |
| **Total** | **21 packages PASS, 0 FAIL** |

## Migration

None required for this requirement.

## Known Limitations

- `zerolog` NDJSON output is wrapped in an additional JSON envelope by Docker's `json-file` log driver; log aggregators (Grafana Loki, Datadog, etc.) must be configured to parse both layers.
- The health endpoint response body always uses 200-level shape (no `error` field populated) even for HTTP 503, per the requirement spec, to allow load balancers that inspect response body to detect degraded state via `db_ok: false`.
- Slow query sanitization uses simple `$\d+` → `?` replacement (safe for PostgreSQL positional parameters); does not handle named parameters or non-standard SQL dialects.
