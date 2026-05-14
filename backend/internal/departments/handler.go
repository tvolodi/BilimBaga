package departments

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for the departments domain.
type Handler struct {
	svc Service
}

// NewHandler creates a new Handler backed by the given Service.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// ListTree handles GET /api/v1/departments.
// Returns the full department tree as nested JSON.
func (h *Handler) ListTree(w http.ResponseWriter, r *http.Request) {
	tree, err := h.svc.ListTree(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list departments")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": tree, "error": nil})
}

// Create handles POST /api/v1/departments.
// Creates a new department; parent_id is optional.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}
	if req.Name == "" {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "name is required")
		return
	}

	userID := auth.UserIDFromCtx(r.Context())
	ipAddress := r.RemoteAddr

	dept, err := h.svc.Create(r.Context(), req, userID, ipAddress)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "parent department not found")
		case errors.Is(err, ErrDuplicateName):
			api.WriteError(w, http.StatusConflict, "DUPLICATE_NAME", "department name already exists under this parent")
		default:
			api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create department")
		}
		return
	}

	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": dept, "error": nil})
}

// Update handles PUT /api/v1/departments/:id.
// Updates only the name field.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}
	if req.Name == "" {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "name is required")
		return
	}

	userID := auth.UserIDFromCtx(r.Context())
	ipAddress := r.RemoteAddr

	dept, err := h.svc.Update(r.Context(), id, req, userID, ipAddress)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "department not found")
		case errors.Is(err, ErrDuplicateName):
			api.WriteError(w, http.StatusConflict, "DUPLICATE_NAME", "department name already exists under this parent")
		default:
			api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update department")
		}
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{"data": dept, "error": nil})
}

// Delete handles DELETE /api/v1/departments/:id.
// Returns 204 on success, 409 if the department has users or child departments.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	userID := auth.UserIDFromCtx(r.Context())
	ipAddress := r.RemoteAddr

	err := h.svc.Delete(r.Context(), id, userID, ipAddress)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "department not found")
		case errors.Is(err, ErrDepartmentHasChildren):
			api.WriteError(w, http.StatusConflict, "DEPARTMENT_HAS_CHILDREN", "department has child departments")
		case errors.Is(err, ErrDepartmentNotEmpty):
			api.WriteError(w, http.StatusConflict, "DEPARTMENT_NOT_EMPTY", "department has users assigned")
		default:
			api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete department")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
