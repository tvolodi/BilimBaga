package categories

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

type mockSvc struct {
	listFn   func(ctx context.Context) ([]*Node, error)
	createFn func(ctx context.Context, req CreateRequest) (*Node, error)
	updateFn func(ctx context.Context, id string, req UpdateRequest) (*Node, error)
	deleteFn func(ctx context.Context, id string) error
}

func (m *mockSvc) ListTree(ctx context.Context) ([]*Node, error) { return m.listFn(ctx) }
func (m *mockSvc) Create(ctx context.Context, req CreateRequest) (*Node, error) {
	return m.createFn(ctx, req)
}
func (m *mockSvc) Update(ctx context.Context, id string, req UpdateRequest) (*Node, error) {
	return m.updateFn(ctx, id, req)
}
func (m *mockSvc) Delete(ctx context.Context, id string) error { return m.deleteFn(ctx, id) }

type envelope struct {
	Data  json.RawMessage `json:"data"`
	Error *errBody        `json:"error"`
}
type errBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func decode(t *testing.T, w *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	return env
}

func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func sampleNode() *Node {
	return &Node{ID: "id-1", Name: "Sample", CreatedAt: time.Now(), UpdatedAt: time.Now(), Children: []*Node{}}
}

func TestList_Returns200(t *testing.T) {
	h := NewHandler(&mockSvc{listFn: func(_ context.Context) ([]*Node, error) {
		return []*Node{sampleNode()}, nil
	}}, nil)
	w := httptest.NewRecorder()
	h.ListTree(w, httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, decode(t, w).Error)
}

func TestList_ServiceError(t *testing.T) {
	h := NewHandler(&mockSvc{listFn: func(_ context.Context) ([]*Node, error) {
		return nil, errors.New("boom")
	}}, nil)
	w := httptest.NewRecorder()
	h.ListTree(w, httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreate_InvalidBody(t *testing.T) {
	h := NewHandler(&mockSvc{}, nil)
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader("{")))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "ERR_INVALID_BODY", decode(t, w).Error.Code)
}

func TestCreateHandler_InvalidName_Returns400(t *testing.T) {
	h := NewHandler(&mockSvc{createFn: func(_ context.Context, _ CreateRequest) (*Node, error) {
		return nil, ErrInvalidName
	}}, nil)
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(`{"name":""}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "ERR_INVALID_NAME", decode(t, w).Error.Code)
}

func TestCreate_Success(t *testing.T) {
	h := NewHandler(&mockSvc{createFn: func(_ context.Context, _ CreateRequest) (*Node, error) {
		return sampleNode(), nil
	}}, nil)
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(`{"name":"X"}`)))
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestUpdate_Cycle_Returns400(t *testing.T) {
	h := NewHandler(&mockSvc{updateFn: func(_ context.Context, _ string, _ UpdateRequest) (*Node, error) {
		return nil, ErrCycle
	}}, nil)
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodPut, "/api/v1/categories/a", strings.NewReader(`{}`)), "id", "a")
	h.Update(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "ERR_CATEGORY_CYCLE", decode(t, w).Error.Code)
}

func TestDelete_InUse_Returns409(t *testing.T) {
	h := NewHandler(&mockSvc{deleteFn: func(_ context.Context, _ string) error { return ErrCategoryInUse }}, nil)
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodDelete, "/api/v1/categories/a", nil), "id", "a")
	h.Delete(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "ERR_CATEGORY_IN_USE", decode(t, w).Error.Code)
}

func TestDelete_Success_Returns204(t *testing.T) {
	h := NewHandler(&mockSvc{deleteFn: func(_ context.Context, _ string) error { return nil }}, nil)
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodDelete, "/api/v1/categories/a", nil), "id", "a")
	h.Delete(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}
