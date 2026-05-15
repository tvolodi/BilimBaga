package reports

import (
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
)

// Handler handles HTTP requests for the reports domain.
type Handler struct {
	svc Service
}

// NewHandler creates a new Handler backed by the given Service.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// GetDashboard handles GET /api/v1/admin/dashboard (FR-BB51).
// Role restriction (examiner+) is enforced by the router via rbac.RequirePermission.
func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.svc.GetDashboardMetrics(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to load dashboard metrics")
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": metrics, "error": nil})
}
