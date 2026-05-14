package tenant

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bilimbaga/bilimbaga/internal/audit"
)

// Handler handles HTTP requests for tenant configuration.
type Handler struct {
	svc    Service
	writer *audit.Writer
}

// NewHandler creates a new Handler backed by the given Service and audit Writer.
func NewHandler(svc Service, writer *audit.Writer) *Handler {
	return &Handler{svc: svc, writer: writer}
}

type apiResponse struct {
	Data  any `json:"data"`
	Error any `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, data any, apiErr *apiError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResponse{Data: data, Error: apiErr})
}

// GetConfig handles GET /api/v1/tenant/config.
// Returns the public branding subset of tenant configuration.
// Authentication is not required.
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := h.svc.GetPublicConfig()
	writeJSON(w, http.StatusOK, cfg, nil)
}

// UpdateConfig handles PUT /api/v1/tenant/config.
// Accepts a partial JSON object and updates the supplied keys.
// TODO: add super_admin RBAC middleware once FR-BB15/FR-BB16 are implemented.
func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var updates map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeJSON(w, http.StatusBadRequest, nil, &apiError{Code: "INVALID_BODY", Message: "invalid JSON body"})
		return
	}

	updatedKeys, err := h.svc.UpdateConfig(r.Context(), updates)
	if err != nil {
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			status := http.StatusBadRequest
			if valErr.Code == "LOGO_TOO_LARGE" {
				status = http.StatusRequestEntityTooLarge
			}
			writeJSON(w, status, nil, &apiError{Code: valErr.Code, Message: valErr.Message})
			return
		}
		writeJSON(w, http.StatusInternalServerError, nil, &apiError{Code: "INTERNAL_ERROR", Message: "failed to update configuration"})
		return
	}

	h.writer.Write(r.Context(), r, "tenant_config.update", "", nil, map[string]any{"updated_keys": updatedKeys})
	writeJSON(w, http.StatusOK, map[string]any{"updated": updatedKeys}, nil)
}

// GetLogo handles GET /api/v1/tenant/logo.
// Responds with the raw logo bytes and correct Content-Type, or 204 if no logo is set.
// Authentication is not required.
func (h *Handler) GetLogo(w http.ResponseWriter, r *http.Request) {
	imgBytes, contentType, err := h.svc.GetLogoData()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, nil, &apiError{Code: "INTERNAL_ERROR", Message: "failed to retrieve logo"})
		return
	}
	if imgBytes == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(imgBytes)
}
