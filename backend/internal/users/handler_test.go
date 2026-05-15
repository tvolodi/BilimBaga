package users

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	resetPwdFn   func(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) (*ResetPasswordResponse, error)
	importFn     func(ctx context.Context, rows []CSVRow, commit bool, callerRole, callerDeptID, callerUserID, ip string) (*ImportPreview, error)
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
func (m *mockUserService) ResetPassword(ctx context.Context, id, callerRole, callerDeptID, callerUserID, ip string) (*ResetPasswordResponse, error) {
	return m.resetPwdFn(ctx, id, callerRole, callerDeptID, callerUserID, ip)
}
func (m *mockUserService) ImportUsers(ctx context.Context, rows []CSVRow, commit bool, callerRole, callerDeptID, callerUserID, ip string) (*ImportPreview, error) {
	return m.importFn(ctx, rows, commit, callerRole, callerDeptID, callerUserID, ip)
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
