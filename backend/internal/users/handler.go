package users

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/bilimbaga/bilimbaga/internal/api"
	"github.com/bilimbaga/bilimbaga/internal/audit"
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/bilimbaga/bilimbaga/internal/upload"
	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for the users domain.
type Handler struct {
	svc    Service
	writer auditWriter
	// permsFor resolves a role name to its "resource:action" permissions (GET /users/me, AC-16).
	permsFor func(role string) []string
}

// WithPermissionsProvider sets the function used to populate permissions on GET /users/me.
func (h *Handler) WithPermissionsProvider(fn func(role string) []string) *Handler {
	h.permsFor = fn
	return h
}

// meResponse is the GET /users/me payload: the profile plus the caller's permissions.
type meResponse struct {
	*User
	Permissions []string `json:"permissions"`
}

// auditWriter is the subset of *audit.Writer the handler uses (allows a fake in tests).
type auditWriter interface {
	Write(ctx context.Context, r *http.Request, action, entityType string, entityID *string, metadata any)
}

// NewHandler creates a new Handler backed by the given Service and audit Writer.
func NewHandler(svc Service, writer *audit.Writer) *Handler {
	return &Handler{svc: svc, writer: writer}
}

// ListUsers handles GET /api/v1/users.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	callerRole := auth.RoleFromCtx(r.Context())
	callerDeptID := auth.DepartmentIDFromCtx(r.Context())

	f := ListFilters{
		Page:    parseIntParam(r, "page", 1),
		PerPage: parseIntParam(r, "per_page", 20),
	}
	// department_id and role_id are UUID columns; reject anything else with a 422
	// instead of letting Postgres fail with "invalid input syntax for type uuid" (500).
	deptID, ok := api.UUIDQuery(w, r, "department_id")
	if !ok {
		return
	}
	if deptID != "" {
		f.DepartmentID = &deptID
	}
	roleID, ok := api.UUIDQuery(w, r, "role_id")
	if !ok {
		return
	}
	if roleID != "" {
		f.RoleID = &roleID
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
	perms := []string{}
	if h.permsFor != nil {
		if p := h.permsFor(u.RoleName); p != nil {
			perms = p
		}
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": meResponse{User: u, Permissions: perms}, "error": nil})
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

// ListRoles handles GET /api/v1/users/roles.
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.ListRoles(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list roles")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": roles, "error": nil})
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
	h.writer.Write(r.Context(), r, "user.create", "user", &resp.User.ID, map[string]any{"email": resp.User.Email})
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
	h.writer.Write(r.Context(), r, "user.update", "user", &id, nil)
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
	h.writer.Write(r.Context(), r, "user.deactivate", "user", &id, nil)
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
	h.writer.Write(r.Context(), r, "user.password_reset", "user", &id, nil)
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": resp, "error": nil})
}

// UnlockUser handles POST /api/v1/users/:id/unlock (FR-BB115).
func (h *Handler) UnlockUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	callerRole := auth.RoleFromCtx(r.Context())
	callerDeptID := auth.DepartmentIDFromCtx(r.Context())
	callerUserID := auth.UserIDFromCtx(r.Context())

	user, err := h.svc.UnlockUser(r.Context(), id, callerRole, callerDeptID, callerUserID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		case errors.Is(err, ErrForbidden):
			api.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		default:
			api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to unlock user")
		}
		return
	}
	h.writer.Write(r.Context(), r, "users.unlock", "user", &id, nil)
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": user, "error": nil})
}

// ImportUsers handles POST /api/v1/users/import.
// Accepts multipart/form-data with a "file" field containing a CSV.
// Without ?commit=true, returns a preview only; with ?commit=true, writes valid rows.
func (h *Handler) ImportUsers(w http.ResponseWriter, r *http.Request) {
	if err := upload.ParseImportMultipart(w, r); err != nil {
		if errors.Is(err, upload.ErrFileTooLarge) {
			api.WriteError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "CSV file must not exceed 10 MB")
			return
		}
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "failed to parse multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "MISSING_FILE", "file field is required")
		return
	}
	defer file.Close()

	// AC-3 (FR-BB64): read all bytes for magic-byte validation before parsing.
	rawBytes, err := io.ReadAll(file)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "failed to read uploaded file")
		return
	}
	if err := upload.ValidateCSVFile(rawBytes); err != nil {
		switch err {
		case upload.ErrFileTooLarge:
			api.WriteError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "CSV file must not exceed 10 MB")
		default:
			api.WriteError(w, http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE", "uploaded file must be a plain-text CSV")
		}
		return
	}

	reader := csv.NewReader(strings.NewReader(string(rawBytes)))
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

// RemindEmployee handles POST /api/v1/admin/users/:userId/remind (FR-BB56 AC-7 stub).
// Returns HTTP 200 with empty data until the notification system is built.
func (h *Handler) RemindEmployee(w http.ResponseWriter, r *http.Request) {
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": nil, "error": nil})
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
