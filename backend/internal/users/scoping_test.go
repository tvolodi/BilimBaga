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

// FR-BB117: a custom role (any name other than super_admin) holding users:read /
// users:manage must be confined to its own department -- never org-wide.

const customRole = "qa_lead"

func seedTwoDepts(repo *mockRepo) {
	repo.users["u1"] = makeUser("u1", "dept-1", "role-emp", "employee")
	repo.users["u2"] = makeUser("u2", "dept-1", "role-emp", "employee")
	repo.users["u3"] = makeUser("u3", "dept-2", "role-emp", "employee")
}

func TestScoping_ListUsers_CustomRoleIsDepartmentScoped(t *testing.T) {
	repo := newMockRepo()
	seedTwoDepts(repo)
	res, err := NewService(repo).ListUsers(context.Background(), customRole, "dept-1", ListFilters{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, 2, res.Meta.Total)
	for _, u := range res.Items {
		assert.Equal(t, "dept-1", *u.DepartmentID)
	}
}

func TestScoping_ListUsers_NoDepartmentSeesNothing(t *testing.T) {
	repo := newMockRepo()
	seedTwoDepts(repo)
	for _, role := range []string{customRole, "department_admin", "", "employee"} {
		res, err := NewService(repo).ListUsers(context.Background(), role, "", ListFilters{Page: 1, PerPage: 20})
		require.NoError(t, err, role)
		assert.Equal(t, 0, res.Meta.Total, role)
		assert.Empty(t, res.Items, role)
	}
}

func TestScoping_ListUsers_SuperAdminOrgWide(t *testing.T) {
	repo := newMockRepo()
	seedTwoDepts(repo)
	res, err := NewService(repo).ListUsers(context.Background(), "super_admin", "", ListFilters{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, 3, res.Meta.Total)
}

func TestScoping_GetUser_CustomRole(t *testing.T) {
	repo := newMockRepo()
	seedTwoDepts(repo)
	perm := func(role, res, act string) bool { return role == customRole && res == "users" && act == "read" }
	svc := WithPermissionChecker(NewService(repo), perm)

	_, err := svc.GetUser(context.Background(), "u1", customRole, "caller", "dept-1")
	assert.NoError(t, err, "same dept allowed with users:read")
	_, err = svc.GetUser(context.Background(), "u3", customRole, "caller", "dept-1")
	assert.ErrorIs(t, err, ErrForbidden, "other dept denied")
	_, err = svc.GetUser(context.Background(), "u3", customRole, "u3", "dept-1")
	assert.NoError(t, err, "self allowed")
	_, err = svc.GetUser(context.Background(), "u1", customRole, "caller", "")
	assert.ErrorIs(t, err, ErrForbidden, "no department => no scope")
}

func TestScoping_GetUser_CustomRoleWithoutUsersReadOrChecker(t *testing.T) {
	repo := newMockRepo()
	seedTwoDepts(repo)
	// No checker attached: default-deny, only self.
	_, err := NewService(repo).GetUser(context.Background(), "u1", customRole, "caller", "dept-1")
	assert.ErrorIs(t, err, ErrForbidden)
	// Checker says no users:read.
	svc := WithPermissionChecker(NewService(repo), func(string, string, string) bool { return false })
	_, err = svc.GetUser(context.Background(), "u1", customRole, "caller", "dept-1")
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestScoping_Mutations_CustomRoleOtherDeptForbidden(t *testing.T) {
	repo := newMockRepo()
	seedTwoDepts(repo)
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.UpdateUser(ctx, "u3", UpdateRequest{FullName: "X", DepartmentID: strPtr("dept-1"), RoleID: "role-emp"}, customRole, "dept-1", "c", "")
	assert.ErrorIs(t, err, ErrForbidden)
	assert.ErrorIs(t, svc.DeactivateUser(ctx, "u3", customRole, "dept-1", "c", ""), ErrForbidden)
	_, err = svc.ResetPassword(ctx, "u3", customRole, "dept-1", "c", "")
	assert.ErrorIs(t, err, ErrForbidden)
	_, err = svc.UnlockUser(ctx, "u3", customRole, "dept-1", "c", "")
	assert.ErrorIs(t, err, ErrForbidden)
	_, err = svc.CreateUser(ctx, CreateRequest{Email: "n@example.com", FullName: "N", DepartmentID: strPtr("dept-2"), RoleID: "role-emp"}, customRole, "dept-1", "c", "")
	assert.ErrorIs(t, err, ErrForbidden)
	// A caller with no department cannot create department-less users either.
	_, err = svc.CreateUser(ctx, CreateRequest{Email: "n@example.com", FullName: "N", RoleID: "role-emp"}, customRole, "", "c", "")
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestScoping_UpdateUser_CustomRoleCannotMoveUserOutOfDepartment(t *testing.T) {
	repo := newMockRepo()
	seedTwoDepts(repo)
	_, err := NewService(repo).UpdateUser(context.Background(), "u1",
		UpdateRequest{FullName: "X", DepartmentID: strPtr("dept-2"), RoleID: "role-emp"}, customRole, "dept-1", "c", "")
	assert.ErrorIs(t, err, ErrForbidden)
	assert.Equal(t, "dept-1", *repo.users["u1"].DepartmentID)
}

func TestScoping_UpdateUser_CustomRoleOwnDeptOK(t *testing.T) {
	repo := newMockRepo()
	seedTwoDepts(repo)
	_, err := NewService(repo).UpdateUser(context.Background(), "u1",
		UpdateRequest{FullName: "X", DepartmentID: strPtr("dept-1"), RoleID: "role-emp"}, customRole, "dept-1", "c", "")
	assert.NoError(t, err)
}

func TestScoping_CustomRoleCannotAssignSuperAdmin(t *testing.T) {
	repo := newMockRepo()
	seedTwoDepts(repo)
	_, err := NewService(repo).UpdateUser(context.Background(), "u1",
		UpdateRequest{FullName: "X", DepartmentID: strPtr("dept-1"), RoleID: "role-sa"}, customRole, "dept-1", "c", "")
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestScoping_Import_ScopedAndCannotImportSuperAdmin(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	rows := []CSVRow{
		{RowNum: 1, Email: "a@example.com", FullName: "A", DepartmentName: "Engineering", RoleName: "employee"},
		{RowNum: 2, Email: "b@example.com", FullName: "B", DepartmentName: "Sales", RoleName: "employee"},
		{RowNum: 3, Email: "c@example.com", FullName: "C", DepartmentName: "Engineering", RoleName: "super_admin"},
	}
	for _, role := range []string{customRole, "department_admin"} {
		prev, err := svc.ImportUsers(context.Background(), rows, false, role, "dept-1", "c", "")
		require.NoError(t, err)
		assert.Len(t, prev.Valid, 1, role)
		assert.Len(t, prev.Errors, 2, role)
	}
}

func TestHandlerGetMe_IncludesPermissions(t *testing.T) {
	svc := &mockUserService{getMeFn: func(_ context.Context, _ string) (*User, error) {
		u := sampleUser("u1")
		u.RoleName = customRole
		return u, nil
	}}
	h := NewHandler(svc, nil).WithPermissionsProvider(func(role string) []string {
		if role == customRole {
			return []string{"exams:read", "users:read"}
		}
		return nil
	})
	req := withAuthCtx(httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil), "u1", customRole, "dept-1")
	w := httptest.NewRecorder()
	h.GetMe(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var env struct {
		Data struct {
			Email       string   `json:"email"`
			RoleName    string   `json:"role_name"`
			Permissions []string `json:"permissions"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	assert.Equal(t, "u1@example.com", env.Data.Email)
	assert.Equal(t, customRole, env.Data.RoleName)
	assert.Equal(t, []string{"exams:read", "users:read"}, env.Data.Permissions)
}

func TestHandlerGetMe_NoProviderReturnsEmptyPermissionsArray(t *testing.T) {
	svc := &mockUserService{getMeFn: func(_ context.Context, _ string) (*User, error) { return sampleUser("u1"), nil }}
	h := NewHandler(svc, nil)
	w := httptest.NewRecorder()
	h.GetMe(w, withAuthCtx(httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil), "u1", "employee", "dept-1"))
	assert.Contains(t, w.Body.String(), `"permissions":[]`)
}

func TestScoping_CustomRoleCannotAssignMorePowerfulRole(t *testing.T) {
	repo := newMockRepo()
	seedTwoDepts(repo)
	perms := map[string][]string{
		customRole: {"users:read", "users:manage", "portal:read"},
		"department_admin": {"users:read", "reports:read"},
		"examiner": {"exams:read", "questions:write"},
		"employee": {"portal:read"},
	}
	has := func(role, res, act string) bool {
		for _, p := range perms[role] {
			if p == res+":"+act {
				return true
			}
		}
		return false
	}
	svc := WithPermissionsLookup(WithPermissionChecker(NewService(repo), has), func(r string) []string { return perms[r] })
	ctx := context.Background()
	upd := func(roleID string) error {
		_, err := svc.UpdateUser(ctx, "u1", UpdateRequest{FullName: "X", DepartmentID: strPtr("dept-1"), RoleID: roleID}, customRole, "dept-1", "c", "")
		return err
	}
	assert.ErrorIs(t, upd("role-ex"), ErrForbidden, "examiner exceeds caller permissions")
	assert.ErrorIs(t, upd("role-da"), ErrForbidden, "department_admin has reports:read the caller lacks")
	assert.NoError(t, upd("role-emp"), "employee perms are a subset of the caller's")
}
