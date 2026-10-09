package users

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISS-217 / FR-BB117 D-1: a non-super_admin caller may act on a target only if the
// target's role is not super_admin and its permissions are a subset of the caller's
// (built-in caller vs built-in target keeps the historical behaviour: only super_admin
// is off limits, because the seeded department_admin set is not a strict superset of
// examiner/employee).

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
		{"dept_admin vs dept_admin peer", "department_admin", "da"},
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
