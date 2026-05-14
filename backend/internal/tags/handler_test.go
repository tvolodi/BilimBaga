package tags

import (
	"context"
	"encoding/json"
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
	listFn   func(ctx context.Context) ([]Tag, error)
	createFn func(ctx context.Context, req CreateRequest) (*Tag, error)
	updateFn func(ctx context.Context, id string, req UpdateRequest) (*Tag, error)
	deleteFn func(ctx context.Context, id string) error
}

func (m *mockSvc) List(ctx context.Context) ([]Tag, error) { return m.listFn(ctx) }
func (m *mockSvc) Create(ctx context.Context, req CreateRequest) (*Tag, error) {
	return m.createFn(ctx, req)
}
func (m *mockSvc) Update(ctx context.Context, id string, req UpdateRequest) (*Tag, error) {
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

func TestList_Returns200(t *testing.T) {
	h := NewHandler(&mockSvc{listFn: func(_ context.Context) ([]Tag, error) {
		return []Tag{{ID: "t1", Name: "gdpr", CreatedAt: time.Now()}}, nil
	}}, nil)
	w := httptest.NewRecorder()
	h.List(w, httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCreate_Duplicate_Returns409(t *testing.T) {
	h := NewHandler(&mockSvc{createFn: func(_ context.Context, _ CreateRequest) (*Tag, error) {
		return nil, ErrDuplicate
	}}, nil)
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/api/v1/tags", strings.NewReader(`{"name":"gdpr"}`)))
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "ERR_TAG_DUPLICATE", decode(t, w).Error.Code)
}

func TestCreate_Invalid_Returns400(t *testing.T) {
	h := NewHandler(&mockSvc{createFn: func(_ context.Context, _ CreateRequest) (*Tag, error) {
		return nil, ErrInvalidName
	}}, nil)
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodPost, "/api/v1/tags", strings.NewReader(`{"name":""}`)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDelete_InUse_Returns409(t *testing.T) {
	h := NewHandler(&mockSvc{deleteFn: func(_ context.Context, _ string) error { return ErrTagInUse }}, nil)
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodDelete, "/api/v1/tags/t1", nil), "id", "t1")
	h.Delete(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "ERR_TAG_IN_USE", decode(t, w).Error.Code)
}

func TestDelete_NotFound_Returns404(t *testing.T) {
	h := NewHandler(&mockSvc{deleteFn: func(_ context.Context, _ string) error { return ErrNotFound }}, nil)
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodDelete, "/api/v1/tags/x", nil), "id", "x")
	h.Delete(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDelete_Success_Returns204(t *testing.T) {
	h := NewHandler(&mockSvc{deleteFn: func(_ context.Context, _ string) error { return nil }}, nil)
	w := httptest.NewRecorder()
	req := withChiParam(httptest.NewRequest(http.MethodDelete, "/api/v1/tags/x", nil), "id", "x")
	h.Delete(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestUpdate_Success_Returns200(t *testing.T) {
	h := NewHandler(&mockSvc{updateFn: func(_ context.Context, id string, req UpdateRequest) (*Tag, error) {
		return &Tag{ID: id, Name: req.Name, CreatedAt: time.Now()}, nil
	}}, nil)
	w := httptest.NewRecorder()
	req := withChiParam(
		httptest.NewRequest(http.MethodPut, "/api/v1/tags/t1", strings.NewReader(`{"name":"renamed"}`)),
		"id", "t1",
	)
	h.Update(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdate_Duplicate_Returns409(t *testing.T) {
	h := NewHandler(&mockSvc{updateFn: func(_ context.Context, _ string, _ UpdateRequest) (*Tag, error) {
		return nil, ErrDuplicate
	}}, nil)
	w := httptest.NewRecorder()
	req := withChiParam(
		httptest.NewRequest(http.MethodPut, "/api/v1/tags/t1", strings.NewReader(`{"name":"taken"}`)),
		"id", "t1",
	)
	h.Update(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "ERR_TAG_DUPLICATE", decode(t, w).Error.Code)
}

func TestUpdate_NotFound_Returns404(t *testing.T) {
	h := NewHandler(&mockSvc{updateFn: func(_ context.Context, _ string, _ UpdateRequest) (*Tag, error) {
		return nil, ErrNotFound
	}}, nil)
	w := httptest.NewRecorder()
	req := withChiParam(
		httptest.NewRequest(http.MethodPut, "/api/v1/tags/missing", strings.NewReader(`{"name":"x"}`)),
		"id", "missing",
	)
	h.Update(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdate_Invalid_Returns400(t *testing.T) {
	h := NewHandler(&mockSvc{updateFn: func(_ context.Context, _ string, _ UpdateRequest) (*Tag, error) {
		return nil, ErrInvalidName
	}}, nil)
	w := httptest.NewRecorder()
	req := withChiParam(
		httptest.NewRequest(http.MethodPut, "/api/v1/tags/t1", strings.NewReader(`{"name":""}`)),
		"id", "t1",
	)
	h.Update(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_BadJSON_Returns400(t *testing.T) {
	h := NewHandler(&mockSvc{}, nil)
	w := httptest.NewRecorder()
	req := withChiParam(
		httptest.NewRequest(http.MethodPut, "/api/v1/tags/t1", strings.NewReader(`not json`)),
		"id", "t1",
	)
	h.Update(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "ERR_INVALID_BODY", decode(t, w).Error.Code)
}

func TestList_PropagatesUsageCount(t *testing.T) {
	h := NewHandler(&mockSvc{listFn: func(_ context.Context) ([]Tag, error) {
		return []Tag{{ID: "t1", Name: "gdpr", UsageCount: 7}}, nil
	}}, nil)
	w := httptest.NewRecorder()
	h.List(w, httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	var out []Tag
	require.NoError(t, json.Unmarshal(decode(t, w).Data, &out))
	require.Len(t, out, 1)
	assert.Equal(t, 7, out[0].UsageCount)
}
