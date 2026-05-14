package users

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for the users domain.
type Handler struct {
	svc Service
}

// NewHandler creates a new Handler backed by the given Service.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// ListUsers handles GET /api/v1/users.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	callerRole := auth.RoleFromCtx(r.Context())
	callerDeptID := auth.DepartmentIDFromCtx(r.Context())

	f := ListFilters{
		Page:    parseIntParam(r, "page", 1),
		PerPage: parseIntParam(r, "per_page", 20),
	}
	if v := r.URL.Query().Get("department_id"); v != "" {
		f.DepartmentID = &v
	}
	if v := r.URL.Query().Get("role_id"); v != "" {
		f.RoleID = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		f.Status = &v
	}

	result, err := h.svc.ListUsers(r.Context(), callerRole, callerDeptID, f)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list users")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": result, "error": nil})
}

// GetMe handles GET /api/v1/users/me.
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	callerUserID := auth.UserIDFromCtx(r.Context())
	u, err := h.svc.GetMe(r.Context(), callerUserID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get profile")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": u, "error": nil})
}

// GetUser handles GET /api/v1/users/:id.
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	callerRole := auth.RoleFromCtx(r.Context())
	callerUserID := auth.UserIDFromCtx(r.Context())
	callerDeptID := auth.DepartmentIDFromCtx(r.Context())

	u, err := h.svc.GetUser(r.Context(), id, callerRole, callerUserID, callerDeptID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		case errors.Is(err, ErrForbidden):
			api.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		default:
			api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get user")
		}
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": u, "error": nil})
}

// CreateUser handles POST /api/v1/users.
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}

	callerRole := auth.RoleFromCtx(r.Context())
	callerDeptID := auth.DepartmentIDFromCtx(r.Context())
	callerUserID := auth.UserIDFromCtx(r.Context())

	resp, err := h.svc.CreateUser(r.Context(), req, callerRole, callerDeptID, callerUserID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, ErrValidation):
			api.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		case errors.Is(err, ErrForbidden):
			api.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		case errors.Is(err, ErrDuplicateEmail):
			api.WriteError(w, http.StatusConflict, "DUPLICATE_EMAIL", "a user with this email already exists")
		default:
			api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create user")
		}
		return
	}
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": resp, "error": nil})
}

// UpdateUser handles PUT /api/v1/users/:id.
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}

	callerRole := auth.RoleFromCtx(r.Context())
	callerDeptID := auth.DepartmentIDFromCtx(r.Context())
	callerUserID := auth.UserIDFromCtx(r.Context())

	u, err := h.svc.UpdateUser(r.Context(), id, req, callerRole, callerDeptID, callerUserID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		case errors.Is(err, ErrForbidden):
			api.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		case errors.Is(err, ErrValidation):
			api.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		default:
			api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update user")
		}
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": u, "error": nil})
}

// DeactivateUser handles POST /api/v1/users/:id/deactivate.
func (h *Handler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	callerRole := auth.RoleFromCtx(r.Context())
	callerDeptID := auth.DepartmentIDFromCtx(r.Context())
	callerUserID := auth.UserIDFromCtx(r.Context())

	if err := h.svc.DeactivateUser(r.Context(), id, callerRole, callerDeptID, callerUserID, clientIP(r)); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		case errors.Is(err, ErrForbidden):
			api.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		default:
			api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to deactivate user")
		}
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{}, "error": nil})
}

// ResetPassword handles POST /api/v1/users/:id/reset-password.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	callerRole := auth.RoleFromCtx(r.Context())
	callerDeptID := auth.DepartmentIDFromCtx(r.Context())
	callerUserID := auth.UserIDFromCtx(r.Context())

	resp, err := h.svc.ResetPassword(r.Context(), id, callerRole, callerDeptID, callerUserID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		case errors.Is(err, ErrForbidden):
			api.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		default:
			api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to reset password")
		}
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": resp, "error": nil})
}

// ImportUsers handles POST /api/v1/users/import.
// Accepts multipart/form-data with a "file" field containing a CSV.
// Without ?commit=true, returns a preview only; with ?commit=true, writes valid rows.
func (h *Handler) ImportUsers(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "failed to parse multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "MISSING_FILE", "file field is required")
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	allRows, err := reader.ReadAll()
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "INVALID_CSV", "failed to parse CSV file")
		return
	}

	if len(allRows) < 2 {
		api.WriteError(w, http.StatusBadRequest, "INVALID_CSV", "CSV must contain a header row and at least one data row")
		return
	}

	dataRows := allRows[1:] // skip header
	if len(dataRows) > 500 {
		api.WriteError(w, http.StatusBadRequest, "TOO_MANY_ROWS", "CSV must not exceed 500 rows")
		return
	}

	var rows []CSVRow
	for i, record := range dataRows {
		if len(record) < 4 {
			msg := "row has fewer than 4 columns"
			rows = append(rows, CSVRow{
				RowNum: i + 2, // 1-based, +1 for header
			})
			_ = msg
			continue
		}
		rows = append(rows, CSVRow{
			RowNum:         i + 2,
			Email:          strings.TrimSpace(record[0]),
			FullName:       strings.TrimSpace(record[1]),
			DepartmentName: strings.TrimSpace(record[2]),
			RoleName:       strings.TrimSpace(record[3]),
		})
	}

	commit := r.URL.Query().Get("commit") == "true"
	callerRole := auth.RoleFromCtx(r.Context())
	callerDeptID := auth.DepartmentIDFromCtx(r.Context())
	callerUserID := auth.UserIDFromCtx(r.Context())

	preview, err := h.svc.ImportUsers(r.Context(), rows, commit, callerRole, callerDeptID, callerUserID, clientIP(r))
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "import failed")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": preview, "error": nil})
}

// clientIP extracts the caller's IP address, preferring X-Forwarded-For.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ip := strings.TrimSpace(strings.SplitN(xff, ",", 2)[0]); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// parseIntParam reads a query param as int, falling back to defaultVal on error or absence.
func parseIntParam(r *http.Request, name string, defaultVal int) int {
	v := r.URL.Query().Get(name)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}
