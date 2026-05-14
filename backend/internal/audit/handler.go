package audit

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
)

// Handler handles HTTP requests for the audit domain.
type Handler struct {
	svc    Service
	writer *Writer
}

// NewHandler creates a new Handler backed by the given Service and Writer.
func NewHandler(svc Service, writer *Writer) *Handler {
	return &Handler{svc: svc, writer: writer}
}

// List handles GET /api/v1/audit.
// Returns a paginated, tenant-scoped audit log.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := ctxkeys.TenantIDFromCtx(r.Context())
	if tenantID == "" {
		api.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing tenant context")
		return
	}

	page := parseIntParam(r, "page", 1)
	perPage := parseIntParam(r, "per_page", 50)

	filters := parseFilters(r)

	entries, total, err := h.svc.List(r.Context(), tenantID, filters, page, perPage)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list audit log")
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"items": entries,
			"meta": map[string]any{
				"page":     page,
				"per_page": perPage,
				"total":    total,
			},
		},
		"error": nil,
	})
}

// Export handles GET /api/v1/audit/export.
// Returns the filtered audit log as a CSV file.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	tenantID := ctxkeys.TenantIDFromCtx(r.Context())
	if tenantID == "" {
		api.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing tenant context")
		return
	}

	filters := parseFilters(r)

	entries, err := h.svc.Export(r.Context(), tenantID, filters)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to export audit log")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="audit_export.csv"`)
	w.WriteHeader(http.StatusOK)

	csvWriter := csv.NewWriter(w)
	_ = csvWriter.Write([]string{"id", "actor_id", "action", "entity_type", "entity_id", "ip", "metadata", "created_at"})

	for _, e := range entries {
		actorID := ""
		if e.ActorID != nil {
			actorID = *e.ActorID
		}
		entityType := ""
		if e.EntityType != nil {
			entityType = *e.EntityType
		}
		entityID := ""
		if e.EntityID != nil {
			entityID = *e.EntityID
		}
		metadata := ""
		if e.Metadata != nil {
			metadata = string(e.Metadata)
		}
		_ = csvWriter.Write([]string{
			e.ID,
			actorID,
			e.Action,
			entityType,
			entityID,
			e.IP,
			metadata,
			e.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	csvWriter.Flush()
}

// parseFilters reads the allowed query params from r and populates AuditFilters.
// Only explicitly allowlisted params are mapped — free-form WHERE injection is not possible.
func parseFilters(r *http.Request) AuditFilters {
	var f AuditFilters
	if v := r.URL.Query().Get("actor_id"); v != "" {
		f.ActorID = &v
	}
	if v := r.URL.Query().Get("action"); v != "" {
		f.Action = &v
	}
	if v := r.URL.Query().Get("entity_type"); v != "" {
		f.EntityType = &v
	}
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.From = &t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.To = &t
		}
	}
	return f
}

// parseIntParam reads an integer query param, returning defaultVal on missing or invalid input.
func parseIntParam(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}
