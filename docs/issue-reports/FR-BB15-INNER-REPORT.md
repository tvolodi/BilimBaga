# FR-BB15: Implementation Inner Report

**Date**: 2026-05-14T00:00:00Z
**Pipeline**: A
**Commit**: (pending)

## Summary

Implemented the JWT authentication middleware layer for BilimBaga (FR-BB15). This delivers a reusable `Authenticate()` Chi middleware that validates Bearer tokens on every protected route, injects typed claims (user ID, role, department ID, tenant ID) into the request context via safe context-key helpers, and a global `TenantContext()` middleware that seeds the tenant slug on every request. A shared `WriteJSON`/`WriteError` response utility was extracted to `internal/api/response.go` to unify API response formatting. The router was updated to accept `jwtSecret` at construction time and wire both middlewares correctly; `change-password` was moved into the authenticated route group.

## Files Changed

| File | Action |
|------|--------|
| `backend/internal/api/response.go` | created — shared WriteJSON/WriteError utilities |
| `backend/internal/auth/context.go` | created — typed context keys + UserIDFromCtx, RoleFromCtx, DepartmentIDFromCtx, TenantIDFromCtx |
| `backend/internal/auth/middleware.go` | created — Authenticate() JWT middleware with parseJWT, isExpired |
| `backend/internal/auth/tenant.go` | created — TenantContext() global middleware |
| `backend/internal/auth/middleware_test.go` | created — 8 table-driven unit tests covering all AC items |
| `backend/internal/auth/tenant_test.go` | created — 2 tenant middleware tests |
| `backend/internal/router/router.go` | modified — New() accepts jwtSecret; TenantContext wired globally; change-password in protected group |
| `backend/cmd/api/main.go` | modified — passes cfg.JWTSecret to router.New() |
| `backend/go.mod` | modified — added github.com/golang-jwt/jwt/v5 v5.3.1 and golang.org/x/crypto v0.51.0 |
| `backend/go.sum` | modified — updated checksums |
| `backend/internal/config/config.go` | modified — whitespace/alignment cleanup |
| `docs/requirements/FR-BB15.JWT-middleware.md` | modified — status → Implemented |
| `docs/requirements/README.md` | modified — status updated to Implemented |

## Acceptance Criteria Verified

| AC | Verified By |
|----|-------------|
| AC-1: Missing Authorization header → 401 MISSING_TOKEN | test: TestAuthenticate/AC-1 |
| AC-2: Expired token → 401 TOKEN_EXPIRED | test: TestAuthenticate/AC-2 |
| AC-3: Wrong signature → 401 INVALID_TOKEN | test: TestAuthenticate/AC-3 |
| AC-4: Valid token → 200, claims injected into context | test: TestAuthenticate/AC-4 |
| AC-5: TenantContext injects tenant_id on every request | test: TestTenantContext/AC-5 |
| AC-8: Non-Bearer prefix → 401 INVALID_TOKEN | test: TestAuthenticate/AC-8 |
| AC-9: Missing role/sub claim → 401 INVALID_TOKEN | test: TestAuthenticate/AC-9 |
| AC-9+: Valid token without department_id → 200, empty department | test: TestAuthenticate/AC-4+AC-9 |

## Test Results

- Backend: 25 passed, 0 failed (internal/auth package)
- All suites: ok internal/auth, ok internal/config, ok internal/db, ok internal/health, ok internal/tenant

## Migration Applied

None — FR-BB15 is a middleware layer only; no schema changes required.

## Notable Decisions

- Typed unexported context key type (`ctxKey`) avoids collisions with third-party middleware.
- `isExpired()` extracted as a pure function for ease of testing without real clock dependency.
- `TenantContext()` defaults to `"public"` slug when no `X-Tenant-ID` header is present, ensuring all routes have a tenant in context.
- `WriteJSON`/`WriteError` in `internal/api` (not `internal/auth`) to avoid import cycles between auth middleware and the API layer.

## Known Limitations

- Tenant resolution currently uses a static header value (`X-Tenant-ID`); dynamic DB-backed lookup is deferred to the tenant management feature (FR-BB22).
- Role-based access control (require specific roles) is implemented as a separate `RequireRole()` middleware, deferred to the RBAC feature.
