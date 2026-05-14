package router

import (
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/bilimbaga/bilimbaga/internal/health"
	"github.com/bilimbaga/bilimbaga/internal/tenant"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// New creates and returns a configured Chi router with all registered routes.
// jwtSecret is passed to auth.Authenticate() so the router package never calls os.Getenv().
func New(tenantHandler *tenant.Handler, authHandler *auth.Handler, jwtSecret string) *chi.Mux {
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

			// Tenant configuration — protected endpoint.
			// TODO: add super_admin RBAC guard once FR-BB16 is implemented.
			r.Put("/tenant/config", tenantHandler.UpdateConfig)

			// All subsequent authenticated routes are registered here.
		})
	})

	return r
}
