package router

import (
	"github.com/bilimbaga/bilimbaga/internal/audit"
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/bilimbaga/bilimbaga/internal/departments"
	"github.com/bilimbaga/bilimbaga/internal/health"
	"github.com/bilimbaga/bilimbaga/internal/rbac"
	"github.com/bilimbaga/bilimbaga/internal/tenant"
	"github.com/bilimbaga/bilimbaga/internal/users"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// New creates and returns a configured Chi router with all registered routes.
// jwtSecret is passed to auth.Authenticate() so the router package never calls os.Getenv().
// rbacCache is the in-memory permission cache loaded at startup.
func New(tenantHandler *tenant.Handler, authHandler *auth.Handler, deptHandler *departments.Handler, usersHandler *users.Handler, auditHandler *audit.Handler, jwtSecret string, rbacCache *rbac.Cache) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Global middleware — injects tenant_id for every request (public and protected alike).
	r.Use(auth.TenantContext())

	r.Route("/api/v1", func(r chi.Router) {
		// Public routes — no JWT validation.
		r.Get("/health", health.Handler())
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)
		r.Post("/auth/logout", authHandler.Logout)
		r.Get("/tenant/config", tenantHandler.GetConfig)
		r.Get("/tenant/logo", tenantHandler.GetLogo)

		// Protected routes — Bearer JWT required.
		r.Group(func(r chi.Router) {
			r.Use(auth.Authenticate(jwtSecret))
			r.Post("/auth/change-password", authHandler.ChangePassword)

			// Tenant configuration — requires super_admin.
			r.With(rbac.RequirePermission(rbacCache, "tenant", "manage")).
				Put("/tenant/config", tenantHandler.UpdateConfig)

			// Department management.
			r.With(rbac.RequirePermission(rbacCache, "departments", "read")).
				Get("/departments", deptHandler.ListTree)
			r.With(rbac.RequirePermission(rbacCache, "departments", "manage")).
				Post("/departments", deptHandler.Create)
			r.With(rbac.RequirePermission(rbacCache, "departments", "manage")).
				Put("/departments/{id}", deptHandler.Update)
			r.With(rbac.RequirePermission(rbacCache, "departments", "manage")).
				Delete("/departments/{id}", deptHandler.Delete)

			// User management.
			// GetMe is available to every authenticated user — no extra permission required.
			r.Get("/users/me", usersHandler.GetMe)
			// Import is a static sub-path; must be registered before /{id} routes.
			r.With(rbac.RequirePermission(rbacCache, "users", "manage")).
				Post("/users/import", usersHandler.ImportUsers)
			r.With(rbac.RequirePermission(rbacCache, "users", "read")).
				Get("/users", usersHandler.ListUsers)
			r.With(rbac.RequirePermission(rbacCache, "users", "manage")).
				Post("/users", usersHandler.CreateUser)
			// Per-user routes — GET self-access is enforced inside the handler.
			r.Get("/users/{id}", usersHandler.GetUser)
			r.With(rbac.RequirePermission(rbacCache, "users", "manage")).
				Put("/users/{id}", usersHandler.UpdateUser)
			r.With(rbac.RequirePermission(rbacCache, "users", "manage")).
				Post("/users/{id}/deactivate", usersHandler.DeactivateUser)
			r.With(rbac.RequirePermission(rbacCache, "users", "manage")).
				Post("/users/{id}/reset-password", usersHandler.ResetPassword)

			// Audit log.
			r.With(rbac.RequirePermission(rbacCache, "audit", "read")).
				Get("/audit", auditHandler.List)
			r.With(rbac.RequirePermission(rbacCache, "audit", "read")).
				Get("/audit/export", auditHandler.Export)
		})
	})

	return r
}
