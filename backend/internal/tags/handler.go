package tags

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

// List handles GET /api/v1/tags.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.List(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to list tags")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": out, "error": nil})
}

// Create handles POST /api/v1/tags.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}

	tag, err := h.svc.Create(r.Context(), req)
	if err != nil {
		writeTagError(w, err)
		return
	}

	h.writer.Write(r.Context(), r, "tag.create", "tag", &tag.ID, map[string]any{"name": tag.Name})
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": tag, "error": nil})
}

// Update handles PUT /api/v1/tags/{id}. Renames an existing tag in place so
// every question_tags row that references it keeps the same tag_id link.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "invalid JSON body")
		return
	}
	tag, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		writeTagError(w, err)
		return
	}
	h.writer.Write(r.Context(), r, "tag.update", "tag", &tag.ID, map[string]any{"name": tag.Name})
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": tag, "error": nil})
}

// Delete handles DELETE /api/v1/tags/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeTagError(w, err)
		return
	}
	h.writer.Write(r.Context(), r, "tag.delete", "tag", &id, nil)
	w.WriteHeader(http.StatusNoContent)
}

func writeTagError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		api.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "tag not found")
	case errors.Is(err, ErrDuplicate):
		api.WriteError(w, http.StatusConflict, "ERR_TAG_DUPLICATE", "tag with this name already exists")
	case errors.Is(err, ErrTagInUse):
		api.WriteError(w, http.StatusConflict, "ERR_TAG_IN_USE", "tag is referenced by one or more questions")
	case errors.Is(err, ErrInvalidName):
		api.WriteError(w, http.StatusBadRequest, "ERR_INVALID_NAME", "tag name is required and must be at most 64 characters")
	default:
		api.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "failed to process tag request")
	}
}
