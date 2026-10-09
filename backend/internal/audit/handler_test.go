package audit_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/audit"
	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockService is a test double for audit.Service.
type mockService struct {
	listFn   func(ctx context.Context, tenantID string, filters audit.AuditFilters, page, perPage int) ([]audit.AuditEntry, int, error)
	exportFn func(ctx context.Context, tenantID string, filters audit.AuditFilters) ([]audit.AuditEntry, error)
}

func (m *mockService) List(ctx context.Context, tenantID string, filters audit.AuditFilters, page, perPage int) ([]audit.AuditEntry, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, filters, page, perPage)
	}
	return nil, 0, nil
}

func (m *mockService) Export(ctx context.Context, tenantID string, filters audit.AuditFilters) ([]audit.AuditEntry, error) {
	if m.exportFn != nil {
		return m.exportFn(ctx, tenantID, filters)
	}
	return nil, nil
}

func ctxWithTenant(tenantID string) context.Context {
	return context.WithValue(context.Background(), ctxkeys.CtxTenantID, tenantID)
}

func TestHandler_List_ReturnsEntries(t *testing.T) {
	entry := audit.AuditEntry{
		ID:        "aaaaaaaa-0000-4000-8000-000000000001",
		TenantID:  "public",
		Action:    "auth.login.success",
		CreatedAt: time.Now().UTC(),
	}
	svc := &mockService{
		listFn: func(_ context.Context, tenantID string, _ audit.AuditFilters, page, perPage int) ([]audit.AuditEntry, int, error) {
			assert.Equal(t, "public", tenantID)
			assert.Equal(t, 1, page)
			assert.Equal(t, 50, perPage)
			return []audit.AuditEntry{entry}, 1, nil
		},
	}
	h := audit.NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
	req = req.WithContext(ctxWithTenant("public"))
	w := httptest.NewRecorder()

	h.List(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Nil(t, body["error"])
	data := body["data"].(map[string]any)
	items := data["items"].([]any)
	assert.Len(t, items, 1)
	meta := data["meta"].(map[string]any)
	assert.Equal(t, float64(1), meta["total"])
	assert.Equal(t, float64(1), meta["page"])
	assert.Equal(t, float64(50), meta["per_page"])
}

func TestHandler_List_MissingTenant_Returns401(t *testing.T) {
	h := audit.NewHandler(&mockService{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Export_ReturnsCsv(t *testing.T) {
	actorID := "aaaaaaaa-0000-4000-8000-000000000002"
	actorName := "Test User"
	entry := audit.AuditEntry{
		ID:        "aaaaaaaa-0000-4000-8000-000000000003",
		TenantID:  "public",
		ActorID:   &actorID,
		ActorName: &actorName,
		Action:    "user.create",
		CreatedAt: time.Now().UTC(),
	}
	svc := &mockService{
		exportFn: func(_ context.Context, tenantID string, _ audit.AuditFilters) ([]audit.AuditEntry, error) {
			assert.Equal(t, "public", tenantID)
			return []audit.AuditEntry{entry}, nil
		},
	}
	h := audit.NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/export", nil)
	req = req.WithContext(ctxWithTenant("public"))
	w := httptest.NewRecorder()

	h.Export(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
	assert.Contains(t, w.Header().Get("Content-Disposition"), "audit-")
	assert.Contains(t, w.Header().Get("Content-Disposition"), ".csv")
	// CSV body should contain the AC-8 header columns and entry data.
	body := w.Body.String()
	assert.Contains(t, body, "timestamp")
	assert.Contains(t, body, "actor_name")
	assert.Contains(t, body, "metadata_json")
	assert.Contains(t, body, entry.Action)
	assert.Contains(t, body, actorName)
}

func TestHandler_Export_MissingTenant_Returns401(t *testing.T) {
	h := audit.NewHandler(&mockService{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/export", nil)
	w := httptest.NewRecorder()

	h.Export(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_List_ParsesFiltersAndPaging(t *testing.T) {
	var got audit.AuditFilters
	var gotPage, gotPer int
	svc := &mockService{
		listFn: func(_ context.Context, _ string, f audit.AuditFilters, page, perPage int) ([]audit.AuditEntry, int, error) {
			got, gotPage, gotPer = f, page, perPage
			return nil, 0, nil
		},
	}
	h := audit.NewHandler(svc, nil)
	url := "/api/v1/audit?actor_id=3f2b8c1e-9d4a-4b6e-8a1f-0c7d5e9a1b22&actor=ann&action=a.b&entity_type=exam" +
		"&from=2026-01-02T03:04:05Z&to=not-a-date&page=3&per_page=abc"
	req := httptest.NewRequest(http.MethodGet, url, nil).WithContext(ctxWithTenant("public"))
	w := httptest.NewRecorder()

	h.List(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, got.ActorID)
	assert.Equal(t, "3f2b8c1e-9d4a-4b6e-8a1f-0c7d5e9a1b22", *got.ActorID)
	assert.Equal(t, "ann", *got.Actor)
	assert.Equal(t, "a.b", *got.Action)
	assert.Equal(t, "exam", *got.EntityType)
	require.NotNil(t, got.From)
	assert.Equal(t, 2026, got.From.Year())
	assert.Nil(t, got.To, "unparseable date is ignored")
	assert.Equal(t, 3, gotPage)
	assert.Equal(t, 50, gotPer, "invalid per_page falls back to default")
}

func TestHandler_List_NonPositivePage_UsesDefault(t *testing.T) {
	var gotPage int
	svc := &mockService{
		listFn: func(_ context.Context, _ string, _ audit.AuditFilters, page, _ int) ([]audit.AuditEntry, int, error) {
			gotPage = page
			return nil, 0, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit?page=-2", nil).WithContext(ctxWithTenant("public"))
	audit.NewHandler(svc, nil).List(httptest.NewRecorder(), req)
	assert.Equal(t, 1, gotPage)
}

func TestHandler_List_ServiceError_Returns500(t *testing.T) {
	svc := &mockService{
		listFn: func(context.Context, string, audit.AuditFilters, int, int) ([]audit.AuditEntry, int, error) {
			return nil, 0, errors.New("db")
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil).WithContext(ctxWithTenant("public"))
	w := httptest.NewRecorder()
	audit.NewHandler(svc, nil).List(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_Export_ServiceError_Returns500(t *testing.T) {
	svc := &mockService{
		exportFn: func(context.Context, string, audit.AuditFilters) ([]audit.AuditEntry, error) {
			return nil, errors.New("db")
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/export?action=x", nil).WithContext(ctxWithTenant("public"))
	w := httptest.NewRecorder()
	audit.NewHandler(svc, nil).Export(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ISS-141: actor_id is a UUID column; malformed values must 422, not 500.
func TestHandler_List_ActorIDNonUUID_Returns422(t *testing.T) {
	called := false
	svc := &mockService{
		listFn: func(_ context.Context, _ string, _ audit.AuditFilters, _, _ int) ([]audit.AuditEntry, int, error) {
			called = true
			return nil, 0, nil
		},
		exportFn: func(_ context.Context, _ string, _ audit.AuditFilters) ([]audit.AuditEntry, error) {
			called = true
			return nil, nil
		},
	}
	h := audit.NewHandler(svc, nil)

	for name, fn := range map[string]http.HandlerFunc{"list": h.List, "export": h.Export} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit?actor_id=u1", nil).WithContext(ctxWithTenant("public"))
		w := httptest.NewRecorder()
		fn(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code, name)
		assert.Contains(t, w.Body.String(), "VALIDATION_ERROR", name)
	}
	assert.False(t, called)
}

// ISS-158: user_id (alias of actor_id) must be validated, not silently ignored.
func TestHandler_UserIDNonUUID_Returns422(t *testing.T) {
	called := false
	svc := &mockService{
		listFn: func(_ context.Context, _ string, _ audit.AuditFilters, _, _ int) ([]audit.AuditEntry, int, error) {
			called = true
			return nil, 0, nil
		},
		exportFn: func(_ context.Context, _ string, _ audit.AuditFilters) ([]audit.AuditEntry, error) {
			called = true
			return nil, nil
		},
	}
	h := audit.NewHandler(svc, nil)
	for name, fn := range map[string]http.HandlerFunc{"list": h.List, "export": h.Export} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit?user_id=bad", nil).WithContext(ctxWithTenant("public"))
		w := httptest.NewRecorder()
		fn(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code, name)
		assert.Contains(t, w.Body.String(), "VALIDATION_ERROR", name)
	}
	assert.False(t, called)
}

func TestHandler_UserIDValid_PassedAsActorFilter(t *testing.T) {
	const id = "3f2b8c1e-9d4a-4b6e-8a1f-0c7d5e9a1b22"
	var got audit.AuditFilters
	svc := &mockService{
		listFn: func(_ context.Context, _ string, f audit.AuditFilters, _, _ int) ([]audit.AuditEntry, int, error) {
			got = f
			return nil, 0, nil
		},
	}
	h := audit.NewHandler(svc, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit?user_id="+id, nil).WithContext(ctxWithTenant("public"))
	w := httptest.NewRecorder()
	h.List(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, got.ActorID)
	assert.Equal(t, id, *got.ActorID)
}
