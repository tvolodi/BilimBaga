package users

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISS-217 / FR-BB117 D-1 (Supervisor decision): strict rank hierarchy super_admin >
// department_admin > examiner > employee (builtinRank). A built-in caller may act on a
// built-in target only of STRICTLY LOWER rank; against a custom-role target the target's
// permissions must be a subset of the caller's. A custom-role caller ranks below
// department_admin: never department_admin/super_admin targets, otherwise permission subset.
// Unknown/empty roles fail closed.

const d1Custom = "user_helper" // custom role holding only users:read + users:manage

func d1Perms() map[string][]string {
	return map[string][]string{
		d1Custom:           {"users:read", "users:manage"},
		"department_admin": {"users:read", "users:manage", "questions:read", "exams:read", "exams:assign", "reports:read"},
		"examiner":         {"questions:read", "questions:write", "exams:read", "exams:write", "exams:assign", "reports:read"},
		"employee":         {"portal:read", "portal:submit"},
		"super_admin":      {"users:read", "users:manage", "audit:read"},
		"narrow_custom":    {"users:read"},
		"portal_custom":    {"users:read", "users:manage", "portal:read", "portal:submit"},
		"big_custom":       {"users:read", "audit:read"},
		"ex_custom":        {"users:manage", "questions:read", "questions:write", "exams:read", "exams:write", "exams:assign", "reports:read"},
		"sensitive_custom": {"roles:read"},
	}
}

func d1Setup() (*mockRepo, Service) {
	perms := d1Perms()
	has := func(role, res, act string) bool {
		for _, p := range perms[role] {
			if p == res+":"+act {
				return true
			}
		}
		return false
	}
	repo := newMockRepo()
	repo.users["sa"] = makeUser("sa", "dept-1", "role-sa", "super_admin")
	repo.users["da"] = makeUser("da", "dept-1", "role-da", "department_admin")
	repo.users["ex"] = makeUser("ex", "dept-1", "role-ex", "examiner")
	repo.users["emp"] = makeUser("emp", "dept-1", "role-emp", "employee")
	repo.users["legacy"] = makeUser("legacy", "dept-1", "role-gone", "")
	repo.users["cust"] = makeUser("cust", "dept-1", "role-cust", d1Custom)
	repo.users["narrow"] = makeUser("narrow", "dept-1", "role-narrow", "narrow_custom")
	repo.users["big"] = makeUser("big", "dept-1", "role-big", "big_custom")
	for id, name := range map[string]string{"role-cust": d1Custom, "role-narrow": "narrow_custom", "role-big": "big_custom"} {
		repo.roles[name] = id
		repo.roleByID[id] = name
	}
	svc := WithPermissionsLookup(WithPermissionChecker(NewService(repo), has), func(r string) []string { return perms[r] })
	return repo, svc
}

type d1Call struct {
	name string
	run  func(svc Service, repo *mockRepo, target, caller string) (tempPwd string, err error)
}

func d1Calls() []d1Call {
	ctx := context.Background()
	return []d1Call{
		{"reset", func(svc Service, _ *mockRepo, target, caller string) (string, error) {
			r, err := svc.ResetPassword(ctx, target, caller, "dept-1", "actor", "")
			if r != nil {
				return r.TemporaryPassword, err
			}
			return "", err
		}},
		{"update", func(svc Service, repo *mockRepo, target, caller string) (string, error) {
			// keep the target's own role id so only the target check is exercised
			_, err := svc.UpdateUser(ctx, target, UpdateRequest{FullName: "Renamed", DepartmentID: strPtr("dept-1"), RoleID: repo.users[target].RoleID}, caller, "dept-1", "actor", "")
			return "", err
		}},
		{"deactivate", func(svc Service, _ *mockRepo, target, caller string) (string, error) {
			return "", svc.DeactivateUser(ctx, target, caller, "dept-1", "actor", "")
		}},
		{"unlock", func(svc Service, _ *mockRepo, target, caller string) (string, error) {
			_, err := svc.UnlockUser(ctx, target, caller, "dept-1", "actor", "")
			return "", err
		}},
	}
}

func assertNoWrites(t *testing.T, repo *mockRepo, target string) {
	t.Helper()
	u := repo.users[target]
	assert.False(t, u.ForcePasswordChange, "no password write for %s", target)
	assert.Equal(t, "active", u.Status, "not deactivated: %s", target)
	assert.Empty(t, repo.unlocked)
	assert.Empty(t, repo.revoked)
	assert.Equal(t, "Test "+target, u.FullName, "not renamed: %s", target)
}

func TestD1_ForbiddenTargets(t *testing.T) {
	cases := []struct{ name, caller, target string }{
		{"dept_admin vs super_admin", "department_admin", "sa"},
		{"dept_admin vs peer dept_admin", "department_admin", "da"},
		{"examiner vs dept_admin", "examiner", "da"},
		{"examiner vs peer examiner", "examiner", "ex"},
		{"examiner vs super_admin", "examiner", "sa"},
		{"employee vs employee", "employee", "emp"},
		{"employee vs examiner", "employee", "ex"},
		{"unknown caller role vs employee", "ghost_role", "emp"},
		{"empty caller role vs employee", "", "emp"},
		{"custom holding all examiner perms vs department_admin", "ex_custom", "da"},
		{"custom narrow (users:read) vs examiner (perms not held)", "narrow_custom", "ex"},
		{"custom users:manage vs super_admin", d1Custom, "sa"},
		{"custom users:manage vs department_admin", d1Custom, "da"},
		{"custom users:manage vs examiner", d1Custom, "ex"},
		{"custom users:manage vs employee (portal perms not held)", d1Custom, "emp"},
		{"dept_admin vs custom role with perms it lacks", "department_admin", "big"},
		{"dept_admin vs unknown/legacy empty role", "department_admin", "legacy"},
		{"custom vs unknown/legacy empty role", d1Custom, "legacy"},
	}
	for _, c := range cases {
		for _, call := range d1Calls() {
			t.Run(c.name+"/"+call.name, func(t *testing.T) {
				repo, svc := d1Setup()
				pwd, err := call.run(svc, repo, c.target, c.caller)
				assert.ErrorIs(t, err, ErrForbidden)
				assert.Empty(t, pwd, "no temporary password for a forbidden target")
				assertNoWrites(t, repo, c.target)
			})
		}
	}
}

func TestD1_AllowedTargets(t *testing.T) {
	cases := []struct{ name, caller, target string }{
		{"dept_admin vs employee", "department_admin", "emp"},
		{"dept_admin vs examiner", "department_admin", "ex"},
		{"examiner vs employee", "examiner", "emp"},
		{"custom holding all examiner perms vs examiner (subset)", "ex_custom", "ex"},
		{"dept_admin vs custom subset role", "department_admin", "narrow"},
		{"custom vs same custom role", d1Custom, "cust"},
		{"custom vs narrower custom", d1Custom, "narrow"},
		{"portal custom vs employee (subset)", "portal_custom", "emp"},
		{"super_admin vs super_admin", "super_admin", "sa"},
		{"super_admin vs department_admin", "super_admin", "da"},
		{"super_admin vs legacy role", "super_admin", "legacy"},
	}
	for _, c := range cases {
		for _, call := range d1Calls() {
			t.Run(c.name+"/"+call.name, func(t *testing.T) {
				repo, svc := d1Setup()
				pwd, err := call.run(svc, repo, c.target, c.caller)
				if call.name == "update" && c.target == "legacy" {
					// legacy has an unresolvable role id for the update body; super_admin path only checks no 403
					assert.NotErrorIs(t, err, ErrForbidden)
					return
				}
				require.NoError(t, err)
				if call.name == "reset" {
					assert.Len(t, pwd, 10)
				}
			})
		}
	}
}

func TestD1_OtherDepartmentStillForbiddenEvenWhenSubset(t *testing.T) {
	repo, svc := d1Setup()
	repo.users["emp2"] = makeUser("emp2", "dept-2", "role-emp", "employee")
	_, err := svc.ResetPassword(context.Background(), "emp2", "department_admin", "dept-1", "actor", "")
	assert.ErrorIs(t, err, ErrForbidden)
	assert.False(t, repo.users["emp2"].ForcePasswordChange)
}

func TestD1_SelfServiceStillWorks(t *testing.T) {
	_, svc := d1Setup()
	ctx := context.Background()
	_, err := svc.UpdateUser(ctx, "da", UpdateRequest{FullName: "Me", DepartmentID: strPtr("dept-1"), RoleID: "role-da"}, "department_admin", "dept-1", "da", "")
	assert.NoError(t, err)
	_, err = svc.UpdateUser(ctx, "cust", UpdateRequest{FullName: "Me", DepartmentID: strPtr("dept-1"), RoleID: "role-cust"}, d1Custom, "dept-1", "cust", "")
	assert.NoError(t, err)
	// own role change remains forbidden.
	_, err = svc.UpdateUser(ctx, "da", UpdateRequest{FullName: "Me", DepartmentID: strPtr("dept-1"), RoleID: "role-ex"}, "department_admin", "dept-1", "da", "")
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestD1_UpdateCannotAssignSuperAdminOrMorePowerfulRole(t *testing.T) {
	repo, svc := d1Setup()
	ctx := context.Background()
	_, err := svc.UpdateUser(ctx, "emp", UpdateRequest{FullName: "X", DepartmentID: strPtr("dept-1"), RoleID: "role-sa"}, "department_admin", "dept-1", "actor", "")
	assert.ErrorIs(t, err, ErrForbidden)
	_, err = svc.UpdateUser(ctx, "emp", UpdateRequest{FullName: "X", DepartmentID: strPtr("dept-1"), RoleID: "role-big"}, "department_admin", "dept-1", "actor", "")
	assert.ErrorIs(t, err, ErrForbidden)
	assert.Equal(t, "role-emp", repo.users["emp"].RoleID)
}

// ---- handler level: real service behind the handler ------------------------------

func d1Req(target, callerRole string) *http.Request {
	return withChiID(withAuthCtx(httptest.NewRequest(http.MethodPost, "/x", nil), "actor", callerRole, "dept-1"), target)
}

func TestD1_Handler_ResetPassword_Returns403WithoutTempPassword(t *testing.T) {
	for _, target := range []string{"sa", "da"} {
		repo, svc := d1Setup()
		aw := &fakeAudit{}
		h := &Handler{svc: svc, writer: aw}
		w := httptest.NewRecorder()
		h.ResetPassword(w, d1Req(target, d1Custom))

		assert.Equal(t, http.StatusForbidden, w.Code, target)
		assert.Contains(t, w.Body.String(), `"FORBIDDEN"`)
		assert.NotContains(t, w.Body.String(), "temporary_password")
		assert.Empty(t, aw.actions, "no audit success entry")
		assert.False(t, repo.users[target].ForcePasswordChange)
	}
}

func TestD1_Handler_ResetPassword_EmployeeAllowedForDeptAdmin(t *testing.T) {
	_, svc := d1Setup()
	h := &Handler{svc: svc, writer: &fakeAudit{}}
	w := httptest.NewRecorder()
	h.ResetPassword(w, d1Req("emp", "department_admin"))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "temporary_password")
}

func TestD1_Handler_DeactivateAndUnlock_Return403(t *testing.T) {
	repo, svc := d1Setup()
	h := &Handler{svc: svc, writer: &fakeAudit{}}
	w := httptest.NewRecorder()
	h.DeactivateUser(w, d1Req("sa", "department_admin"))
	assert.Equal(t, http.StatusForbidden, w.Code)
	w = httptest.NewRecorder()
	h.UnlockUser(w, d1Req("sa", "department_admin"))
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "active", repo.users["sa"].Status)
	assert.Empty(t, repo.unlocked)
}

func TestD1_Handler_DeptAdminVsPeerDeptAdmin_Returns403(t *testing.T) {
	repo, svc := d1Setup()
	aw := &fakeAudit{}
	h := &Handler{svc: svc, writer: aw}
	w := httptest.NewRecorder()
	h.ResetPassword(w, d1Req("da", "department_admin"))
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"FORBIDDEN"`)
	assert.NotContains(t, w.Body.String(), "temporary_password")
	assert.Empty(t, aw.actions)
	assert.False(t, repo.users["da"].ForcePasswordChange)
	for _, fn := range []func(http.ResponseWriter, *http.Request){h.DeactivateUser, h.UnlockUser} {
		w = httptest.NewRecorder()
		fn(w, d1Req("da", "department_admin"))
		assert.Equal(t, http.StatusForbidden, w.Code)
	}
	assert.Equal(t, "active", repo.users["da"].Status)
	for _, target := range []string{"ex", "emp"} {
		w = httptest.NewRecorder()
		h.ResetPassword(w, d1Req(target, "department_admin"))
		assert.Equal(t, http.StatusOK, w.Code, target)
	}
}

func TestD1_RankTable(t *testing.T) {
	assert.Equal(t, map[string]int{"employee": 1, "examiner": 2, "department_admin": 3, "super_admin": 4}, builtinRank)
}

// Role assignment hierarchy on create/update.
func TestD1_AssignmentHierarchy(t *testing.T) {
	cases := []struct {
		name, caller, roleID string
		ok                   bool
	}{
		{"dept_admin assigns employee", "department_admin", "role-emp", true},
		{"dept_admin assigns examiner", "department_admin", "role-ex", true},
		{"dept_admin assigns peer department_admin", "department_admin", "role-da", false},
		{"dept_admin assigns super_admin", "department_admin", "role-sa", false},
		{"examiner assigns examiner", "examiner", "role-ex", false},
		{"examiner assigns employee", "examiner", "role-emp", true},
		{"examiner assigns department_admin", "examiner", "role-da", false},
		{"employee assigns employee", "employee", "role-emp", false},
		{"custom assigns department_admin", d1Custom, "role-da", false},
		{"custom assigns examiner (perms not held)", d1Custom, "role-ex", false},
		{"ex_custom assigns examiner (subset)", "ex_custom", "role-ex", true},
		{"dept_admin assigns custom with perms it lacks", "department_admin", "role-big", false},
		{"dept_admin assigns custom subset", "department_admin", "role-narrow", true},
		{"unknown caller assigns employee", "ghost_role", "role-emp", false},
		{"super_admin assigns super_admin", "super_admin", "role-sa", true},
	}
	for _, c := range cases {
		t.Run(c.name+"/create", func(t *testing.T) {
			repo, svc := d1Setup()
			repo.roleByID["role-ex"], repo.roleByID["role-da"] = "examiner", "department_admin"
			repo.roleByID["role-emp"], repo.roleByID["role-sa"] = "employee", "super_admin"
			_, err := svc.CreateUser(context.Background(), CreateRequest{Email: "n@example.com", FullName: "N", DepartmentID: strPtr("dept-1"), RoleID: c.roleID}, c.caller, "dept-1", "actor", "")
			if c.ok {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, ErrForbidden)
			}
		})
	}
}

func TestD1_CustomRoleWithSensitivePermsNotAssignable(t *testing.T) {
	repo, _ := d1Setup()
	repo.roles["sensitive_custom"] = "role-sens"
	repo.roleByID["role-sens"] = "sensitive_custom"
	// even a caller that holds roles:read may not hand it out
	perms := d1Perms()
	perms["department_admin"] = append(perms["department_admin"], "roles:read")
	has := func(role, res, act string) bool {
		for _, p := range perms[role] {
			if p == res+":"+act {
				return true
			}
		}
		return false
	}
	svc := WithPermissionsLookup(WithPermissionChecker(NewService(repo), has), func(r string) []string { return perms[r] })
	_, err := svc.CreateUser(context.Background(), CreateRequest{Email: "n@example.com", FullName: "N", DepartmentID: strPtr("dept-1"), RoleID: "role-sens"}, "department_admin", "dept-1", "actor", "")
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestD1_FailClosedWithoutPermissionLookup(t *testing.T) {
	repo := newMockRepo()
	repo.users["emp"] = makeUser("emp", "dept-1", "role-emp", "employee")
	repo.users["c"] = makeUser("c", "dept-1", "role-c", "some_custom")
	svc := NewService(repo)
	_, err := svc.ResetPassword(context.Background(), "emp", "some_custom", "dept-1", "actor", "")
	assert.ErrorIs(t, err, ErrForbidden)
	_, err = svc.ResetPassword(context.Background(), "c", "department_admin", "dept-1", "actor", "")
	assert.ErrorIs(t, err, ErrForbidden)
	// built-in vs built-in lower rank needs no lookup
	_, err = svc.ResetPassword(context.Background(), "emp", "department_admin", "dept-1", "actor", "")
	assert.NoError(t, err)
}

func TestD1_SelfServiceUnchanged(t *testing.T) {
	_, svc := d1Setup()
	ctx := context.Background()
	for _, c := range []struct{ id, role, roleID string }{{"da", "department_admin", "role-da"}, {"ex", "examiner", "role-ex"}, {"emp", "employee", "role-emp"}} {
		_, err := svc.UpdateUser(ctx, c.id, UpdateRequest{FullName: "Me", DepartmentID: strPtr("dept-1"), RoleID: c.roleID}, c.role, "dept-1", c.id, "")
		assert.NoError(t, err, c.id)
	}
	// a stale token role does not unlock self-service on a different stored role
	_, err := svc.ResetPassword(ctx, "da", "examiner", "dept-1", "da", "")
	assert.ErrorIs(t, err, ErrForbidden)
}
