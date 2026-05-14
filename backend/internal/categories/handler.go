package categories

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/audit"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc    Service
	writer *audit.Writer
}

func NewHandler(svc Service, writer *audit.Writer) *Handler {
	return &Handler{svc: svc, writer: writer}
}

// ListTree handles GET /api/v1/categories.
func (h *Handler) ListTree(w http.ResponseWriter, r *http.Request) {
	tree, err := h.svc.ListTree(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to list categories")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": tree, "error": nil})
}

// Create handles POST /api/v1/categories.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}

	node, err := h.svc.Create(r.Context(), req)
	if err != nil {
		writeCategoryError(w, err)
		return
	}

	h.writer.Write(r.Context(), r, "category.create", "category", &node.ID, map[string]any{
		"name":       node.Name,
		"parent_id":  node.ParentID,
		"track":      node.Track,
		"sort_order": node.SortOrder,
	})
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": node, "error": nil})
}

// Update handles PUT /api/v1/categories/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}

	node, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		writeCategoryError(w, err)
		return
	}

	diff := map[string]any{}
	if req.Name != nil {
		diff["name"] = node.Name
	}
	if req.Track != nil {
		diff["track"] = node.Track
	}
	if req.SortOrder != nil {
		diff["sort_order"] = node.SortOrder
	}
	if req.ClearParent || req.ParentID != nil {
		diff["parent_id"] = node.ParentID
	}
	h.writer.Write(r.Context(), r, "category.update", "category", &node.ID, diff)
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": node, "error": nil})
}

// Delete handles DELETE /api/v1/categories/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeCategoryError(w, err)
		return
	}
	h.writer.Write(r.Context(), r, "category.delete", "category", &id, nil)
	w.WriteHeader(http.StatusNoContent)
}

func writeCategoryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "category not found")
	case errors.Is(err, ErrParentNotFound):
		api.WriteError(w, http.StatusBadRequest, "ERR_PARENT_NOT_FOUND", "parent category not found")
	case errors.Is(err, ErrCycle):
		api.WriteError(w, http.StatusBadRequest, "ERR_CATEGORY_CYCLE", "cyclic parent reference")
	case errors.Is(err, ErrCategoryInUse):
		api.WriteError(w, http.StatusConflict, "ERR_CATEGORY_IN_USE", "category has child categories or referenced questions")
	case errors.Is(err, ErrInvalidName):
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_NAME", "name is required and must be at most 128 characters")
	default:
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to process category request")
	}
}
