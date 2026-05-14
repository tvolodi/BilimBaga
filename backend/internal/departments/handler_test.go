package departments

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Service ---

type mockService struct {
	listTreeFn func(ctx context.Context) ([]*DepartmentNode, error)
	createFn   func(ctx context.Context, req CreateRequest, userID, ip string) (*DepartmentNode, error)
	updateFn   func(ctx context.Context, id string, req UpdateRequest, userID, ip string) (*DepartmentNode, error)
	deleteFn   func(ctx context.Context, id, userID, ip string) error
}

func (m *mockService) ListTree(ctx context.Context) ([]*DepartmentNode, error) {
	return m.listTreeFn(ctx)
}

func (m *mockService) Create(ctx context.Context, req CreateRequest, userID, ip string) (*DepartmentNode, error) {
	return m.createFn(ctx, req, userID, ip)
}

func (m *mockService) Update(ctx context.Context, id string, req UpdateRequest, userID, ip string) (*DepartmentNode, error) {
	return m.updateFn(ctx, id, req, userID, ip)
}

func (m *mockService) Delete(ctx context.Context, id, userID, ip string) error {
	return m.deleteFn(ctx, id, userID, ip)
}

// --- Helpers ---

type envelope struct {
	Data  json.RawMessage `json:"data"`
	Error *apiErrBody     `json:"error"`
}

type apiErrBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) (json.RawMessage, *apiErrBody) {
	t.Helper()
	var env envelope
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	return env.Data, env.Error
}

func sampleNode() *DepartmentNode {
	return &DepartmentNode{
		ID:        "dept-1",
		Name:      "Engineering",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Children:  []*DepartmentNode{},
	}
}

func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// --- ListTree ---

func TestListTree_Returns200(t *testing.T) {
	svc := &mockService{
		listTreeFn: func(_ context.Context) ([]*DepartmentNode, error) {
			return []*DepartmentNode{sampleNode()}, nil
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/departments", nil)
	w := httptest.NewRecorder()
	h.ListTree(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data, apiErr := decodeEnvelope(t, w)
	assert.Nil(t, apiErr)
	assert.NotEmpty(t, data)
}

func TestListTree_ServiceError_Returns500(t *testing.T) {
	svc := &mockService{
		listTreeFn: func(_ context.Context) ([]*DepartmentNode, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/departments", nil)
	w := httptest.NewRecorder()
	h.ListTree(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "INTERNAL_ERROR", apiErr.Code)
}

// --- Create ---

func TestCreate_Returns201(t *testing.T) {
	svc := &mockService{
		createFn: func(_ context.Context, _ CreateRequest, _, _ string) (*DepartmentNode, error) {
			return sampleNode(), nil
		},
	}
	h := NewHandler(svc, nil)

	body := `{"name":"Engineering"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/departments", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Create(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	data, apiErr := decodeEnvelope(t, w)
	assert.Nil(t, apiErr)
	assert.NotEmpty(t, data)
}

func TestCreate_EmptyName_Returns400(t *testing.T) {
	h := NewHandler(&mockService{}, nil)

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/departments", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Create(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "INVALID_BODY", apiErr.Code)
}

func TestCreate_ParentNotFound_Returns404(t *testing.T) {
	svc := &mockService{
		createFn: func(_ context.Context, _ CreateRequest, _, _ string) (*DepartmentNode, error) {
			return nil, ErrNotFound
		},
	}
	h := NewHandler(svc, nil)

	body := `{"name":"Child","parent_id":"missing"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/departments", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Create(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "NOT_FOUND", apiErr.Code)
}

func TestCreate_DuplicateName_Returns409(t *testing.T) {
	svc := &mockService{
		createFn: func(_ context.Context, _ CreateRequest, _, _ string) (*DepartmentNode, error) {
			return nil, ErrDuplicateName
		},
	}
	h := NewHandler(svc, nil)

	body := `{"name":"Duplicate"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/departments", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Create(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "DUPLICATE_NAME", apiErr.Code)
}

// --- Update ---

func TestUpdate_Returns200(t *testing.T) {
	svc := &mockService{
		updateFn: func(_ context.Context, _ string, _ UpdateRequest, _, _ string) (*DepartmentNode, error) {
			return sampleNode(), nil
		},
	}
	h := NewHandler(svc, nil)

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/departments/dept-1", strings.NewReader(body))
	req = withChiParam(req, "id", "dept-1")
	w := httptest.NewRecorder()
	h.Update(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	data, apiErr := decodeEnvelope(t, w)
	assert.Nil(t, apiErr)
	assert.NotEmpty(t, data)
}

func TestUpdate_EmptyName_Returns400(t *testing.T) {
	h := NewHandler(&mockService{}, nil)

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/departments/dept-1", strings.NewReader(body))
	req = withChiParam(req, "id", "dept-1")
	w := httptest.NewRecorder()
	h.Update(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_NotFound_Returns404(t *testing.T) {
	svc := &mockService{
		updateFn: func(_ context.Context, _ string, _ UpdateRequest, _, _ string) (*DepartmentNode, error) {
			return nil, ErrNotFound
		},
	}
	h := NewHandler(svc, nil)

	body := `{"name":"X"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/departments/missing", strings.NewReader(body))
	req = withChiParam(req, "id", "missing")
	w := httptest.NewRecorder()
	h.Update(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "NOT_FOUND", apiErr.Code)
}

// --- Delete ---

func TestDelete_Returns204(t *testing.T) {
	svc := &mockService{
		deleteFn: func(_ context.Context, _, _, _ string) error { return nil },
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/departments/dept-1", nil)
	req = withChiParam(req, "id", "dept-1")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDelete_NotFound_Returns404(t *testing.T) {
	svc := &mockService{
		deleteFn: func(_ context.Context, _, _, _ string) error { return ErrNotFound },
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/departments/missing", nil)
	req = withChiParam(req, "id", "missing")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "NOT_FOUND", apiErr.Code)
}

func TestDelete_HasChildren_Returns409(t *testing.T) {
	svc := &mockService{
		deleteFn: func(_ context.Context, _, _, _ string) error { return ErrDepartmentHasChildren },
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/departments/dept-1", nil)
	req = withChiParam(req, "id", "dept-1")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "DEPARTMENT_HAS_CHILDREN", apiErr.Code)
}

func TestDelete_HasUsers_Returns409(t *testing.T) {
	svc := &mockService{
		deleteFn: func(_ context.Context, _, _, _ string) error { return ErrDepartmentNotEmpty },
	}
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/departments/dept-1", nil)
	req = withChiParam(req, "id", "dept-1")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	_, apiErr := decodeEnvelope(t, w)
	require.NotNil(t, apiErr)
	assert.Equal(t, "DEPARTMENT_NOT_EMPTY", apiErr.Code)
}
