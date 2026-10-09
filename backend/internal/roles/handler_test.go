package roles

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bilimbaga/bilimbaga/internal/ctxkeys"
	"github.com/bilimbaga/bilimbaga/internal/rbac"
)

type mockService struct {
	listFn   func() ([]Role, error)
	getFn    func(id string) (*Role, error)
	permsFn  func() ([]Permission, error)
	createFn func(req CreateRequest) (*Role, error)
	updateFn func(id string, req UpdateRequest) (*UpdateResult, error)
	deleteFn func(id string) (*Role, error)
}

func (m *mockService) List(context.Context) ([]Role, error) { return m.listFn() }
func (m *mockService) Get(_ context.Context, id string) (*Role, error) {
	return m.getFn(id)
}
func (m *mockService) ListPermissions(context.Context) ([]Permission, error) { return m.permsFn() }
func (m *mockService) Create(_ context.Context, r CreateRequest) (*Role, error) {
	return m.createFn(r)
}
func (m *mockService) Update(_ context.Context, id string, r UpdateRequest) (*UpdateResult, error) {
	return m.updateFn(id, r)
}
func (m *mockService) Delete(_ context.Context, id string) (*Role, error) { return m.deleteFn(id) }

type auditCall struct {
	action, entityType string
	entityID           string
	meta               map[string]any
}

type fakeAudit struct{ calls []auditCall }

func (f *fakeAudit) Write(_ context.Context, _ *http.Request, action, entityType string, entityID *string, metadata any) {
	c := auditCall{action: action, entityType: entityType}
	if entityID != nil {
		c.entityID = *entityID
	}
	c.meta, _ = metadata.(map[string]any)
	f.calls = append(f.calls, c)
}

func do(h http.HandlerFunc, method, body, id string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v1/roles", strings.NewReader(body))
	if id != "" {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}
	w := httptest.NewRecorder()
	h(w, req)
	return w
}

type errBody struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decode(t *testing.T, w *httptest.ResponseRecorder) errBody {
	t.Helper()
	var e errBody
	require.NoError(t, json.NewDecoder(w.Body).Decode(&e))
	return e
}

func newH(svc *mockService) (*Handler, *fakeAudit) {
	a := &fakeAudit{}
	return NewHandler(svc, a), a
}

func TestHandlerList(t *testing.T) {
	h, _ := newH(&mockService{listFn: func() ([]Role, error) { return []Role{{ID: "r", Name: "x", Permissions: []string{}}}, nil }})
	w := do(h.List, http.MethodGet, "", "")
	assert.Equal(t, http.StatusOK, w.Code)
	e := decode(t, w)
	assert.Nil(t, e.Error)
	assert.Contains(t, string(e.Data), `"user_count"`)
	assert.Contains(t, string(e.Data), `"is_system"`)
}

func TestHandlerList_NilBecomesEmptyArray(t *testing.T) {
	h, _ := newH(&mockService{listFn: func() ([]Role, error) { return nil, nil }})
	assert.JSONEq(t, `{"data":[],"error":null}`, do(h.List, http.MethodGet, "", "").Body.String())
}

func TestHandlerList_Error500(t *testing.T) {
	h, _ := newH(&mockService{listFn: func() ([]Role, error) { return nil, errors.New("db") }})
	w := do(h.List, http.MethodGet, "", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "INTERNAL", decode(t, w).Error.Code)
}

func TestHandlerListPermissions(t *testing.T) {
	h, _ := newH(&mockService{permsFn: func() ([]Permission, error) { return []Permission{{ID: "p", Resource: "users", Action: "read"}}, nil }})
	w := do(h.ListPermissions, http.MethodGet, "", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"resource":"users"`)
}

func TestHandlerGet(t *testing.T) {
	h, _ := newH(&mockService{getFn: func(id string) (*Role, error) {
		if id == "missing" {
			return nil, ErrNotFound
		}
		return &Role{ID: id, Name: "x"}, nil
	}})
	assert.Equal(t, http.StatusOK, do(h.Get, http.MethodGet, "", "r1").Code)
	w := do(h.Get, http.MethodGet, "", "missing")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "NOT_FOUND", decode(t, w).Error.Code)
}

func TestHandlerCreate_201AndAudit(t *testing.T) {
	h, a := newH(&mockService{createFn: func(req CreateRequest) (*Role, error) {
		return &Role{ID: "rid", Name: req.Name, Permissions: []string{"users:read"}}, nil
	}})
	w := do(h.Create, http.MethodPost, `{"name":"qa_lead","permissions":["p1"]}`, "")
	assert.Equal(t, http.StatusCreated, w.Code)
	require.Len(t, a.calls, 1)
	assert.Equal(t, "role.create", a.calls[0].action)
	assert.Equal(t, "role", a.calls[0].entityType)
	assert.Equal(t, "rid", a.calls[0].entityID)
	assert.Equal(t, "qa_lead", a.calls[0].meta["name"])
}

func TestHandlerCreate_Errors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"validation", ErrValidation, http.StatusUnprocessableEntity, "VALIDATION_ERROR"},
		{"duplicate", ErrNameTaken, http.StatusConflict, "ROLE_NAME_TAKEN"},
		{"cache", ErrCacheReload, http.StatusInternalServerError, "INTERNAL"},
	}
	for _, c := range cases {
		h, a := newH(&mockService{createFn: func(CreateRequest) (*Role, error) { return nil, c.err }})
		w := do(h.Create, http.MethodPost, `{"name":"abc"}`, "")
		assert.Equal(t, c.status, w.Code, c.name)
		assert.Equal(t, c.code, decode(t, w).Error.Code, c.name)
		assert.Empty(t, a.calls, "failed requests write no audit entry: "+c.name)
	}
}

func TestHandlerCreate_BadJSON(t *testing.T) {
	h, a := newH(&mockService{})
	w := do(h.Create, http.MethodPost, `{`, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, a.calls)
}

func TestHandlerUpdate_200AndAuditDiff(t *testing.T) {
	h, a := newH(&mockService{updateFn: func(id string, _ UpdateRequest) (*UpdateResult, error) {
		return &UpdateResult{Role: &Role{ID: id, Name: "qa_lead"}, PermissionsAdded: []string{"exams:read"}, PermissionsRemoved: []string{"users:read"}}, nil
	}})
	w := do(h.Update, http.MethodPut, `{"description":"d","permissions":[]}`, "rid")
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, a.calls, 1)
	assert.Equal(t, "role.update", a.calls[0].action)
	assert.Equal(t, "rid", a.calls[0].entityID)
	assert.Equal(t, "qa_lead", a.calls[0].meta["name"])
	assert.Equal(t, []string{"exams:read"}, a.calls[0].meta["permissions_added"])
	assert.Equal(t, []string{"users:read"}, a.calls[0].meta["permissions_removed"])
}

func TestHandlerUpdate_Errors(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{ErrSystemImmutable, http.StatusConflict, "ROLE_SYSTEM_IMMUTABLE"},
		{ErrNotFound, http.StatusNotFound, "NOT_FOUND"},
		{ErrValidation, http.StatusUnprocessableEntity, "VALIDATION_ERROR"},
	}
	for _, c := range cases {
		h, a := newH(&mockService{updateFn: func(string, UpdateRequest) (*UpdateResult, error) { return nil, c.err }})
		w := do(h.Update, http.MethodPut, `{"permissions":[]}`, "rid")
		assert.Equal(t, c.status, w.Code)
		assert.Equal(t, c.code, decode(t, w).Error.Code)
		assert.Empty(t, a.calls)
	}
}

func TestHandlerDelete_204AndAudit(t *testing.T) {
	h, a := newH(&mockService{deleteFn: func(id string) (*Role, error) { return &Role{ID: id, Name: "qa_lead"}, nil }})
	w := do(h.Delete, http.MethodDelete, "", "rid")
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
	require.Len(t, a.calls, 1)
	assert.Equal(t, "role.delete", a.calls[0].action)
	assert.Equal(t, "qa_lead", a.calls[0].meta["name"])
}

func TestHandlerDelete_Errors(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
		msg    string
	}{
		{ErrSystemImmutable, http.StatusConflict, "ROLE_SYSTEM_IMMUTABLE", ""},
		{&InUseError{Count: 4}, http.StatusConflict, "ROLE_IN_USE", "4"},
		{ErrNotFound, http.StatusNotFound, "NOT_FOUND", ""},
		{errors.New("db"), http.StatusInternalServerError, "INTERNAL", ""},
	}
	for _, c := range cases {
		h, a := newH(&mockService{deleteFn: func(string) (*Role, error) { return nil, c.err }})
		w := do(h.Delete, http.MethodDelete, "", "rid")
		assert.Equal(t, c.status, w.Code)
		e := decode(t, w)
		assert.Equal(t, c.code, e.Error.Code)
		assert.Contains(t, e.Error.Message, c.msg)
		assert.Empty(t, a.calls)
	}
}

// AC-10: routes are guarded by RequirePermission; only super_admin holds roles:*.
func TestRoutesRequireRolesPermissions_403ForNonSuperAdmin(t *testing.T) {
	cache := rbac.NewCache()
	cache.LoadFromMap(map[string]rbac.PermissionSet{
		"super_admin":      {"roles:read": true, "roles:manage": true},
		"department_admin": {"users:read": true},
		"employee":         {"portal:read": true},
	})
	h, _ := newH(&mockService{
		listFn:   func() ([]Role, error) { return []Role{}, nil },
		deleteFn: func(id string) (*Role, error) { return &Role{ID: id}, nil },
	})
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				role := req.Header.Get("X-Test-Role")
				next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), ctxkeys.CtxRole, role)))
			})
		})
		r.With(rbac.RequirePermission(cache, "roles", "read")).Get("/roles", h.List)
		r.With(rbac.RequirePermission(cache, "roles", "manage")).Delete("/roles/{id}", h.Delete)
	})
	for _, role := range []string{"department_admin", "employee", "qa_lead"} {
		for _, rq := range []struct{ m, p string }{{http.MethodGet, "/roles"}, {http.MethodDelete, "/roles/x"}} {
			req := httptest.NewRequest(rq.m, rq.p, nil)
			req.Header.Set("X-Test-Role", role)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code, role+" "+rq.m)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/roles", nil)
	req.Header.Set("X-Test-Role", "super_admin")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
