package audit_test

import (
	"context"
	"encoding/json"
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
	entry := audit.AuditEntry{
		ID:        "aaaaaaaa-0000-4000-8000-000000000003",
		TenantID:  "public",
		ActorID:   &actorID,
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
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "audit_export.csv")
	// CSV body should contain at least the header row and the entry row.
	body := w.Body.String()
	assert.Contains(t, body, "id")
	assert.Contains(t, body, entry.ID)
}

func TestHandler_Export_MissingTenant_Returns401(t *testing.T) {
	h := audit.NewHandler(&mockService{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/export", nil)
	w := httptest.NewRecorder()

	h.Export(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
