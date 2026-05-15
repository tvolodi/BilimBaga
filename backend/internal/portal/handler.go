package portal

import (
	"errors"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for the employee exam portal.
type Handler struct {
	svc Service
}

// NewHandler creates a new Handler backed by the given Service.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// ListMyExams handles GET /api/v1/portal/exams.
// Returns all active exams assigned to the authenticated user with computed status.
func (h *Handler) ListMyExams(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	deptID := auth.DepartmentIDFromCtx(r.Context())

	items, err := h.svc.ListMyExams(r.Context(), userID, deptID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to list exams")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": items, "error": nil})
}

// GetMyExam handles GET /api/v1/portal/exams/:id.
// Returns the full detail of one exam assigned to the authenticated user.
// Returns 403 if the exam exists but is not assigned (AC-7).
func (h *Handler) GetMyExam(w http.ResponseWriter, r *http.Request) {
	examID := chi.URLParam(r, "id")
	userID := auth.UserIDFromCtx(r.Context())
	deptID := auth.DepartmentIDFromCtx(r.Context())

	detail, err := h.svc.GetMyExam(r.Context(), examID, userID, deptID)
	if err != nil {
		if errors.Is(err, ErrNotAssigned) {
			api.WriteJSON(w, http.StatusForbidden, map[string]any{
				"data": nil,
				"error": map[string]string{
					"code":    "EXAM_NOT_ASSIGNED",
					"message": "You do not have access to this exam.",
				},
			})
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to get exam")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": detail, "error": nil})
}
