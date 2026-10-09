package audit

import (
	"encoding/csv"
	"fmt"
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

	filters, ok := parseFilters(w, r)
	if !ok {
		return
	}

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
// Returns the filtered audit log as a CSV file with columns matching AC-8:
// timestamp, actor_id, actor_name, action, entity_type, entity_id, ip_address, metadata_json.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	tenantID := ctxkeys.TenantIDFromCtx(r.Context())
	if tenantID == "" {
		api.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing tenant context")
		return
	}

	filters, ok := parseFilters(w, r)
	if !ok {
		return
	}

	entries, err := h.svc.Export(r.Context(), tenantID, filters)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to export audit log")
		return
	}

	filename := fmt.Sprintf("audit-%s.csv", time.Now().UTC().Format("20060102"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)

	csvWriter := csv.NewWriter(w)
	_ = csvWriter.Write([]string{"timestamp", "actor_id", "actor_name", "action", "entity_type", "entity_id", "ip_address", "metadata_json"})

	for _, e := range entries {
		actorID := ""
		if e.ActorID != nil {
			actorID = *e.ActorID
		}
		actorName := ""
		if e.ActorName != nil {
			actorName = *e.ActorName
		}
		entityType := ""
		if e.EntityType != nil {
			entityType = *e.EntityType
		}
		entityID := ""
		if e.EntityID != nil {
			entityID = *e.EntityID
		}
		ip := ""
		if e.IP != nil {
			ip = *e.IP
		}
		metadata := ""
		if e.Metadata != nil {
			metadata = string(e.Metadata)
		}
		_ = csvWriter.Write([]string{
			e.CreatedAt.UTC().Format(time.RFC3339),
			actorID,
			actorName,
			e.Action,
			entityType,
			entityID,
			ip,
			metadata,
		})
	}
	csvWriter.Flush()
}

// parseFilters reads the allowed query params from r and populates AuditFilters.
// Only explicitly allowlisted params are mapped — free-form WHERE injection is not possible.
// The "actor" param is a free-text partial match on actor full_name (ILIKE).
// The "actor_id" param is an exact UUID match.
func parseFilters(w http.ResponseWriter, r *http.Request) (AuditFilters, bool) {
	var f AuditFilters
	v, ok := api.UUIDQuery(w, r, "actor_id")
	if !ok {
		return f, false
	}
	if v != "" {
		f.ActorID = &v
	}
	if v := r.URL.Query().Get("actor"); v != "" {
		f.Actor = &v
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
	return f, true
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
