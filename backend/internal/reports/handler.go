package reports

import (
	"errors"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/go-chi/chi/v5"
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

// GetExamAnalytics handles GET /api/v1/admin/exams/{id}/analytics (FR-BB52).
// Role restriction (examiner+) is enforced by the router via rbac.RequirePermission.
func (h *Handler) GetExamAnalytics(w http.ResponseWriter, r *http.Request) {
	examID := chi.URLParam(r, "id")
	if examID == "" {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_PARAM", "exam id is required")
		return
	}

	result, err := h.svc.GetExamAnalytics(r.Context(), examID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, "EXAM_NOT_FOUND", "exam not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load exam analytics")
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": result, "error": nil})
}
