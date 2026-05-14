package rbac

import (
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/auth"
)

// RequirePermission returns a Chi-compatible middleware that enforces a permission check.
// It must be composed after auth.Authenticate in the middleware chain.
// If no role is present in the request context, it short-circuits with 401.
// If the role does not hold the requested permission, it returns 403.
func RequirePermission(cache *Cache, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := auth.RoleFromCtx(r.Context())
			if role == "" {
				api.WriteError(w, http.StatusUnauthorized, "MISSING_TOKEN", "authentication required")
				return
			}
			if !cache.Has(role, resource, action) {
				api.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
