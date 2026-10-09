package users

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISS-229: GET /users/roles annotates each role with `assignable` for the current caller.

var rolesFixture = map[string]string{
	"r-sa": "super_admin", "r-da": "department_admin", "r-ex": "examiner", "r-emp": "employee",
	"r-narrow": "narrow_custom", "r-portal": "portal_custom", "r-big": "big_custom",
	"r-excust": "ex_custom", "r-sens": "sensitive_custom", "r-helper": d1Custom, "r-ghost": "ghost_role",
}

func rolesSvc() (*mockRepo, Service) {
	repo, svc := d1Setup()
	for id, name := range rolesFixture {
		repo.roleByID[id] = name
		repo.roles[name] = id
	}
	return repo, svc
}

func assignableMap(t *testing.T, svc Service, caller string) map[string]bool {
	t.Helper()
	rows, err := svc.ListRoles(context.Background(), caller)
	require.NoError(t, err)
	out := map[string]bool{}
	for _, r := range rows {
		out[r.Name] = r.Assignable
	}
	return out
}

func TestListRoles_SuperAdminAllAssignable(t *testing.T) {
	_, svc := rolesSvc()
	m := assignableMap(t, svc, "super_admin")
	require.Len(t, m, len(rolesFixture))
	for name, ok := range m {
		assert.True(t, ok, name)
	}
}

func TestListRoles_DepartmentAdmin(t *testing.T) {
	_, svc := rolesSvc()
	m := assignableMap(t, svc, "department_admin")
	assert.True(t, m["examiner"])
	assert.True(t, m["employee"])
	assert.False(t, m["department_admin"])
	assert.False(t, m["super_admin"])
	// custom roles: subset of the caller's permissions only
	assert.True(t, m["narrow_custom"])
	assert.True(t, m[d1Custom])
	assert.False(t, m["portal_custom"])
	assert.False(t, m["big_custom"])
	assert.False(t, m["ex_custom"])
	assert.False(t, m["sensitive_custom"])
	// unknown permissions (role absent from cache) fail closed
	assert.False(t, m["ghost_role"])
}

func TestListRoles_CustomRoleCaller(t *testing.T) {
	_, svc := rolesSvc()
	m := assignableMap(t, svc, d1Custom)
	assert.False(t, m["super_admin"])
	assert.False(t, m["department_admin"])
	assert.True(t, m["narrow_custom"])
	assert.False(t, m["examiner"], "examiner permissions are not a subset of the caller's")
	assert.False(t, m["employee"])
	assert.False(t, m["ghost_role"])
}

func TestListRoles_BuiltinExaminerAndEmployee(t *testing.T) {
	_, svc := rolesSvc()
	m := assignableMap(t, svc, "examiner")
	assert.True(t, m["employee"])
	assert.False(t, m["examiner"])
	assert.False(t, m["department_admin"])
	e := assignableMap(t, svc, "employee")
	for name, ok := range e {
		assert.False(t, ok, name)
	}
}

func TestListRoles_UnknownCallerFailsClosed(t *testing.T) {
	_, svc := rolesSvc()
	for _, caller := range []string{"", "no_such_role"} {
		for name, ok := range assignableMap(t, svc, caller) {
			assert.False(t, ok, caller+"/"+name)
		}
	}
}

// The listing flag must equal the enforcement outcome of checkRoleAssignment (no drift).
func TestListRoles_FlagMatchesCheckRoleAssignment(t *testing.T) {
	_, svc := rolesSvc()
	s := svc.(*service)
	for _, caller := range []string{"super_admin", "department_admin", "examiner", "employee", d1Custom, "narrow_custom", "big_custom", "", "no_such_role"} {
		m := assignableMap(t, svc, caller)
		for id, name := range rolesFixture {
			err := s.checkRoleAssignment(context.Background(), id, caller)
			assert.Equal(t, err == nil, m[name], "caller=%q role=%q", caller, name)
		}
	}
}

func TestListRolesHandler_AssignableFlagPassedAndBackwardCompatible(t *testing.T) {
	var gotCaller string
	svc := &mockUserService{listRolesFn: func(_ context.Context, callerRole string) ([]RoleRow, error) {
		gotCaller = callerRole
		return []RoleRow{{ID: "1", Name: "examiner", Assignable: true}, {ID: "2", Name: "super_admin"}}, nil
	}}
	h := NewHandler(svc, nil)
	req := withAuthCtx(httptest.NewRequest(http.MethodGet, "/api/v1/users/roles", nil), "u1", "department_admin", "d1")
	w := httptest.NewRecorder()
	h.ListRoles(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "department_admin", gotCaller)

	data, apiErr := decodeHandlerEnvelope(t, w)
	require.Nil(t, apiErr)
	var rows []map[string]any
	require.NoError(t, json.Unmarshal(data, &rows))
	require.Len(t, rows, 2)
	for _, r := range rows {
		assert.Contains(t, r, "id")
		assert.Contains(t, r, "name")
		assert.Contains(t, r, "assignable")
		assert.NotContains(t, r, "permissions", "other roles' permission lists must not be exposed")
	}
	assert.Equal(t, true, rows[0]["assignable"])
	assert.Equal(t, false, rows[1]["assignable"])
}
