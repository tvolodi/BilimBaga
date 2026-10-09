package users

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── mock service ─────────────────────────────────────────────────────────────

type mockUserService struct {
	listFn       func(ctx context.Context, callerRole, callerDeptID string, f ListFilters) (*ListResult, error)
	getUserFn    func(ctx context.Context, id, callerRole, callerUserID, callerDeptID string) (*User, error)
	getMeFn      func(ctx context.Context, userID string) (*User, error)
	createFn     func(ctx context.Context, req CreateRequest, callerRole, callerDeptID, callerUserID, ip string) (*CreateResponse, error)
	updateFn     func(ctx context.Context, id string, req UpdateRequest, callerRole, callerDeptID, callerUserID, ip string) (*User, error)
	deactivateFn func(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) error
	reactivateFn func(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) error
	resetPwdFn   func(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) (*ResetPasswordResponse, error)
	importFn     func(ctx context.Context, rows []CSVRow, commit bool, callerRole, callerDeptID, callerUserID, ip string) (*ImportPreview, error)
	listRolesFn  func(ctx context.Context, callerRole string) ([]RoleRow, error)
	unlockFn     func(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) (*User, error)
	remindFn     func(ctx context.Context, actorID, userID, examID string) (*RemindResult, error)
	updateMeFn   func(ctx context.Context, userID string, locale *string) (*LocaleUpdate, error)
}

func (m *mockUserService) UpdateMyLocale(ctx context.Context, userID string, locale *string) (*LocaleUpdate, error) {
	return m.updateMeFn(ctx, userID, locale)
}

func (m *mockUserService) ListUsers(ctx context.Context, callerRole, callerDeptID string, f ListFilters) (*ListResult, error) {
	return m.listFn(ctx, callerRole, callerDeptID, f)
}
func (m *mockUserService) GetUser(ctx context.Context, id, callerRole, callerUserID, callerDeptID string) (*User, error) {
	return m.getUserFn(ctx, id, callerRole, callerUserID, callerDeptID)
}
func (m *mockUserService) GetMe(ctx context.Context, userID string) (*User, error) {
	return m.getMeFn(ctx, userID)
}
func (m *mockUserService) CreateUser(ctx context.Context, req CreateRequest, callerRole, callerDeptID, callerUserID, ip string) (*CreateResponse, error) {
	return m.createFn(ctx, req, callerRole, callerDeptID, callerUserID, ip)
}
func (m *mockUserService) UpdateUser(ctx context.Context, id string, req UpdateRequest, callerRole, callerDeptID, callerUserID, ip string) (*User, error) {
	return m.updateFn(ctx, id, req, callerRole, callerDeptID, callerUserID, ip)
}
func (m *mockUserService) DeactivateUser(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) error {
	return m.deactivateFn(ctx, id, callerRole, callerDeptID, callerUserID, ip)
}
func (m *mockUserService) ReactivateUser(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) error {
	return m.reactivateFn(ctx, id, callerRole, callerDeptID, callerUserID, ip)
}
func (m *mockUserService) ResetPassword(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) (*ResetPasswordResponse, error) {
	return m.resetPwdFn(ctx, id, callerRole, callerDeptID, callerUserID, ip)
}
func (m *mockUserService) ImportUsers(ctx context.Context, rows []CSVRow, commit bool, callerRole, callerDeptID, callerUserID, ip string) (*ImportPreview, error) {
	return m.importFn(ctx, rows, commit, callerRole, callerDeptID, callerUserID, ip)
}
func (m *mockUserService) UnlockUser(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) (*User, error) {
	return m.unlockFn(ctx, id, callerRole, callerDeptID, callerUserID, ip)
}
func (m *mockUserService) ListRoles(ctx context.Context, callerRole string) ([]RoleRow, error) {
	if m.listRolesFn != nil {
		return m.listRolesFn(ctx, callerRole)
	}
	return []RoleRow{}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

type handlerEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Error *handlerErrBody `json:"error"`
}

type handlerErrBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func decodeHandlerEnvelope(t *testing.T, w *httptest.ResponseRecorder) (json.RawMessage, *handlerErrBody) {
	t.Helper()
	var env handlerEnvelope
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	return env.Data, env.Error
}

func withAuthCtx(r *http.Request, userID, role, deptID string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, ctxkeys.CtxUserID, userID)
	ctx = context.WithValue(ctx, ctxkeys.CtxRole, role)
	ctx = context.WithValue(ctx, ctxkeys.CtxDepartmentID, deptID)
	return r.WithContext(ctx)
}

func withChiID(r *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func sampleUser(id string) *User {
	deptID := "dept-1"
	deptName := "Engineering"
	return &User{
		ID:             id,
		Email:          id + "@example.com",
		FullName:       "Test " + id,
		DepartmentID:   &deptID,
		DepartmentName: &deptName,
		RoleID:         "role-emp",
		RoleName:       "employee",
		Status:         "active",
		CreatedAt:      time.Now(),
	}
}

// ── ListUsers ─────────────────────────────────────────────────────────────────

func TestHandlerListUsers_Returns200(t *testing.T) {
	svc := &mockUserService{
		listFn: func(_ context.Context, _, _ string, _ ListFilters) (*ListResult, error) {
			return &ListResult{
				Items: []User{*sampleUser("u1")},
				Meta:  Meta{Total: 1, Page: 1, PerPage: 20},
			}, nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req = withAuthCtx(req, "caller-1", "super_admin", "")
	w := httptest.NewRecorder()
	h.ListUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data, apiErr := decodeHandlerEnvelope(t, w)
	assert.Nil(t, apiErr)
	assert.NotEmpty(t, data)
}

func TestHandlerListUsers_ServiceError_Returns500(t *testing.T) {
	svc := &mockUserService{
		listFn: func(_ context.Context, _, _ string, _ ListFilters) (*ListResult, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req = withAuthCtx(req, "caller-1", "super_admin", "")
	w := httptest.NewRecorder()
	h.ListUsers(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "INTERNAL_ERROR", apiErr.Code)
}

// ── GetMe ─────────────────────────────────────────────────────────────────────

func TestHandlerGetMe_Returns200(t *testing.T) {
	svc := &mockUserService{
		getMeFn: func(_ context.Context, _ string) (*User, error) {
			return sampleUser("u1"), nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req = withAuthCtx(req, "u1", "employee", "dept-1")
	w := httptest.NewRecorder()
	h.GetMe(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data, apiErr := decodeHandlerEnvelope(t, w)
	assert.Nil(t, apiErr)
	assert.NotEmpty(t, data)
}

func TestHandlerGetMe_NotFound_Returns404(t *testing.T) {
	svc := &mockUserService{
		getMeFn: func(_ context.Context, _ string) (*User, error) {
			return nil, ErrNotFound
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req = withAuthCtx(req, "missing", "employee", "")
	w := httptest.NewRecorder()
	h.GetMe(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "NOT_FOUND", apiErr.Code)
}

// ── GetUser ───────────────────────────────────────────────────────────────────

func TestHandlerGetUser_Returns200(t *testing.T) {
	svc := &mockUserService{
		getUserFn: func(_ context.Context, id, _, _, _ string) (*User, error) {
			return sampleUser(id), nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/u1", nil)
	req = withAuthCtx(req, "caller", "super_admin", "")
	req = withChiID(req, "u1")
	w := httptest.NewRecorder()
	h.GetUser(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandlerGetUser_NotFound_Returns404(t *testing.T) {
	svc := &mockUserService{
		getUserFn: func(_ context.Context, _, _, _, _ string) (*User, error) {
			return nil, ErrNotFound
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/missing", nil)
	req = withAuthCtx(req, "caller", "super_admin", "")
	req = withChiID(req, "missing")
	w := httptest.NewRecorder()
	h.GetUser(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "NOT_FOUND", apiErr.Code)
}

func TestHandlerGetUser_Forbidden_Returns403(t *testing.T) {
	svc := &mockUserService{
		getUserFn: func(_ context.Context, _, _, _, _ string) (*User, error) {
			return nil, ErrForbidden
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/u2", nil)
	req = withAuthCtx(req, "caller", "department_admin", "dept-1")
	req = withChiID(req, "u2")
	w := httptest.NewRecorder()
	h.GetUser(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "FORBIDDEN", apiErr.Code)
}

// ── CreateUser ────────────────────────────────────────────────────────────────

func TestHandlerCreateUser_Returns201(t *testing.T) {
	u := sampleUser("new-1")
	svc := &mockUserService{
		createFn: func(_ context.Context, _ CreateRequest, _, _, _, _ string) (*CreateResponse, error) {
			return &CreateResponse{User: *u, TemporaryPassword: "Abc1!abcde"}, nil
		},
	}
	h := NewHandler(svc, nil)

	body := `{"email":"new@example.com","full_name":"New User","role_id":"role-emp"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req = withAuthCtx(req, "caller", "super_admin", "")
	w := httptest.NewRecorder()
	h.CreateUser(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	data, apiErr := decodeHandlerEnvelope(t, w)
	assert.Nil(t, apiErr)
	assert.NotEmpty(t, data)
}

func TestHandlerCreateUser_InvalidJSON_Returns400(t *testing.T) {
	h := NewHandler(&mockUserService{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader("not-json"))
	req = withAuthCtx(req, "caller", "super_admin", "")
	w := httptest.NewRecorder()
	h.CreateUser(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "INVALID_BODY", apiErr.Code)
}

func TestHandlerCreateUser_ValidationError_Returns422(t *testing.T) {
	svc := &mockUserService{
		createFn: func(_ context.Context, _ CreateRequest, _, _, _, _ string) (*CreateResponse, error) {
			return nil, ErrValidation
		},
	}
	h := NewHandler(svc, nil)

	body := `{"email":"bad","full_name":"X","role_id":"r"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req = withAuthCtx(req, "caller", "super_admin", "")
	w := httptest.NewRecorder()
	h.CreateUser(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "VALIDATION_ERROR", apiErr.Code)
}

func TestHandlerCreateUser_DuplicateEmail_Returns409(t *testing.T) {
	svc := &mockUserService{
		createFn: func(_ context.Context, _ CreateRequest, _, _, _, _ string) (*CreateResponse, error) {
			return nil, ErrDuplicateEmail
		},
	}
	h := NewHandler(svc, nil)

	body := `{"email":"dup@example.com","full_name":"Dup","role_id":"r"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body))
	req = withAuthCtx(req, "caller", "super_admin", "")
	w := httptest.NewRecorder()
	h.CreateUser(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "DUPLICATE_EMAIL", apiErr.Code)
}

// ── UpdateUser ────────────────────────────────────────────────────────────────

func TestHandlerUpdateUser_Returns200(t *testing.T) {
	svc := &mockUserService{
		updateFn: func(_ context.Context, id string, _ UpdateRequest, _, _, _, _ string) (*User, error) {
			return sampleUser(id), nil
		},
	}
	h := NewHandler(svc, nil)

	body := `{"full_name":"Updated Name","role_id":"role-emp"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/u1", strings.NewReader(body))
	req = withAuthCtx(req, "caller", "super_admin", "")
	req = withChiID(req, "u1")
	w := httptest.NewRecorder()
	h.UpdateUser(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandlerUpdateUser_NotFound_Returns404(t *testing.T) {
	svc := &mockUserService{
		updateFn: func(_ context.Context, _ string, _ UpdateRequest, _, _, _, _ string) (*User, error) {
			return nil, ErrNotFound
		},
	}
	h := NewHandler(svc, nil)

	body := `{"full_name":"X","role_id":"r"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/missing", strings.NewReader(body))
	req = withAuthCtx(req, "caller", "super_admin", "")
	req = withChiID(req, "missing")
	w := httptest.NewRecorder()
	h.UpdateUser(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── DeactivateUser ────────────────────────────────────────────────────────────

func TestHandlerDeactivateUser_Returns200(t *testing.T) {
	svc := &mockUserService{
		deactivateFn: func(_ context.Context, _, _, _, _, _ string) error { return nil },
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/u1/deactivate", nil)
	req = withAuthCtx(req, "caller", "super_admin", "")
	req = withChiID(req, "u1")
	w := httptest.NewRecorder()
	h.DeactivateUser(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandlerDeactivateUser_NotFound_Returns404(t *testing.T) {
	svc := &mockUserService{
		deactivateFn: func(_ context.Context, _, _, _, _, _ string) error { return ErrNotFound },
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/missing/deactivate", nil)
	req = withAuthCtx(req, "caller", "super_admin", "")
	req = withChiID(req, "missing")
	w := httptest.NewRecorder()
	h.DeactivateUser(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandlerDeactivateUser_Forbidden_Returns403(t *testing.T) {
	svc := &mockUserService{
		deactivateFn: func(_ context.Context, _, _, _, _, _ string) error { return ErrForbidden },
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/u2/deactivate", nil)
	req = withAuthCtx(req, "caller", "department_admin", "dept-1")
	req = withChiID(req, "u2")
	w := httptest.NewRecorder()
	h.DeactivateUser(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ── ResetPassword ─────────────────────────────────────────────────────────────

func TestHandlerResetPassword_Returns200(t *testing.T) {
	svc := &mockUserService{
		resetPwdFn: func(_ context.Context, _, _, _, _, _ string) (*ResetPasswordResponse, error) {
			return &ResetPasswordResponse{TemporaryPassword: "Abc1!abcde"}, nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/u1/reset-password", nil)
	req = withAuthCtx(req, "caller", "super_admin", "")
	req = withChiID(req, "u1")
	w := httptest.NewRecorder()
	h.ResetPassword(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data, apiErr := decodeHandlerEnvelope(t, w)
	assert.Nil(t, apiErr)
	assert.NotEmpty(t, data)
}

func TestHandlerResetPassword_NotFound_Returns404(t *testing.T) {
	svc := &mockUserService{
		resetPwdFn: func(_ context.Context, _, _, _, _, _ string) (*ResetPasswordResponse, error) {
			return nil, ErrNotFound
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/missing/reset-password", nil)
	req = withAuthCtx(req, "caller", "super_admin", "")
	req = withChiID(req, "missing")
	w := httptest.NewRecorder()
	h.ResetPassword(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── ImportUsers ───────────────────────────────────────────────────────────────

func buildCSVMultipart(t *testing.T, csvContent string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "users.csv")
	require.NoError(t, err)
	fw.Write([]byte(csvContent))
	mw.Close()
	return &buf, mw.FormDataContentType()
}

func TestHandlerImportUsers_Preview_Returns200(t *testing.T) {
	svc := &mockUserService{
		importFn: func(_ context.Context, _ []CSVRow, _ bool, _, _, _, _ string) (*ImportPreview, error) {
			return &ImportPreview{
				Valid:  []ImportRowResult{{RowNum: 2, Email: "a@example.com", FullName: "Alice"}},
				Errors: []ImportRowResult{},
			}, nil
		},
	}
	h := NewHandler(svc, nil)

	csvData := "email,full_name,department,role\na@example.com,Alice,Engineering,employee\n"
	body, ct := buildCSVMultipart(t, csvData)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/import", body)
	req.Header.Set("Content-Type", ct)
	req = withAuthCtx(req, "caller", "super_admin", "")
	w := httptest.NewRecorder()
	h.ImportUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data, apiErr := decodeHandlerEnvelope(t, w)
	assert.Nil(t, apiErr)
	assert.NotEmpty(t, data)
}

func TestHandlerImportUsers_NoFile_Returns400(t *testing.T) {
	h := NewHandler(&mockUserService{}, nil)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/import", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req = withAuthCtx(req, "caller", "super_admin", "")
	w := httptest.NewRecorder()
	h.ImportUsers(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "MISSING_FILE", apiErr.Code)
}

func TestHandlerImportUsers_TooManyRows_Returns400(t *testing.T) {
	h := NewHandler(&mockUserService{}, nil)

	var sb strings.Builder
	sb.WriteString("email,full_name,department,role\n")
	for i := 0; i < 501; i++ {
		sb.WriteString("a@example.com,Alice,Engineering,employee\n")
	}
	body, ct := buildCSVMultipart(t, sb.String())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/import", body)
	req.Header.Set("Content-Type", ct)
	req = withAuthCtx(req, "caller", "super_admin", "")
	w := httptest.NewRecorder()
	h.ImportUsers(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "TOO_MANY_ROWS", apiErr.Code)
}

// ── UnlockUser (FR-BB115 AC-5, AC-6) ──────────────────────────────────────────

type fakeAudit struct {
	actions []string
	ids     []*string
	metas   []any
}

func (f *fakeAudit) Write(_ context.Context, _ *http.Request, action, _ string, entityID *string, metadata any) {
	f.actions = append(f.actions, action)
	f.ids = append(f.ids, entityID)
	f.metas = append(f.metas, metadata)
}

func TestHandlerUnlockUser_Returns200AndAudits(t *testing.T) {
	aw := &fakeAudit{}
	svc := &mockUserService{unlockFn: func(_ context.Context, id, role, _, caller, _ string) (*User, error) {
		assert.Equal(t, "u1", id)
		assert.Equal(t, "super_admin", role)
		assert.Equal(t, "admin-1", caller)
		u := sampleUser(id)
		u.IsLocked = false
		return u, nil
	}}
	h := &Handler{svc: svc, writer: aw}

	req := withChiID(withAuthCtx(httptest.NewRequest(http.MethodPost, "/api/v1/users/u1/unlock", nil), "admin-1", "super_admin", ""), "u1")
	w := httptest.NewRecorder()
	h.UnlockUser(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data, apiErr := decodeHandlerEnvelope(t, w)
	assert.Nil(t, apiErr)
	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, "u1", got["id"])
	assert.Equal(t, false, got["is_locked"])
	require.Equal(t, []string{"users.unlock"}, aw.actions)
	require.NotNil(t, aw.ids[0])
	assert.Equal(t, "u1", *aw.ids[0])
}

func TestHandlerUnlockUser_NotFound_Returns404NoAudit(t *testing.T) {
	aw := &fakeAudit{}
	svc := &mockUserService{unlockFn: func(context.Context, string, string, string, string, string) (*User, error) {
		return nil, ErrNotFound
	}}
	h := &Handler{svc: svc, writer: aw}

	req := withChiID(withAuthCtx(httptest.NewRequest(http.MethodPost, "/x", nil), "admin-1", "super_admin", ""), "nope")
	w := httptest.NewRecorder()
	h.UnlockUser(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "NOT_FOUND", apiErr.Code)
	assert.Empty(t, aw.actions)
}

func TestHandlerUnlockUser_Forbidden_Returns403(t *testing.T) {
	svc := &mockUserService{unlockFn: func(context.Context, string, string, string, string, string) (*User, error) {
		return nil, ErrForbidden
	}}
	h := &Handler{svc: svc, writer: &fakeAudit{}}

	req := withChiID(withAuthCtx(httptest.NewRequest(http.MethodPost, "/x", nil), "da-1", "department_admin", "dept-1"), "u1")
	w := httptest.NewRecorder()
	h.UnlockUser(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandlerUnlockUser_ServiceError_Returns500(t *testing.T) {
	svc := &mockUserService{unlockFn: func(context.Context, string, string, string, string, string) (*User, error) {
		return nil, errors.New("db down")
	}}
	h := &Handler{svc: svc, writer: &fakeAudit{}}

	req := withChiID(withAuthCtx(httptest.NewRequest(http.MethodPost, "/x", nil), "admin-1", "super_admin", ""), "u1")
	w := httptest.NewRecorder()
	h.UnlockUser(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ISS-133: the role filter must pass a role UUID through to the service and
// reject non-UUID values (the UI used to send role names) with 422, not 500.
func TestHandlerListUsers_RoleIDFilter_PassedToService(t *testing.T) {
	const roleID = "3f2b8c1e-9d4a-4b6e-8a1f-0c7d5e9a1b22"
	var got ListFilters
	svc := &mockUserService{
		listFn: func(_ context.Context, _, _ string, f ListFilters) (*ListResult, error) {
			got = f
			return &ListResult{Items: []User{}, Meta: Meta{Page: 1, PerPage: 20}}, nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?role_id="+roleID, nil)
	req = withAuthCtx(req, "caller-1", "super_admin", "")
	w := httptest.NewRecorder()
	h.ListUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, got.RoleID)
	assert.Equal(t, roleID, *got.RoleID)
}

func TestHandlerListUsers_RoleIDFilter_NonUUID_Returns422(t *testing.T) {
	called := false
	svc := &mockUserService{
		listFn: func(_ context.Context, _, _ string, _ ListFilters) (*ListResult, error) {
			called = true
			return &ListResult{}, nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?role_id=super_admin", nil)
	req = withAuthCtx(req, "caller-1", "super_admin", "")
	w := httptest.NewRecorder()
	h.ListUsers(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "VALIDATION_ERROR", apiErr.Code)
	assert.False(t, called, "service must not be called for an invalid role_id")
}

// ISS-141: department_id is a UUID column too; malformed values must 422.
func TestHandlerListUsers_DepartmentIDFilter_PassedToService(t *testing.T) {
	const deptID = "3f2b8c1e-9d4a-4b6e-8a1f-0c7d5e9a1b22"
	var got ListFilters
	svc := &mockUserService{
		listFn: func(_ context.Context, _, _ string, f ListFilters) (*ListResult, error) {
			got = f
			return &ListResult{Items: []User{}, Meta: Meta{Page: 1, PerPage: 20}}, nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?department_id="+deptID, nil)
	req = withAuthCtx(req, "caller-1", "super_admin", "")
	w := httptest.NewRecorder()
	h.ListUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, got.DepartmentID)
	assert.Equal(t, deptID, *got.DepartmentID)
}

func TestHandlerListUsers_DepartmentIDFilter_NonUUID_Returns422(t *testing.T) {
	called := false
	svc := &mockUserService{
		listFn: func(_ context.Context, _, _ string, _ ListFilters) (*ListResult, error) {
			called = true
			return &ListResult{}, nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?department_id=engineering", nil)
	req = withAuthCtx(req, "caller-1", "super_admin", "")
	w := httptest.NewRecorder()
	h.ListUsers(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "VALIDATION_ERROR", apiErr.Code)
	assert.False(t, called)
}

func TestHandlerImportUsers_BodyOverCap_Returns413(t *testing.T) {
	h := NewHandler(&mockUserService{}, nil)

	body, ct := buildCSVMultipart(t, strings.Repeat("a", 13<<20))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/import", body)
	req.Header.Set("Content-Type", ct)
	req = withAuthCtx(req, "caller", "super_admin", "")
	w := httptest.NewRecorder()
	h.ImportUsers(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "FILE_TOO_LARGE", apiErr.Code)
}

// ── ISS-248: account-state cache invalidation hook ───────────────────────────

func TestHandlerMutations_InvokeUserChangedHook(t *testing.T) {
	svc := &mockUserService{
		updateFn: func(_ context.Context, id string, _ UpdateRequest, _, _, _, _ string) (*User, error) {
			return sampleUser(id), nil
		},
		deactivateFn: func(_ context.Context, _, _, _, _, _ string) error { return nil },
		resetPwdFn: func(_ context.Context, _, _, _, _, _ string) (*ResetPasswordResponse, error) {
			return &ResetPasswordResponse{TemporaryPassword: "x"}, nil
		},
		unlockFn: func(_ context.Context, id, _, _, _, _ string) (*User, error) { return sampleUser(id), nil },
	}
	cases := []struct {
		name string
		call func(h *Handler, w http.ResponseWriter, r *http.Request)
		body string
	}{
		{"update", (*Handler).UpdateUser, `{"full_name":"N","role_id":"role-emp"}`},
		{"deactivate", (*Handler).DeactivateUser, ""},
		{"reset-password", (*Handler).ResetPassword, ""},
		{"unlock", (*Handler).UnlockUser, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got []string
			h := NewHandler(svc, nil)
			h.SetUserChangedHook(func(id string) { got = append(got, id) })
			req := httptest.NewRequest(http.MethodPost, "/api/v1/users/u1", strings.NewReader(c.body))
			req = withChiID(withAuthCtx(req, "caller", "super_admin", ""), "u1")
			w := httptest.NewRecorder()
			c.call(h, w, req)
			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, []string{"u1"}, got)
		})
	}
}

func TestHandlerMutations_FailureDoesNotInvokeHook(t *testing.T) {
	svc := &mockUserService{
		deactivateFn: func(_ context.Context, _, _, _, _, _ string) error { return ErrForbidden },
		unlockFn:     func(_ context.Context, _, _, _, _, _ string) (*User, error) { return nil, ErrNotFound },
	}
	called := false
	h := NewHandler(svc, nil)
	h.SetUserChangedHook(func(string) { called = true })
	for _, call := range []func(http.ResponseWriter, *http.Request){h.DeactivateUser, h.UnlockUser} {
		req := withChiID(withAuthCtx(httptest.NewRequest(http.MethodPost, "/x", nil), "caller", "super_admin", ""), "u1")
		call(httptest.NewRecorder(), req)
	}
	assert.False(t, called)
}

func TestHandlerMutations_NoHookIsSafe(t *testing.T) {
	svc := &mockUserService{deactivateFn: func(_ context.Context, _, _, _, _, _ string) error { return nil }}
	h := NewHandler(svc, nil)
	req := withChiID(withAuthCtx(httptest.NewRequest(http.MethodPost, "/x", nil), "caller", "super_admin", ""), "u1")
	w := httptest.NewRecorder()
	h.DeactivateUser(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---- FR-BB116: PATCH /users/me and preferred_locale in read payloads ---------

// patchMe sends a raw PATCH /api/v1/users/me body as the given caller.
func patchMe(h *Handler, callerID, body string) *httptest.ResponseRecorder {
	req := withAuthCtx(httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", strings.NewReader(body)), callerID, "employee", "dept-1")
	w := httptest.NewRecorder()
	h.UpdateMe(w, req)
	return w
}

// localeUpdateSvc returns a mock whose UpdateMyLocale reports the given previous and new value
// and records the arguments it was called with.
func localeUpdateSvc(called *bool, gotID *string, gotLocale **string, previous, next *string) *mockUserService {
	return &mockUserService{updateMeFn: func(_ context.Context, userID string, locale *string) (*LocaleUpdate, error) {
		*called = true
		*gotID = userID
		*gotLocale = locale
		u := sampleUser(userID)
		u.PreferredLocale = next
		return &LocaleUpdate{User: u, Previous: previous}, nil
	}}
}

// AC-2 + AC-4: a valid code is persisted via the service, returned as 200 data, and audited with
// the old and new value.
func TestHandlerUpdateMe_ValidLocale_Returns200AndAudits(t *testing.T) {
	var called bool
	var gotID string
	var gotLocale *string
	aw := &fakeAudit{}
	h := &Handler{svc: localeUpdateSvc(&called, &gotID, &gotLocale, nil, strPtr("ru")), writer: aw}

	w := patchMe(h, "u-caller", `{"preferred_locale":"ru"}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	data, apiErr := decodeHandlerEnvelope(t, w)
	assert.Nil(t, apiErr)
	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, "ru", got["preferred_locale"])

	require.True(t, called)
	assert.Equal(t, "u-caller", gotID, "service must receive the caller's own id")
	require.NotNil(t, gotLocale)
	assert.Equal(t, "ru", *gotLocale)

	require.Equal(t, []string{"user.preferred_locale_updated"}, aw.actions)
	require.NotNil(t, aw.ids[0])
	assert.Equal(t, "u-caller", *aw.ids[0], "audit entity is the caller")
	assert.Equal(t, map[string]any{"old": nil, "new": "ru"}, aw.metas[0])
}

// AC-2: an explicit null clears the preference; the response carries null and the audit records
// the cleared value as old.
func TestHandlerUpdateMe_NullClears_Returns200AndAuditsOldValue(t *testing.T) {
	var called bool
	var gotID string
	var gotLocale *string
	aw := &fakeAudit{}
	h := &Handler{svc: localeUpdateSvc(&called, &gotID, &gotLocale, strPtr("kk"), nil), writer: aw}

	w := patchMe(h, "u-caller", `{"preferred_locale":null}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Nil(t, gotLocale, "null must reach the service as nil")
	data, _ := decodeHandlerEnvelope(t, w)
	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Contains(t, got, "preferred_locale")
	assert.Nil(t, got["preferred_locale"])
	assert.Equal(t, map[string]any{"old": "kk", "new": nil}, aw.metas[0])
}

// AC-4: the audit entry is written only when the stored value actually changes.
func TestHandlerUpdateMe_UnchangedValue_NoAudit(t *testing.T) {
	var called bool
	var gotID string
	var gotLocale *string
	aw := &fakeAudit{}
	h := &Handler{svc: localeUpdateSvc(&called, &gotID, &gotLocale, strPtr("ru"), strPtr("ru")), writer: aw}

	w := patchMe(h, "u-caller", `{"preferred_locale":"ru"}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Empty(t, aw.actions)
}

// AC-2: a code not in the tenant's available_locales is 422 VALIDATION_ERROR and not audited.
func TestHandlerUpdateMe_UnavailableLocale_Returns422(t *testing.T) {
	aw := &fakeAudit{}
	svc := &mockUserService{updateMeFn: func(context.Context, string, *string) (*LocaleUpdate, error) {
		return nil, fmt.Errorf("%w: preferred_locale must be one of the tenant available locales", ErrValidation)
	}}
	h := &Handler{svc: svc, writer: aw}

	w := patchMe(h, "u-caller", `{"preferred_locale":"de"}`)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "VALIDATION_ERROR", apiErr.Code)
	assert.Empty(t, aw.actions)
}

// AC-3: any key other than preferred_locale is rejected with 400 and the service is never reached,
// so nothing is persisted.
func TestHandlerUpdateMe_ExtraField_Returns400AndNothingPersisted(t *testing.T) {
	for _, body := range []string{
		`{"preferred_locale":"ru","role_id":"role-sa"}`,
		`{"preferred_locale":"ru","department_id":"dept-2"}`,
		`{"preferred_locale":"ru","email":"x@example.com"}`,
		`{"preferred_locale":"ru","status":"active"}`,
		`{"role_id":"role-sa"}`,
	} {
		t.Run(body, func(t *testing.T) {
			var called bool
			svc := &mockUserService{updateMeFn: func(context.Context, string, *string) (*LocaleUpdate, error) {
				called = true
				return nil, nil
			}}
			aw := &fakeAudit{}
			h := &Handler{svc: svc, writer: aw}

			w := patchMe(h, "u-caller", body)

			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			_, apiErr := decodeHandlerEnvelope(t, w)
			require.NotNil(t, apiErr)
			assert.Equal(t, "INVALID_BODY", apiErr.Code)
			assert.False(t, called, "service must not be reached")
			assert.Empty(t, aw.actions)
		})
	}
}

// AC-2: preferred_locale is required; an absent key is not treated as a null that clears it.
// The locale value is invalid, so the status is 422 VALIDATION_ERROR.
func TestHandlerUpdateMe_MissingKey_Returns422(t *testing.T) {
	var called bool
	svc := &mockUserService{updateMeFn: func(context.Context, string, *string) (*LocaleUpdate, error) {
		called = true
		return nil, nil
	}}
	h := &Handler{svc: svc, writer: &fakeAudit{}}

	w := patchMe(h, "u-caller", `{}`)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "VALIDATION_ERROR", apiErr.Code)
	assert.False(t, called)
}

func TestHandlerUpdateMe_TrailingData_Returns400(t *testing.T) {
	svc := &mockUserService{updateMeFn: func(context.Context, string, *string) (*LocaleUpdate, error) {
		t.Fatal("service must not be reached")
		return nil, nil
	}}
	h := &Handler{svc: svc, writer: &fakeAudit{}}

	for _, body := range []string{`{"preferred_locale":"ru"} {}`, `{"preferred_locale":"ru"}garbage`} {
		w := patchMe(h, "u-caller", body)
		assert.Equal(t, http.StatusBadRequest, w.Code, body)
		_, apiErr := decodeHandlerEnvelope(t, w)
		require.NotNil(t, apiErr, body)
		assert.Equal(t, "INVALID_BODY", apiErr.Code, body)
	}
}

// AC-2: a non-string, non-null preferred_locale is a locale validation error: 422 VALIDATION_ERROR,
// and the service is never reached.
func TestHandlerUpdateMe_NonStringLocale_Returns422(t *testing.T) {
	svc := &mockUserService{updateMeFn: func(context.Context, string, *string) (*LocaleUpdate, error) {
		t.Fatal("service must not be reached")
		return nil, nil
	}}
	h := &Handler{svc: svc, writer: &fakeAudit{}}

	for _, body := range []string{`{"preferred_locale":5}`, `{"preferred_locale":["ru"]}`} {
		w := patchMe(h, "u-caller", body)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code, body)
		_, apiErr := decodeHandlerEnvelope(t, w)
		require.NotNil(t, apiErr, body)
		assert.Equal(t, "VALIDATION_ERROR", apiErr.Code, body)
	}
}

func TestHandlerUpdateMe_MalformedJSON_Returns400(t *testing.T) {
	svc := &mockUserService{updateMeFn: func(context.Context, string, *string) (*LocaleUpdate, error) {
		t.Fatal("service must not be reached")
		return nil, nil
	}}
	h := &Handler{svc: svc, writer: &fakeAudit{}}

	w := patchMe(h, "u-caller", `{"preferred_locale":`)

	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "INVALID_BODY", apiErr.Code)
}

func TestHandlerUpdateMe_UserNotFound_Returns404NoAudit(t *testing.T) {
	aw := &fakeAudit{}
	svc := &mockUserService{updateMeFn: func(context.Context, string, *string) (*LocaleUpdate, error) {
		return nil, ErrNotFound
	}}
	h := &Handler{svc: svc, writer: aw}

	w := patchMe(h, "ghost", `{"preferred_locale":"ru"}`)

	assert.Equal(t, http.StatusNotFound, w.Code)
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "NOT_FOUND", apiErr.Code)
	assert.Empty(t, aw.actions)
}

func TestHandlerUpdateMe_ServiceError_Returns500NoAudit(t *testing.T) {
	aw := &fakeAudit{}
	svc := &mockUserService{updateMeFn: func(context.Context, string, *string) (*LocaleUpdate, error) {
		return nil, errors.New("db down")
	}}
	h := &Handler{svc: svc, writer: aw}

	w := patchMe(h, "u-caller", `{"preferred_locale":"ru"}`)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	_, apiErr := decodeHandlerEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "INTERNAL_ERROR", apiErr.Code)
	assert.Empty(t, aw.actions)
}

// AC-1: preferred_locale is part of the GET /users/me, GET /users/{id} and list payloads, and is
// null when unset. Existing fields are still present.
func TestHandlerReads_IncludePreferredLocale(t *testing.T) {
	withLocale := sampleUser("u1")
	withLocale.PreferredLocale = strPtr("ru")
	noLocale := sampleUser("u2")

	svc := &mockUserService{
		getMeFn: func(context.Context, string) (*User, error) { return withLocale, nil },
		getUserFn: func(context.Context, string, string, string, string) (*User, error) {
			return noLocale, nil
		},
		listFn: func(context.Context, string, string, ListFilters) (*ListResult, error) {
			return &ListResult{Items: []User{*withLocale, *noLocale}, Meta: Meta{Page: 1, PerPage: 20, Total: 2}}, nil
		},
	}
	h := NewHandler(svc, nil)

	// GET /users/me
	req := withAuthCtx(httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil), "u1", "employee", "dept-1")
	w := httptest.NewRecorder()
	h.GetMe(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	meData, _ := decodeHandlerEnvelope(t, w)
	var me map[string]any
	require.NoError(t, json.Unmarshal(meData, &me))
	assert.Equal(t, "ru", me["preferred_locale"])
	assert.Contains(t, me, "email")
	assert.Contains(t, me, "permissions")

	// GET /users/{id}
	req = withChiID(withAuthCtx(httptest.NewRequest(http.MethodGet, "/api/v1/users/u2", nil), "admin", "super_admin", ""), "u2")
	w = httptest.NewRecorder()
	h.GetUser(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	detailData, _ := decodeHandlerEnvelope(t, w)
	var detail map[string]any
	require.NoError(t, json.Unmarshal(detailData, &detail))
	assert.Contains(t, detail, "preferred_locale")
	assert.Nil(t, detail["preferred_locale"])

	// GET /users (list)
	req = withAuthCtx(httptest.NewRequest(http.MethodGet, "/api/v1/users", nil), "admin", "super_admin", "")
	w = httptest.NewRecorder()
	h.ListUsers(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	listData, _ := decodeHandlerEnvelope(t, w)
	var list struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.Unmarshal(listData, &list))
	require.Len(t, list.Items, 2)
	assert.Equal(t, "ru", list.Items[0]["preferred_locale"])
	assert.Contains(t, list.Items[1], "preferred_locale")
	assert.Nil(t, list.Items[1]["preferred_locale"])
}

// ── ReactivateUser (FR-BB18 AC-13) ───────────────────────────────────────────

func TestHandlerReactivateUser_Returns200AndAuditsOnce(t *testing.T) {
	svc := &mockUserService{
		reactivateFn: func(_ context.Context, _, _, _, _, _ string) error { return nil },
	}
	spy := &spyWriter{}
	h := &Handler{svc: svc, writer: spy}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/u1/reactivate", nil)
	req = withAuthCtx(req, "caller", "super_admin", "")
	req = withChiID(req, "u1")
	w := httptest.NewRecorder()
	h.ReactivateUser(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user reactivated")
	assert.Equal(t, "user.reactivate", spy.action)
	assert.Equal(t, "user", spy.entity)
	require.NotNil(t, spy.id)
	assert.Equal(t, "u1", *spy.id)
}

func TestHandlerReactivateUser_ErrorMapping_NoAudit(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"not found", ErrNotFound, http.StatusNotFound, "NOT_FOUND"},
		{"forbidden", ErrForbidden, http.StatusForbidden, "FORBIDDEN"},
		{"already active", ErrUserAlreadyActive, http.StatusConflict, "USER_ALREADY_ACTIVE"},
		{"unexpected", errors.New("db down"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockUserService{
				reactivateFn: func(_ context.Context, _, _, _, _, _ string) error { return tc.err },
			}
			spy := &spyWriter{}
			h := &Handler{svc: svc, writer: spy}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/users/u1/reactivate", nil)
			req = withAuthCtx(req, "caller", "super_admin", "")
			req = withChiID(req, "u1")
			w := httptest.NewRecorder()
			h.ReactivateUser(w, req)

			assert.Equal(t, tc.status, w.Code)
			_, apiErr := decodeHandlerEnvelope(t, w)
			require.NotNil(t, apiErr)
			assert.Equal(t, tc.code, apiErr.Code)
			assert.Equal(t, 0, spy.n, "a failed reactivation must not be audited")
		})
	}
}
