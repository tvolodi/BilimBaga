package auth

import (
	"context"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
)

// TenantContext returns a middleware that injects the tenant ID into every
// request context. In Phase 1 the tenant is always "public". Phase 6 will
// replace this with host-based or header-based tenant resolution.
func TenantContext() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ctxkeys.CtxTenantID, "public")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
