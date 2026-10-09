package roles

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/go-chi/chi/v5"
)

// auditWriter is the subset of *audit.Writer used by the handler (fakeable in tests).
type auditWriter interface {
	Write(ctx context.Context, r *http.Request, action, entityType string, entityID *string, metadata any)
}

// Handler serves the /roles endpoints.
type Handler struct {
	svc    Service
	writer auditWriter
}

// NewHandler creates a Handler.
func NewHandler(svc Service, writer auditWriter) *Handler {
	return &Handler{svc: svc, writer: writer}
}

func respond(w http.ResponseWriter, status int, data any) {
	api.WriteJSON(w, status, map[string]any{"data": data, "error": nil})
}

func fail(w http.ResponseWriter, op string, err error) {
	var inUse *InUseError
	switch {
	case errors.Is(err, ErrNotFound):
		api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "role not found")
	case errors.Is(err, ErrValidation):
		api.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, ErrNameTaken):
		api.WriteError(w, http.StatusConflict, "ROLE_NAME_TAKEN", "a role with this name already exists")
	case errors.Is(err, ErrSystemImmutable):
		api.WriteError(w, http.StatusConflict, "ROLE_SYSTEM_IMMUTABLE", "system roles cannot be modified or deleted")
	case errors.As(err, &inUse):
		api.WriteErrorWithDetails(w, http.StatusConflict, "ROLE_IN_USE", inUse.Error(), map[string]interface{}{"count": inUse.Count})
	default:
		slog.Error("roles: "+op+" failed", "err", err)
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
	}
}

// List handles GET /roles.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.List(r.Context())
	if err != nil {
		fail(w, "list", err)
		return
	}
	if out == nil {
		out = []Role{}
	}
	respond(w, http.StatusOK, out)
}

// ListPermissions handles GET /roles/permissions.
func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ListPermissions(r.Context())
	if err != nil {
		fail(w, "list permissions", err)
		return
	}
	if out == nil {
		out = []Permission{}
	}
	respond(w, http.StatusOK, out)
}

// Get handles GET /roles/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	role, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		fail(w, "get", err)
		return
	}
	respond(w, http.StatusOK, role)
}

// Create handles POST /roles.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}
	role, err := h.svc.Create(r.Context(), req)
	if err != nil {
		fail(w, "create", err)
		return
	}
	h.writer.Write(r.Context(), r, "role.create", "role", &role.ID,
		map[string]any{"name": role.Name, "permissions": role.Permissions})
	respond(w, http.StatusCreated, role)
}

// Update handles PUT /roles/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}
	res, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		fail(w, "update", err)
		return
	}
	h.writer.Write(r.Context(), r, "role.update", "role", &res.Role.ID, map[string]any{
		"name":                res.Role.Name,
		"permissions_added":   res.PermissionsAdded,
		"permissions_removed": res.PermissionsRemoved,
	})
	respond(w, http.StatusOK, res.Role)
}

// Delete handles DELETE /roles/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	deleted, err := h.svc.Delete(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		fail(w, "delete", err)
		return
	}
	h.writer.Write(r.Context(), r, "role.delete", "role", &deleted.ID, map[string]any{"name": deleted.Name})
	w.WriteHeader(http.StatusNoContent)
}
