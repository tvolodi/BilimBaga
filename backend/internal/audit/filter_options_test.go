package audit_test

// #557: the audit filter options come from the tenant's own log (GET /audit/actions, audit:read). These tests cover the
// handler envelope, the permission and the tenant scope; the SQL itself runs in filter_options_pg_test.go.

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bilimbaga/bilimbaga/internal/audit"
	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/bilimbaga/bilimbaga/internal/rbac"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_Actions_ReturnsOptionsInTheEnvelope(t *testing.T) {
	svc := &mockService{optionsFn: func(_ context.Context, tenantID string) (audit.AuditFilterOptions, error) {
		assert.Equal(t, "tenant-a", tenantID)
		return audit.AuditFilterOptions{
			Actions:     []string{"auth.login.success", "exam.create"},
			EntityTypes: []string{"exam", "exam_session"},
		}, nil
	}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/actions", nil).WithContext(ctxWithTenant("tenant-a"))

	audit.NewHandler(svc, nil).Actions(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data struct {
			Actions     []string `json:"actions"`
			EntityTypes []string `json:"entity_types"`
		} `json:"data"`
		Error any `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, []string{"auth.login.success", "exam.create"}, body.Data.Actions)
	assert.Equal(t, []string{"exam", "exam_session"}, body.Data.EntityTypes)
	assert.Nil(t, body.Error)
}

// An empty log answers with empty arrays, never null, so the frontend can map over them.
func TestHandler_Actions_EmptyLogSendsEmptyArrays(t *testing.T) {
	svc := &mockService{optionsFn: func(_ context.Context, _ string) (audit.AuditFilterOptions, error) {
		return audit.AuditFilterOptions{Actions: []string{}, EntityTypes: []string{}}, nil
	}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/actions", nil).WithContext(ctxWithTenant("tenant-a"))

	audit.NewHandler(svc, nil).Actions(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"data":{"actions":[],"entity_types":[]},"error":null}`, rec.Body.String())
}

func TestHandler_Actions_MissingTenant_Returns401(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/actions", nil)

	audit.NewHandler(&mockService{}, nil).Actions(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Actions_ServiceError_Returns500(t *testing.T) {
	svc := &mockService{optionsFn: func(_ context.Context, _ string) (audit.AuditFilterOptions, error) {
		return audit.AuditFilterOptions{}, errors.New("db down")
	}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/actions", nil).WithContext(ctxWithTenant("tenant-a"))

	audit.NewHandler(svc, nil).Actions(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "db down", "the database error is not sent to the client")
}

// The route is registered behind the same permission as GET /audit (audit:read). This mirrors router.go: the
// middleware and the handler are composed the same way.
func TestHandler_Actions_RequiresAuditRead(t *testing.T) {
	cache := rbac.NewCache()
	cache.LoadFromMap(map[string]rbac.PermissionSet{
		"auditor":  {"audit:read": true},
		"employee": {"portal:read": true},
	})
	svc := &mockService{optionsFn: func(_ context.Context, _ string) (audit.AuditFilterOptions, error) {
		return audit.AuditFilterOptions{Actions: []string{"exam.create"}, EntityTypes: []string{"exam"}}, nil
	}}
	r := chi.NewRouter()
	r.With(rbac.RequirePermission(cache, "audit", "read")).Get("/audit/actions", audit.NewHandler(svc, nil).Actions)

	call := func(role string) int {
		ctx := context.WithValue(ctxWithTenant("tenant-a"), ctxkeys.CtxRole, role)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/audit/actions", nil).WithContext(ctx))
		return rec.Code
	}

	assert.Equal(t, http.StatusOK, call("auditor"))
	assert.Equal(t, http.StatusForbidden, call("employee"), "a role without audit:read gets 403")
}

// The options are read for the caller's tenant only: the tenant id is the first argument of both queries.
func TestService_FilterOptions_ReadsOnlyTheCallersTenant(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"action"}, [][]driver.Value{{"exam.create"}, {"user.login"}})
	f.queue([]string{"entity_type"}, [][]driver.Value{{"exam"}})

	opts, err := audit.NewService(db).FilterOptions(context.Background(), "tenant-a")
	require.NoError(t, err)

	assert.Equal(t, []string{"exam.create", "user.login"}, opts.Actions)
	assert.Equal(t, []string{"exam"}, opts.EntityTypes)
	require.Len(t, f.args, 2)
	assert.Equal(t, driver.Value("tenant-a"), f.args[0][0])
	assert.Equal(t, driver.Value("tenant-a"), f.args[1][0])
	for _, q := range f.queries {
		assert.Contains(t, q, "tenant_id = $1")
		assert.Contains(t, q, `COLLATE "C"`, "sorting does not depend on the database locale")
	}
}

func TestService_FilterOptions_EmptyLogIsEmptyNotNil(t *testing.T) {
	db, f := newFakeDB(t)
	f.queue([]string{"action"}, nil)
	f.queue([]string{"entity_type"}, nil)

	opts, err := audit.NewService(db).FilterOptions(context.Background(), "tenant-a")
	require.NoError(t, err)

	assert.NotNil(t, opts.Actions)
	assert.Empty(t, opts.Actions)
	assert.NotNil(t, opts.EntityTypes)
	assert.Empty(t, opts.EntityTypes)
}

func TestService_FilterOptions_QueryErrorIsReturned(t *testing.T) {
	db, f := newFakeDB(t)
	f.qErr = errors.New("connection reset")

	_, err := audit.NewService(db).FilterOptions(context.Background(), "tenant-a")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection reset")
}
