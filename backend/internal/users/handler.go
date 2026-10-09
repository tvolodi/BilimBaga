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
	// onUserChanged is called with the user id after a successful update, deactivate,
	// reset-password or unlock so the auth account-state cache drops its entry (ISS-248).
	onUserChanged func(userID string)
}

// SetUserChangedHook registers fn to be called with the affected user id after every
// successful account mutation. The router wires it to auth.AccountStateCache.Invalidate.
func (h *Handler) SetUserChangedHook(fn func(userID string)) { h.onUserChanged = fn }

func (h *Handler) userChanged(userID string) {
	if h.onUserChanged != nil {
		h.onUserChanged(userID)
	}
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

// UpdateMe handles PATCH /api/v1/users/me (FR-BB116). The body may contain only
// preferred_locale; any other key is rejected with 400 INVALID_BODY before anything is
// persisted (AC-3). A successful change of the value writes user.preferred_locale_updated.
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	var req UpdateMeRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid body: only preferred_locale is accepted")
		return
	}
	// Exactly one JSON object: anything after it is rejected, not silently ignored.
	var trailing json.RawMessage
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		api.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid body: only preferred_locale is accepted")
		return
	}
	// A bad locale value is a validation error: 422 VALIDATION_ERROR (api-conventions). Body shape
	// errors above stay 400.
	locale, err := parsePreferredLocale(req.PreferredLocale)
	if err != nil {
		api.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}

	callerUserID := auth.UserIDFromCtx(r.Context())
	change, err := h.svc.UpdateMyLocale(r.Context(), callerUserID, locale)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		case errors.Is(err, ErrValidation):
			api.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		default:
			api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update profile")
		}
		return
	}
	if !sameOptionalString(change.Previous, change.User.PreferredLocale) {
		h.writer.Write(r.Context(), r, "user.preferred_locale_updated", "user", &callerUserID, map[string]any{
			"old": nullableString(change.Previous),
			"new": nullableString(change.User.PreferredLocale),
		})
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": change.User, "error": nil})
}

// parsePreferredLocale turns the raw preferred_locale JSON value into a *string. An absent key
// is a validation error; an explicit null yields nil (clears the preference).
func parsePreferredLocale(raw json.RawMessage) (*string, error) {
	if len(raw) == 0 {
		return nil, errors.New("preferred_locale is required (a locale code or null)")
	}
	if string(raw) == "null" {
		return nil, nil
	}
	var code string
	if err := json.Unmarshal(raw, &code); err != nil {
		return nil, errors.New("preferred_locale must be a string or null")
	}
	return &code, nil
}

// sameOptionalString compares two nullable strings.
func sameOptionalString(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// nullableString returns the string value for audit metadata, or nil when unset.
func nullableString(p *string) any {
	if p == nil {
		return nil
	}
	return *p
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
	roles, err := h.svc.ListRoles(r.Context(), auth.RoleFromCtx(r.Context()))
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
	h.userChanged(id)
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
	h.userChanged(id)
	h.writer.Write(r.Context(), r, "user.deactivate", "user", &id, nil)
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{}, "error": nil})
}

// ReactivateUser handles POST /api/v1/users/:id/reactivate (FR-BB18 AC-13).
func (h *Handler) ReactivateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	callerRole := auth.RoleFromCtx(r.Context())
	callerDeptID := auth.DepartmentIDFromCtx(r.Context())
	callerUserID := auth.UserIDFromCtx(r.Context())

	if err := h.svc.ReactivateUser(r.Context(), id, callerRole, callerDeptID, callerUserID, clientIP(r)); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		case errors.Is(err, ErrForbidden):
			api.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		case errors.Is(err, ErrUserAlreadyActive):
			api.WriteError(w, http.StatusConflict, "USER_ALREADY_ACTIVE", "user is already active")
		default:
			api.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to reactivate user")
		}
		return
	}
	h.userChanged(id)
	h.writer.Write(r.Context(), r, "user.reactivate", "user", &id, nil)
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"message": "user reactivated"}, "error": nil})
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
	h.userChanged(id)
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
	h.userChanged(id)
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
