package router

import (
	"github.com/bilimbaga/bilimbaga/internal/health"
	"github.com/bilimbaga/bilimbaga/internal/tenant"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// New creates and returns a configured Chi router with all registered routes.
func New(tenantHandler *tenant.Handler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", health.Handler())

		// Tenant configuration — public endpoints (no auth required).
		r.Get("/tenant/config", tenantHandler.GetConfig)
		r.Get("/tenant/logo", tenantHandler.GetLogo)

		// Tenant configuration — protected endpoint.
		// TODO: wrap with JWT + super_admin RBAC middleware once FR-BB15/FR-BB16 are implemented.
		r.Put("/tenant/config", tenantHandler.UpdateConfig)
	})

	return r
}
