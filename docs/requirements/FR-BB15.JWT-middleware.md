# FR-BB15 — JWT Middleware

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB15 |
| Phase | 1 — Foundation |
| Priority | 1 |
| Status | Implemented |
| Depends On | FR-BB14 |

## Scope

| Layer | Items |
|-------|-------|
| New files | `internal/api/response.go`, `internal/auth/context.go`, `internal/auth/middleware.go`, `internal/auth/tenant.go` |
| Modified files | `internal/router/router.go` — add `jwtSecret string` parameter, wire `TenantContext()` globally, move `change-password` to protected group |
| Database | None |
| Frontend | None |

## Description
Provides a reusable Chi middleware that validates the Bearer JWT on every protected route, injects the authenticated user's identity into the request context, and returns standardised error responses for missing or invalid tokens. A companion `RequireAuth` helper and a role-aware `RequirePermission` guard (expanded in FR-BB16) are mounted at the router level, keeping all handlers free of authentication boilerplate. Tenant context is injected globally by a dedicated `TenantContext()` middleware (not by the JWT middleware), establishing the schema/context pattern that Phase 6 multi-tenancy will extend. A shared `internal/api` package provides the standard JSON response helpers used by all handlers.

## Acceptance Criteria
- [ ] AC-1: Any request to a protected route without an `Authorization: Bearer <token>` header receives a `401 Unauthorized` response with error code `MISSING_TOKEN`.
- [ ] AC-2: A request with an expired JWT receives `401 Unauthorized` with error code `TOKEN_EXPIRED`.
- [ ] AC-3: A request with a malformed or incorrectly signed JWT receives `401 Unauthorized` with error code `INVALID_TOKEN`.
- [ ] AC-4: A valid JWT causes the middleware to inject `user_id` (UUID), `role` (string), and `department_id` (UUID or empty) into the `context.Context` using typed context keys (not raw strings).
- [ ] AC-5: Tenant context (`tenant_id = "public"` in Phase 1) is injected into every request context by the global `TenantContext()` middleware, regardless of authentication status. The `Authenticate()` middleware does NOT inject tenant context — that is the sole responsibility of `TenantContext()`.
- [ ] AC-6: Protected route handlers can retrieve `user_id`, `role`, and `department_id` from context via exported helper functions (`auth.UserIDFromCtx`, `auth.RoleFromCtx`, `auth.DepartmentIDFromCtx`) without type-asserting raw `interface{}` values.
- [ ] AC-7: Routes are grouped by authentication requirement as follows:
  - **Public (no JWT required)**: `GET /api/v1/health`, `POST /api/v1/auth/login`, `POST /api/v1/auth/refresh`, `POST /api/v1/auth/logout` (intentionally public — server-side logout is a no-op token invalidation in Phase 1; client discards the token), `GET /api/v1/tenant/config`, `GET /api/v1/tenant/logo`.
  - **Protected (Bearer JWT required)**: `POST /api/v1/auth/change-password` (requires an authenticated user per FR-BB14) and all subsequent authenticated routes.
- [ ] AC-8: Middleware correctly handles the edge case where the `Authorization` header is present but the value does not start with `Bearer ` (returns `401 INVALID_TOKEN`).
- [ ] AC-9: If the JWT `role` claim is absent or not a string the middleware returns `401 INVALID_TOKEN` without panicking. The `department_id` claim is optional and defaults to an empty string when absent.

## Technical Specification

### API Endpoints

No new endpoints. This requirement is purely middleware.

### Configuration / Infrastructure

#### Shared API response helpers (`internal/api/response.go`)

This package provides the standard BilimBaga JSON envelope used by all handlers and middleware. It must be created as part of this requirement because `Authenticate()` depends on it.

```go
package api

import (
	"encoding/json"
	"net/http"
)

// WriteJSON encodes payload as JSON and writes it with the given HTTP status.
func WriteJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload) //nolint:errcheck
}

// WriteError writes a standard BilimBaga error envelope:
//
//	{ "data": null, "error": { "code": "...", "message": "..." } }
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, map[string]interface{}{
		"data":  nil,
		"error": map[string]string{"code": code, "message": message},
	})
}
```

#### Context key types (`internal/auth/context.go`)

```go
package auth

import "context"

type contextKey int

const (
	ctxUserID       contextKey = iota
	ctxRole
	ctxDepartmentID
	ctxTenantID
)

func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxUserID).(string)
	return v
}

func RoleFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxRole).(string)
	return v
}

func DepartmentIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxDepartmentID).(string)
	return v
}

func TenantIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxTenantID).(string)
	return v
}
```

#### Tenant context middleware (`internal/auth/tenant.go`)

`TenantContext()` is a global middleware that injects `tenant_id` into every request context before any route handler or auth middleware runs. In Phase 1 it always injects the constant `"public"`.

```go
package auth

import (
	"context"
	"net/http"
)

// TenantContext returns a middleware that injects the tenant ID into every
// request context. In Phase 1 the tenant is always "public". Phase 6 will
// replace this with host-based or header-based tenant resolution.
func TenantContext() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ctxTenantID, "public")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

#### JWT middleware (`internal/auth/middleware.go`)

```go
package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/bilimbaga/bilimbaga/internal/api"
)

// Authenticate validates a Bearer JWT and injects user claims into the context.
// It does NOT inject tenant context — that is handled by TenantContext().
func Authenticate(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				api.WriteError(w, http.StatusUnauthorized, "MISSING_TOKEN", "authorization header required")
				return
			}
			if !strings.HasPrefix(header, "Bearer ") {
				api.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "malformed authorization header")
				return
			}
			raw := strings.TrimPrefix(header, "Bearer ")
			claims, err := parseJWT(raw, jwtSecret)
			if err != nil {
				if isExpired(err) {
					api.WriteError(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "token has expired")
				} else {
					api.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
				}
				return
			}

			// Safe claim extraction — return INVALID_TOKEN instead of panicking.
			userID, ok := claims["sub"].(string)
			if !ok || userID == "" {
				api.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token claims")
				return
			}
			role, ok := claims["role"].(string)
			if !ok {
				api.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token claims")
				return
			}
			deptID, _ := claims["department_id"].(string) // optional; defaults to ""

			ctx := r.Context()
			ctx = context.WithValue(ctx, ctxUserID, userID)
			ctx = context.WithValue(ctx, ctxRole, role)
			ctx = context.WithValue(ctx, ctxDepartmentID, deptID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func parseJWT(raw, secret string) (jwt.MapClaims, error) {
	tok, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok || !tok.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func isExpired(err error) bool {
	return errors.Is(err, jwt.ErrTokenExpired)
}
```

#### `router.New()` signature update

`router.New()` must be updated to accept `jwtSecret string` as a third parameter so that `auth.Authenticate(jwtSecret)` can be wired in without importing `config` into the router package:

```go
// Before (FR-BB14):
func New(tenantHandler *tenant.Handler, authHandler *auth.Handler) *chi.Mux

// After (FR-BB15):
func New(tenantHandler *tenant.Handler, authHandler *auth.Handler, jwtSecret string) *chi.Mux
```

The call site in `cmd/api/main.go` must pass `cfg.JWTSecret`.

#### Router groups (`internal/router/router.go` updated sketch)

```go
func New(tenantHandler *tenant.Handler, authHandler *auth.Handler, jwtSecret string) *chi.Mux {
	r := chi.NewRouter()

	// Standard Chi middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Global middleware — injects tenant_id for every request (public and protected alike)
	r.Use(auth.TenantContext())

	r.Route("/api/v1", func(r chi.Router) {
		// Public routes — no JWT validation
		r.Get("/health", health.Handler())
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)
		r.Post("/auth/logout", authHandler.Logout) // public; client discards token
		r.Get("/tenant/config", tenantHandler.GetConfig)
		r.Get("/tenant/logo", tenantHandler.GetLogo)

		// Protected routes — Bearer JWT required
		r.Group(func(r chi.Router) {
			r.Use(auth.Authenticate(jwtSecret))
			r.Post("/auth/change-password", authHandler.ChangePassword)
			// All subsequent authenticated routes are registered here.
		})
	})

	return r
}
```

#### Error response format

```json
// 401 — missing token
{ "data": null, "error": { "code": "MISSING_TOKEN", "message": "authorization header required" } }

// 401 — expired
{ "data": null, "error": { "code": "TOKEN_EXPIRED", "message": "token has expired" } }

// 401 — invalid signature / malformed / bad claims
{ "data": null, "error": { "code": "INVALID_TOKEN", "message": "invalid token" } }

// 403 — authenticated but insufficient role (from RequirePermission in FR-BB16)
{ "data": null, "error": { "code": "FORBIDDEN", "message": "insufficient permissions" } }
```

## Out of Scope

- **Role-permission checks** — fine-grained permission enforcement is defined in FR-BB16 (`RequirePermission` guard).
- **Tenant resolution by hostname or header** — Phase 1 always uses `"public"`; hostname-based resolution is a Phase 6 concern.
- **Refresh token rotation** — token lifecycle and refresh logic are defined in FR-BB14.
- **Token revocation / blacklisting** — not implemented in Phase 1; logout is client-side only.
- **Rate limiting on auth endpoints** — out of scope for this requirement.

## Notes
- Typed context keys (using an unexported `contextKey int` type) prevent accidental key collisions from other middleware packages.
- Never log or expose the raw JWT string in application logs — log only the `user_id` extracted from claims.
- `department_id` may be an empty string if the user has no department assigned; callers must handle this case.
- Clock skew tolerance of 5 seconds is acceptable for `exp` validation; configure this via `jwt.WithLeeway(5 * time.Second)`.

## Test Strategy

### Unit tests (`internal/auth/middleware_test.go`)

Table-driven tests for `Authenticate()` covering every AC outcome:

| Test case | Input | Expected HTTP status | Expected error code |
|-----------|-------|---------------------|---------------------|
| Missing header | No `Authorization` header | 401 | `MISSING_TOKEN` |
| Non-Bearer prefix | `Authorization: Basic abc` | 401 | `INVALID_TOKEN` |
| Expired token | Valid structure, `exp` in the past | 401 | `TOKEN_EXPIRED` |
| Wrong signature | Token signed with a different secret | 401 | `INVALID_TOKEN` |
| Missing `role` claim | Token without `role` field | 401 | `INVALID_TOKEN` |
| Missing `sub` claim | Token without `sub` field | 401 | `INVALID_TOKEN` |
| Valid token | Correctly signed token with all claims | 200 | — |
| Valid token, no `department_id` | Token without `department_id` | 200 (empty string injected) | — |

### Unit tests (`internal/auth/tenant_test.go`)

- `TenantContext()` injects `tenant_id = "public"` for every request, including unauthenticated ones.
- `TenantIDFromCtx` returns `"public"` after `TenantContext()` runs.

### Integration tests

- Sending a request to `POST /api/v1/auth/login` (public) without an `Authorization` header returns `200`, confirming the public group bypasses `Authenticate()`.
- Sending a request to `POST /api/v1/auth/change-password` without a token returns `401 MISSING_TOKEN`, confirming it is in the protected group.
- Sending a request to `POST /api/v1/auth/logout` without a token returns a non-401 response, confirming it remains public.
