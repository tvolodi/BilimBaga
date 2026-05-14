# FR-BB15 — JWT Middleware

## Metadata
| Field | Value |
|-------|-------|
| ID | FR-BB15 |
| Phase | 1 — Foundation |
| Priority | 1 |
| Status | Draft |
| Depends On | FR-BB14 |

## Description
Provides a reusable Chi middleware that validates the Bearer JWT on every protected route, injects the authenticated user's identity into the request context, and returns standardised error responses for missing or invalid tokens. A companion `RequireAuth` helper and a role-aware `RequirePermission` guard (expanded in FR-BB16) are mounted at the router level, keeping all handlers free of authentication boilerplate. Tenant context is also injected at this middleware layer, establishing the schema/context pattern that Phase 6 multi-tenancy will extend.

## Acceptance Criteria
- [ ] AC-1: Any request to a protected route without an `Authorization: Bearer <token>` header receives a `401 Unauthorized` response with error code `MISSING_TOKEN`.
- [ ] AC-2: A request with an expired JWT receives `401 Unauthorized` with error code `TOKEN_EXPIRED`.
- [ ] AC-3: A request with a malformed or incorrectly signed JWT receives `401 Unauthorized` with error code `INVALID_TOKEN`.
- [ ] AC-4: A valid JWT causes the middleware to inject `user_id` (UUID), `role` (string), and `department_id` (UUID or empty) into the `context.Context` using typed context keys (not raw strings).
- [ ] AC-5: Tenant context (`tenant_id = "public"` in Phase 1) is injected into every request context regardless of authentication status, enabling future multi-tenant routing.
- [ ] AC-6: Protected route handlers can retrieve `user_id`, `role`, and `department_id` from context via exported helper functions (`auth.UserIDFromCtx`, `auth.RoleFromCtx`, `auth.DepartmentIDFromCtx`) without type-asserting raw `interface{}` values.
- [ ] AC-7: Public routes (`/api/v1/health`, `/api/v1/auth/login`, `/api/v1/auth/refresh`, `/api/v1/tenant/config`, `/api/v1/tenant/logo`) are mounted outside the authenticated router group and receive no JWT validation.
- [ ] AC-8: Middleware correctly handles the edge case where the `Authorization` header is present but the value does not start with `Bearer ` (returns `401 INVALID_TOKEN`).

## Technical Specification

### API Endpoints

No new endpoints. This requirement is purely middleware.

### Configuration / Infrastructure

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

#### JWT middleware (`internal/auth/middleware.go`)

```go
package auth

import (
    "net/http"
    "strings"

    "github.com/golang-jwt/jwt/v5"
    "your-module/internal/api"
)

// Authenticate validates a Bearer JWT and injects claims into the context.
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
                switch {
                case isExpired(err):
                    api.WriteError(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "token has expired")
                default:
                    api.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
                }
                return
            }

            ctx := r.Context()
            ctx = context.WithValue(ctx, ctxUserID,       claims.Subject)
            ctx = context.WithValue(ctx, ctxRole,         claims["role"].(string))
            ctx = context.WithValue(ctx, ctxDepartmentID, claims["department_id"].(string))
            ctx = context.WithValue(ctx, ctxTenantID,     "public") // Phase 1 constant

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func parseJWT(raw, secret string) (jwt.MapClaims, error) {
    tok, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, jwt.ErrSignatureInvalid
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

#### Router groups (`internal/router/router.go` sketch)

```go
r := chi.NewRouter()

// Global middleware
r.Use(TenantContext())   // always inject tenant_id

// Public routes
r.Group(func(r chi.Router) {
    r.Get("/api/v1/health", health.Handler)
    r.Post("/api/v1/auth/login", authHandler.Login)
    r.Post("/api/v1/auth/refresh", authHandler.Refresh)
    r.Get("/api/v1/tenant/config", tenantHandler.GetConfig)
    r.Get("/api/v1/tenant/logo", tenantHandler.GetLogo)
})

// Authenticated routes
r.Group(func(r chi.Router) {
    r.Use(auth.Authenticate(cfg.JWTSecret))
    // all protected routes mounted here
})
```

#### Error response format

```json
// 401 — missing token
{
  "data": null,
  "error": { "code": "MISSING_TOKEN", "message": "authorization header required" }
}

// 401 — expired
{
  "data": null,
  "error": { "code": "TOKEN_EXPIRED", "message": "token has expired" }
}

// 401 — invalid signature / malformed
{
  "data": null,
  "error": { "code": "INVALID_TOKEN", "message": "invalid token" }
}

// 403 — authenticated but insufficient role (from RequirePermission in FR-BB16)
{
  "data": null,
  "error": { "code": "FORBIDDEN", "message": "insufficient permissions" }
}
```

## Notes
- Typed context keys (using an unexported `contextKey int` type) prevent accidental key collisions from other middleware packages.
- Never log or expose the raw JWT string in application logs — log only the `user_id` extracted from claims.
- The `TenantContext()` middleware is a stub in Phase 1 that always injects `"public"`. Phase 6 will replace it with a host-based or header-based tenant resolver.
- `department_id` may be an empty string if the user has no department assigned; callers must handle this case.
- Clock skew tolerance of 5 seconds is acceptable for `exp` validation; configure this via `jwt.WithLeeway(5 * time.Second)`.
